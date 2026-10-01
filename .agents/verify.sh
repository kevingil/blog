#!/usr/bin/env bash
# Probe the local Blog Copilot stack. Exits 0 only when API, frontend,
# Postgres, and object storage all answer. Safe to re-run.
set -euo pipefail

cd "$(dirname "$0")/.."

SUDO=""
if [ "$(id -u)" -ne 0 ]; then SUDO="sudo"; fi

wait_for() {
  local name="$1"
  local attempts="${2:-30}"
  shift 2
  local i=0
  while [ "$i" -lt "$attempts" ]; do
    if "$@" >/dev/null 2>&1; then
      echo "ok  $name"
      return 0
    fi
    i=$((i + 1))
    sleep 1
  done
  echo "verify.sh: $name did not become ready" >&2
  return 1
}

wait_for "API http://localhost:8080/health" 30 \
  curl -fsS http://localhost:8080/health

wait_for "frontend http://localhost:3000/" 30 \
  curl -fsS -o /dev/null http://localhost:3000/

wait_for "Postgres localhost:55432" 30 \
  $SUDO docker compose exec -T db pg_isready -U blog -d blog

wait_for "MinIO http://localhost:9000/minio/health/live" 30 \
  curl -fsS http://localhost:9000/minio/health/live

echo
echo "Stack is healthy."
echo "  Frontend:           http://localhost:3000"
echo "  API:                http://localhost:8080"
echo "  Health:             http://localhost:8080/health"
echo "  Swagger UI:         http://localhost:8080/swagger"
echo "  Postgres:           postgres://blog:blog@localhost:55432/blog"
echo "  Object storage:     http://localhost:9000"
echo "  MinIO console:      http://localhost:9001"
echo
echo "Host-side tests (once rustc 1.92 and libpq are available):"
echo "  TEST_DATABASE_URL=postgres://blog:blog@localhost:55432/blog \\"
echo "  TEST_S3_ENDPOINT=http://localhost:9000 \\"
echo "    ./scripts/test-rust.sh blocking"
