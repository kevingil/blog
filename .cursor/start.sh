#!/usr/bin/env bash
# Per-boot startup for the Blog Copilot Docker Compose stack.
# Ensures the nested Docker daemon is running, then brings the full stack up.
set -euo pipefail

cd "$(dirname "$0")/.."

ensure_dockerd() {
  if sudo docker info >/dev/null 2>&1; then
    echo "dockerd already running."
    return
  fi
  echo "Starting dockerd..."
  sudo bash -c 'nohup dockerd >/var/log/dockerd.log 2>&1 &'
  for _ in $(seq 1 30); do
    if sudo docker info >/dev/null 2>&1; then
      echo "dockerd is up."
      return
    fi
    sleep 1
  done
  echo "dockerd did not become ready" >&2
  sudo tail -n 40 /var/log/dockerd.log >&2 || true
  exit 1
}

ensure_dockerd

echo "Bringing up the stack (frontend, API, Postgres+pgvector, MinIO, fixtures)..."
sudo docker compose up --build -d

echo "Waiting for the API to become healthy..."
for _ in $(seq 1 90); do
  if curl -fsS http://localhost:8080/health >/dev/null 2>&1; then
    echo "API healthy at http://localhost:8080/health"
    break
  fi
  sleep 2
done

sudo docker compose ps
cat <<'EOF'

Blog Copilot is running:
  Frontend:      http://localhost:3000
  API:           http://localhost:8080
  Health:        http://localhost:8080/health
  Swagger UI:    http://localhost:8080/swagger
  MinIO console: http://localhost:9001
EOF
