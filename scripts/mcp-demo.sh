#!/usr/bin/env bash
# Boot Postgres and MinIO the way the test profile does, start blog-backend,
# and run the MCP demo in scripts/mcp-demo-prompt.txt.
#
# With OPENAI_API_KEY set (and not the local fixture placeholder), Codex
# (`@openai/codex`, `codex exec`) drives the server. Otherwise a small MCP
# client in scripts/mcp-demo.mjs makes the same tool calls. Either way the
# script checks the rows through the REST API.
#
#   ./scripts/mcp-demo.sh
#   MCP_DEMO_CLIENT=1 ./scripts/mcp-demo.sh
#
# Artifacts go to tmp/mcp-demo, or to MCP_DEMO_ARTIFACT_DIR.
# MCP_DEMO_NO_RECORD=1 skips the asciinema cast.
set -euo pipefail

render_cast() {
  local cast="$1" gif="$2" mp4="$3"
  local agg_bin=""
  if command -v agg >/dev/null 2>&1; then
    agg_bin="$(command -v agg)"
  elif [ -x /tmp/agg ]; then
    agg_bin=/tmp/agg
  fi
  if [ -z "$agg_bin" ]; then
    echo "agg is not installed; left the cast at $cast"
    return 0
  fi
  echo "rendering $gif"
  "$agg_bin" --speed 4 --font-size 14 --idle-time-limit 2 "$cast" "$gif" || {
    echo "agg failed; left the cast at $cast"
    return 0
  }
  if command -v ffmpeg >/dev/null 2>&1; then
    ffmpeg -y -loglevel error -i "$gif" -movflags +faststart -pix_fmt yuv420p "$mp4" || true
  fi
}

root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
cd "$root"

artifact_dir="${MCP_DEMO_ARTIFACT_DIR:-$root/tmp/mcp-demo}"
mkdir -p "$artifact_dir"
prompt="$root/scripts/mcp-demo-prompt.txt"
base="http://127.0.0.1:8080"
demo_email="mcp-demo@example.com"
demo_password="McpDemoPass123!"
demo_name="MCP Demo"

if [ -z "${MCP_DEMO_INNER:-}" ] && [ -z "${MCP_DEMO_NO_RECORD:-}" ] && command -v asciinema >/dev/null 2>&1; then
  export MCP_DEMO_INNER=1
  cast="$artifact_dir/mcp-demo.cast"
  rm -f "$cast"
  set +e
  asciinema rec --overwrite -y -q --cols 110 --rows 32 -c "$0" "$cast"
  status=$?
  set -e
  if [ -s "$cast" ]; then
    render_cast "$cast" "$artifact_dir/mcp-demo.gif" "$artifact_dir/mcp-demo.mp4"
    if [ -f "$artifact_dir/status" ]; then
      exit "$(cat "$artifact_dir/status")"
    fi
    exit "$status"
  fi
  echo "asciinema did not write a cast; continuing without one"
fi

exec > >(tee "$artifact_dir/mcp-demo.log") 2>&1
trap 'printf "%s\n" "$?" > "$artifact_dir/status"' EXIT

if [ -d "${HOME}/.local/bin" ]; then
  export PATH="${HOME}/.local/bin:${PATH}"
fi

real_openai_key="${OPENAI_API_KEY:-}"
if [ "$real_openai_key" = "local-fixture-key" ]; then
  real_openai_key=""
fi

use_client=0
if [ -n "${MCP_DEMO_CLIENT:-}" ] || [ -z "$real_openai_key" ]; then
  use_client=1
fi

compose() {
  if docker info >/dev/null 2>&1; then
    docker compose "$@"
  else
    sudo docker compose "$@"
  fi
}

echo "starting test-profile Postgres and MinIO"
compose --profile test up -d test-db object-storage object-storage-init
init_ok=0
for _ in $(seq 1 60); do
  line="$(compose --profile test ps -a object-storage-init --format '{{.State}} {{.ExitCode}}' 2>/dev/null || true)"
  case "$line" in
    exited\ 0*|Exited\ 0*) init_ok=1; break ;;
    exited\ *|Exited\ *) echo "object-storage-init failed: $line" >&2; exit 1 ;;
  esac
  sleep 2
done
if [ "$init_ok" -ne 1 ]; then
  echo "object-storage-init did not finish: ${line:-no container}" >&2
  exit 1
fi

echo "waiting for Postgres"
ready=0
for _ in $(seq 1 60); do
  if compose --profile test exec -T test-db pg_isready -U blog -d blog_test >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done
if [ "$ready" -ne 1 ]; then
  echo "Postgres did not become ready" >&2
  exit 1
fi

echo "recreating blog_mcp_demo"
compose --profile test exec -T test-db psql -U blog -d blog_test -v ON_ERROR_STOP=1 -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'blog_mcp_demo' AND pid <> pg_backend_pid();" \
  >/dev/null
compose --profile test exec -T test-db psql -U blog -d blog_test -v ON_ERROR_STOP=1 -c \
  "DROP DATABASE IF EXISTS blog_mcp_demo;"
