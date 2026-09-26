#!/usr/bin/env bash
# Probe the local Blog Copilot stack. Exits 0 only when API, frontend,
# Postgres, and object storage all answer. Safe to re-run.
set -euo pipefail

cd "$(dirname "$0")/.."

SUDO=""
if [ "$(id -u)" -ne 0 ]; then SUDO="sudo"; fi

fail() {
  echo "verify.sh: $*" >&2
  exit 1
}

echo "Checking API health at http://localhost:8080/health..."
curl -fsS http://localhost:8080/health >/dev/null || fail "API health check failed"

echo "Checking frontend at http://localhost:3000/..."
curl -fsS -o /dev/null http://localhost:3000/ || fail "frontend did not respond"

echo "Checking Postgres on localhost:55432..."
$SUDO docker compose exec -T db pg_isready -U blog -d blog >/dev/null \
  || fail "Postgres is not ready (expected blog/blog on localhost:55432)"

echo "Checking object storage at http://localhost:9000/minio/health/live..."
curl -fsS http://localhost:9000/minio/health/live >/dev/null \
  || fail "MinIO health check failed"

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
