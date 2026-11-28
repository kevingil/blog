#!/usr/bin/env bash
# Boot Postgres and MinIO the way the test profile does, start blog-backend,
# and run the MCP demo in scripts/mcp-demo-prompt.txt.
#
# MCP_DEMO_MODEL must be provider/model. The script installs OpenCode CLI
# 2.4.2 when that version is not already available, writes a project-scoped
# config, starts a private `opencode serve` in that workspace, and runs
# `opencode run --model`. MCP_DEMO_CLIENT=1 skips OpenCode and uses the
# scripted client in scripts/mcp-demo.mjs. Either way the script
# checks the rows through the REST API.
#
#   MCP_DEMO_MODEL=openai/gpt-6.1-sol ./scripts/mcp-demo.sh
#   MCP_DEMO_MODEL=anthropic/claude-sonnet-5-5 ./scripts/mcp-demo.sh
#   MCP_DEMO_CLIENT=1 ./scripts/mcp-demo.sh
#
# Artifacts go to tmp/mcp-demo, or to MCP_DEMO_ARTIFACT_DIR.
# MCP_DEMO_NO_RECORD=1 skips the asciinema cast.
set -euo pipefail

# https://opencode.ai/docs/mcp-servers/ and https://opencode.ai/docs/cli/commands/
# for this CLI. Remote servers live under mcp.servers, and codemode defaults on.
OPENCODE_VERSION="2.4.2"

use_client=0
model="${MCP_DEMO_MODEL:-}"
if [ -n "${MCP_DEMO_CLIENT:-}" ]; then
  use_client=1
elif [ -z "$model" ]; then
  echo "MCP_DEMO_MODEL is not set. Pass a model as provider/model, or set MCP_DEMO_CLIENT=1 to run the scripted client." >&2
  exit 1
elif [ -z "${model%%/*}" ] || [ -z "${model#*/}" ] || [ "$model" = "${model#*/}" ]; then
  echo "MCP_DEMO_MODEL must be provider/model, got: $model" >&2
  exit 1
fi

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

opencode_version_matches() {
  local bin="$1"
  local got
  got="$("$bin" --version 2>/dev/null || true)"
  [ "$got" = "opencode v${OPENCODE_VERSION}" ]
}

ensure_opencode() {
  local prefix="${HOME}/.cache/mcp-demo-opencode/prefix"
  local bin="$prefix/node_modules/.bin/opencode"
  if command -v opencode >/dev/null 2>&1 && opencode_version_matches "$(command -v opencode)"; then
    OPENCODE_BIN="$(command -v opencode)"
    return 0
  fi
  if [ -x "$bin" ] && opencode_version_matches "$bin"; then
    OPENCODE_BIN="$bin"
    return 0
  fi
  if ! command -v npm >/dev/null 2>&1; then
    echo "npm is required to install OpenCode CLI ${OPENCODE_VERSION}" >&2
    exit 1
  fi
  echo "installing OpenCode CLI ${OPENCODE_VERSION}"
  mkdir -p "$prefix"
  npm install --prefix "$prefix" "@opencode/cli@${OPENCODE_VERSION}"
  if [ ! -x "$bin" ] || ! opencode_version_matches "$bin"; then
    echo "OpenCode CLI ${OPENCODE_VERSION} did not install (got: $("$bin" --version 2>/dev/null || echo missing))" >&2
    exit 1
  fi
  OPENCODE_BIN="$bin"
}

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

http POST /auth/mcp-tokens '{"name":"opencode-demo"}' "$jwt"
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
  echo "MCP_DEMO_CLIENT is set. Skipping OpenCode and using the scripted MCP client."
  trace="$artifact_dir/mcp-client-log.json"
  node "$root/scripts/mcp-demo.mjs" client \
    --base "$base" \
    --token "$mcp_token" \
    --prompt "$prompt" \
    --log "$trace" || driver_status=$?
  mode="client"