compose --profile test exec -T test-db psql -U blog -d blog_test -v ON_ERROR_STOP=1 -c \
  "CREATE DATABASE blog_mcp_demo;"

if ! command -v cargo >/dev/null 2>&1; then
  echo "cargo is required to build blog-backend" >&2
  exit 1
fi
if ! pkg-config --exists libpq 2>/dev/null; then
  echo "libpq development files are required (pkg-config libpq)" >&2
  exit 1
fi

echo "building blog-backend, migrate, and external-fixtures"
# rust-toolchain.toml lives in backend/, and rustup only discovers it from cwd.
(
  cd "$root/backend"
  cargo build --locked --bin blog-backend --bin migrate --bin external-fixtures
)

bin_dir="$root/backend/target/debug"
database_url="postgres://blog:blog@127.0.0.1:55433/blog_mcp_demo"
echo "applying migrations"
DATABASE_URL="$database_url" "$bin_dir/migrate"

pids=()
cleanup() {
  local pid
  for pid in "${pids[@]+"${pids[@]}"}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done
}
trap 'code=$?; printf "%s\n" "$code" > "$artifact_dir/status"; cleanup' EXIT

echo "starting external-fixtures"
"$bin_dir/external-fixtures" >"$artifact_dir/fixtures.log" 2>&1 &
pids+=("$!")

echo "starting blog-backend"
(
  export DATABASE_URL="$database_url"
  export AUTH_SECRET="local-test-secret-that-is-not-used-in-production"
  export HOST="127.0.0.1"
  export PORT="8080"
  export ALLOWED_ORIGINS="http://localhost:3000"
  export OPENAI_API_KEY="local-fixture-key"
  export OPENAI_BASE_URL="http://127.0.0.1:8090/v1"
  export GROQ_API_KEY="local-fixture-key"
  export GROQ_BASE_URL="http://127.0.0.1:8090/v1"
  export EXA_API_KEY="local-fixture-key"
  export EXA_BASE_URL="http://127.0.0.1:8090"
  export PUBLIC_API_URL="http://127.0.0.1:8080"
  export PUBLIC_APP_URL="http://localhost:3000"
  export S3_ENDPOINT="http://127.0.0.1:9000"
  export S3_ACCESS_KEY_ID="blog"
  export S3_ACCESS_KEY_SECRET="blog-local-secret"
  export S3_BUCKET="blog"
  export S3_URL_PREFIX="http://127.0.0.1:9000/blog"
  export RUST_LOG="info"
  exec "$bin_dir/blog-backend"
) >"$artifact_dir/backend.log" 2>&1 &
pids+=("$!")
backend_pid="${pids[${#pids[@]}-1]}"

wait_http() {
  local url="$1"
  local pid="$2"
  local i
  for i in $(seq 1 60); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    if [ -n "$pid" ] && ! kill -0 "$pid" 2>/dev/null; then
      return 1
    fi
    sleep 1
  done
  return 1
}

if ! wait_http "http://127.0.0.1:8090/__fixture/requests" "${pids[0]}"; then
  echo "external-fixtures did not stay up" >&2
  tail -n 40 "$artifact_dir/fixtures.log" >&2 || true
  exit 1
fi
echo "fixtures ready"

echo "waiting for blog-backend (the binary is already built)"
ready=0
for i in $(seq 1 90); do
  if curl -fsS "$base/health" >/dev/null 2>&1; then
    ready=1
    break
  fi
  if ! kill -0 "$backend_pid" 2>/dev/null; then
    echo "blog-backend exited" >&2
    tail -n 60 "$artifact_dir/backend.log" >&2 || true
    exit 1
  fi
  sleep 1
done
if [ "$ready" -ne 1 ]; then
  echo "blog-backend did not become healthy" >&2
  tail -n 60 "$artifact_dir/backend.log" >&2 || true
  exit 1
fi
echo "backend healthy"

http() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local auth="${4:-}"
  local tmp code
  tmp="$(mktemp)"
  local args=(-sS -o "$tmp" -w "%{http_code}" -X "$method" "$base$path" -H "accept: application/json")
  if [ -n "$auth" ]; then
    args+=(-H "authorization: Bearer $auth")
  fi
  if [ -n "$body" ]; then
    args+=(-H "content-type: application/json" --data "$body")
  fi
  code="$(curl "${args[@]}")"
  HTTP_CODE="$code"
  HTTP_BODY="$(cat "$tmp")"
  rm -f "$tmp"
}

echo "registering $demo_email"
http POST /auth/register "{\"name\":\"$demo_name\",\"email\":\"$demo_email\",\"password\":\"$demo_password\"}"
if [ "$HTTP_CODE" -ne 200 ] && [ "$HTTP_CODE" -ne 201 ] && [ "$HTTP_CODE" -ne 409 ]; then
  echo "register failed with HTTP $HTTP_CODE" >&2
  printf '%s\n' "$HTTP_BODY" | head -c 400 >&2
  echo >&2
  exit 1
fi

