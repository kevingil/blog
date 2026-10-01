#!/usr/bin/env bash
# Agent-agnostic environment bootstrap for the Blog Copilot Docker Compose stack.
#
# Purpose: make a bare Linux VM (e.g. a Cloud Agent sandbox that ships without
# Docker) capable of running `docker compose`. On a machine that already has a
# working Docker daemon (a normal dev laptop, most CI images), this only builds
# the compose images and leaves the host's Docker configuration untouched.
#
# Idempotent; safe to re-run. Intended for Debian/Ubuntu hosts.
set -euo pipefail

cd "$(dirname "$0")/.."

SUDO=""
if [ "$(id -u)" -ne 0 ]; then SUDO="sudo"; fi

install_docker() {
  if command -v docker >/dev/null 2>&1; then
    echo "Docker already installed: $(docker --version)"
    return
  fi
  echo "Installing Docker Engine..."
  export DEBIAN_FRONTEND=noninteractive
  $SUDO install -m 0755 -d /etc/apt/keyrings
  if [ ! -f /etc/apt/keyrings/docker.gpg ]; then
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
      | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    $SUDO chmod a+r /etc/apt/keyrings/docker.gpg
  fi
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
    | $SUDO tee /etc/apt/sources.list.d/docker.list >/dev/null
  $SUDO apt-get update -qq
  $SUDO apt-get install -y -o Dpkg::Options::=--force-confold \
    docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin \
    fuse-overlayfs iptables
  echo "Installed: $(docker --version)"
}

# Only touch daemon configuration and start dockerd when no daemon is running.
# This keeps hosts with an existing Docker setup completely untouched.
bootstrap_daemon_if_needed() {
  if $SUDO docker info >/dev/null 2>&1; then
    echo "Docker daemon already running; leaving host config untouched."
    return
  fi
  # A Cloud Agent VM is itself a container, so the nested daemon needs
  # fuse-overlayfs and the legacy iptables backend.
  $SUDO mkdir -p /etc/docker
  echo '{ "storage-driver": "fuse-overlayfs", "iptables": true }' \
    | $SUDO tee /etc/docker/daemon.json >/dev/null
  $SUDO update-alternatives --set iptables /usr/sbin/iptables-legacy >/dev/null 2>&1 || true
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

install_docker
bootstrap_daemon_if_needed

echo "Pre-building compose images (warms the build cache)..."
$SUDO docker compose build

echo "install.sh complete."
