#!/usr/bin/env bash
# Agent-agnostic startup for the Blog Copilot Docker Compose stack.
# Ensures a Docker daemon is available, then brings the full stack up and waits
# for the API health check. Idempotent; safe to re-run.
set -euo pipefail

cd "$(dirname "$0")/.."

SUDO=""
if [ "$(id -u)" -ne 0 ]; then SUDO="sudo"; fi

ensure_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "Docker is not installed. Run: bash .agents/install.sh" >&2
    exit 1
  fi
  if $SUDO docker info >/dev/null 2>&1; then
    echo "Docker daemon already running."
    return
  fi
  # Bare VM (e.g. Cloud Agent sandbox): start the daemon.
  echo "Starting dockerd..."
  $SUDO bash -c 'nohup dockerd >/var/log/dockerd.log 2>&1 &'
  for _ in $(seq 1 30); do
    if $SUDO docker info >/dev/null 2>&1; then
      echo "dockerd is up."
      return
    fi
    sleep 1
  done
  echo "dockerd did not become ready" >&2
  $SUDO tail -n 40 /var/log/dockerd.log >&2 || true
  exit 1
}

ensure_docker

echo "Bringing up the stack (frontend, API, Postgres+pgvector, MinIO, fixtures)..."
$SUDO docker compose up --build -d

echo "Waiting for the API to become healthy..."
healthy=0
for _ in $(seq 1 90); do
  if curl -fsS http://localhost:8080/health >/dev/null 2>&1; then
    echo "API healthy at http://localhost:8080/health"
    healthy=1
    break
  fi
  sleep 2
done

if [ "$healthy" -ne 1 ]; then
  echo "API did not become healthy within 180s" >&2
  $SUDO docker compose ps >&2 || true
  $SUDO docker compose logs --tail=80 >&2 || true
  exit 1
fi

bash .agents/verify.sh