http POST /auth/login "{\"email\":\"$demo_email\",\"password\":\"$demo_password\"}"
if [ "$HTTP_CODE" -ne 200 ]; then
  echo "login failed with HTTP $HTTP_CODE" >&2
  printf '%s\n' "$HTTP_BODY" | head -c 400 >&2
  echo >&2
  exit 1
fi
jwt="$(node -e 'const body=JSON.parse(process.argv[1]); if (body.data.user.role !== "admin") { console.error("account role is " + body.data.user.role); process.exit(1); } process.stdout.write(body.data.token);' "$HTTP_BODY")"
unset HTTP_BODY
echo "logged in as admin"

http POST /auth/mcp-tokens '{"name":"codex-demo"}' "$jwt"
if [ "$HTTP_CODE" -ne 200 ] && [ "$HTTP_CODE" -ne 201 ]; then
  echo "mcp token mint failed with HTTP $HTTP_CODE" >&2
  printf '%s\n' "$HTTP_BODY" | head -c 400 >&2
  echo >&2
  exit 1
fi
mcp_token="$(node -e 'process.stdout.write(JSON.parse(process.argv[1]).data.token)' "$HTTP_BODY")"
unset HTTP_BODY
case "$mcp_token" in
  mcp_*) echo "minted mcp token (${#mcp_token} characters)" ;;
  *) echo "minted token did not use the mcp_ prefix" >&2; exit 1 ;;
esac

trace=""
driver_status=0
if [ "$use_client" -eq 1 ]; then
  if [ -z "$real_openai_key" ]; then
    echo "OPENAI_API_KEY is not set. Codex was not run. A scripted MCP client will cover the same outcomes. No Codex transcript will be written."
  else
    echo "MCP_DEMO_CLIENT is set. Skipping Codex and using the scripted MCP client."
  fi
  trace="$artifact_dir/mcp-client-log.json"
  node "$root/scripts/mcp-demo.mjs" client \
    --base "$base" \
    --token "$mcp_token" \
    --prompt "$prompt" \
    --log "$trace" || driver_status=$?
  mode="client"
else
  if ! command -v codex >/dev/null 2>&1; then
    echo "codex is not on PATH. Install @openai/codex or set MCP_DEMO_CLIENT=1." >&2
    exit 1
  fi
  codex_home="${MCP_DEMO_CODEX_HOME:-${HOME}/.cache/mcp-demo-codex}"
  rm -rf "$codex_home"
  mkdir -p "$codex_home/workspace"
  echo "registering Streamable HTTP MCP server with Codex"
  env CODEX_HOME="$codex_home" codex mcp add blog \
    --url "$base/mcp" \
    --bearer-token-env-var BLOG_MCP_TOKEN
  env CODEX_HOME="$codex_home" codex mcp list
  events="$artifact_dir/codex-events.jsonl"
  : >"$events"
  if grep -q '```' "$prompt"; then
    echo "demo prompt must not contain code snippets" >&2
    exit 1
  fi
  echo "running Codex (OpenAI key length ${#real_openai_key})"
  # exec cannot prompt, and approval_policy=never denies MCP calls outright.
  # Bypass is what lets the non-interactive run call the local server.
  # A comment cannot sit inside this continuation: it would drop the
  # assignments, and Codex authenticates the Responses API with
  # CODEX_API_KEY rather than OPENAI_API_KEY.
  set +e
  set -o pipefail
  env \
    -u OPENAI_BASE_URL \
    OPENAI_API_KEY="$real_openai_key" \
    CODEX_API_KEY="$real_openai_key" \
    BLOG_MCP_TOKEN="$mcp_token" \
    CODEX_HOME="$codex_home" \
    codex exec --json --color never --skip-git-repo-check --trust --ephemeral \
      --dangerously-bypass-approvals-and-sandbox \
      -C "$codex_home/workspace" \
      "$(cat "$prompt")" \
    < /dev/null \
    2> >(sed 's/^/[codex] /' >&2) \
    | tee "$events" \
    | node "$root/scripts/mcp-demo.mjs" follow
  driver_status=${PIPESTATUS[0]}
  set +o pipefail
  set -e
  trace="$events"
  mode="codex"
  if [ "$driver_status" -ne 0 ]; then
    echo "Codex exited $driver_status" >&2
  fi
fi

verify_status=0
if [ -n "$trace" ] && [ -s "$trace" ]; then
  node "$root/scripts/mcp-demo.mjs" verify \
    --base "$base" \
    --token "$jwt" \
    --trace "$trace" \
    --mode "$mode" || verify_status=$?
else
  echo "no MCP trace to verify" >&2
  verify_status=1
fi

if [ "$driver_status" -ne 0 ] || [ "$verify_status" -ne 0 ]; then
  echo "demo failed (driver $driver_status, verify $verify_status)" >&2
  exit 1
fi

echo "demo passed"
echo "log: $artifact_dir/mcp-demo.log"
if [ "$mode" = "codex" ]; then
  echo "codex events: $artifact_dir/codex-events.jsonl"
else
  echo "client log: $trace"
fi