else
  ensure_opencode
  opencode_home="${HOME}/.cache/mcp-demo-opencode"
  workspace="$opencode_home/workspace"
  rm -rf "$workspace"
  mkdir -p "$workspace"
  # shellcheck disable=SC2016 # node source, not a shell expansion
  node -e '
const fs = require("node:fs");
const config = {
  $schema: "https://opencode.ai/config.json",
  mcp: {
    servers: {
      blog: {
        type: "remote",
        url: process.argv[1],
        oauth: false,
        headers: { Authorization: "Bearer {env:BLOG_MCP_TOKEN}" },
        codemode: false,
        protocol: "legacy",
      },
    },
  },
};
fs.writeFileSync(process.argv[2], JSON.stringify(config, null, 2) + "\n");
' "$base/mcp" "$workspace/opencode.json"
  echo "registering the blog MCP server in $workspace/opencode.json"
  # `opencode mcp list --standalone` returns before this CLI applies project
  # config, so keep one private server up and list it once the connection settles.
  opencode_password="$(node -e 'process.stdout.write(require("crypto").randomBytes(24).toString("base64url"))')"
  opencode_log="$artifact_dir/opencode-serve.log"
  : >"$opencode_log"
  (
    cd "$workspace"
    BLOG_MCP_TOKEN="$mcp_token" \
      OPENCODE_PASSWORD="$opencode_password" \
      OPENCODE_SERVER_PASSWORD="$opencode_password" \
      "$OPENCODE_BIN" serve --hostname 127.0.0.1 --port 0
  ) >"$opencode_log" 2>&1 &
  pids+=("$!")
  opencode_url=""
  for _ in $(seq 1 50); do
    opencode_url="$(sed -n 's/^server listening on //p' "$opencode_log")"
    opencode_url="${opencode_url%%$'\n'*}"
    if [ -n "$opencode_url" ]; then
      break
    fi
    sleep 0.1
  done
  if [ -z "$opencode_url" ]; then
    echo "OpenCode serve did not start" >&2
    exit 1
  fi
  listed=""
  for _ in $(seq 1 40); do
    listed="$(
      cd "$workspace"
      OPENCODE_PASSWORD="$opencode_password" \
        OPENCODE_SERVER_PASSWORD="$opencode_password" \
        "$OPENCODE_BIN" mcp list --server "$opencode_url" 2>>"$opencode_log" || true
    )"
    if printf '%s\n' "$listed" | grep -q 'blog.*connected'; then
      break
    fi
    sleep 0.25
  done
  printf '%s\n' "$listed"
  if ! printf '%s\n' "$listed" | grep -q 'blog.*connected'; then
    echo "OpenCode did not connect the blog MCP server" >&2
    exit 1
  fi
  events="$artifact_dir/opencode-events.jsonl"
  : >"$events"
  if grep -q '```' "$prompt"; then
    echo "demo prompt must not contain code snippets" >&2
    exit 1
  fi
  echo "running OpenCode with model $model"
  # --auto answers permission prompts so the headless run can call the server.
  # --server uses the private serve started above, which has the generated
  # project config and BLOG_MCP_TOKEN, and does not write the user global config.
  set +e
  set -o pipefail
  (
    cd "$workspace"
    OPENCODE_PASSWORD="$opencode_password" \
      OPENCODE_SERVER_PASSWORD="$opencode_password" \
      "$OPENCODE_BIN" run --server "$opencode_url" --auto --archived --format json --model "$model" \
      "$(cat "$prompt")"
  ) < /dev/null \
    2> >(sed 's/^/[opencode] /' >&2) \
    | tee "$events" \
    | node "$root/scripts/mcp-demo.mjs" follow
  driver_status=${PIPESTATUS[0]}
  set +o pipefail
  set -e
  trace="$events"
  mode="opencode"
  if [ "$driver_status" -ne 0 ]; then
    echo "OpenCode exited $driver_status" >&2
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
if [ "$mode" = "opencode" ]; then
  echo "opencode events: $artifact_dir/opencode-events.jsonl"
else
  echo "client log: $trace"
fi
