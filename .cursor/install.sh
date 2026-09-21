#!/usr/bin/env bash
# Idempotent environment bootstrap for the Blog Copilot Docker Compose stack.
# Installs Docker Engine (Compose plugin) and pre-builds the compose images so
# the first `start` is fast. Safe to run repeatedly.
set -euo pipefail

cd "$(dirname "$0")/.."

install_docker() {
  if command -v docker >/dev/null 2>&1; then
    echo "Docker already installed: $(docker --version)"
    return
  fi

  echo "Installing Docker Engine..."
  export DEBIAN_FRONTEND=noninteractive
  sudo install -m 0755 -d /etc/apt/keyrings
  if [ ! -f /etc/apt/keyrings/docker.gpg ]; then
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
      | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    sudo chmod a+r /etc/apt/keyrings/docker.gpg
  fi
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
    | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
  sudo apt-get update -qq
  sudo apt-get install -y -o Dpkg::Options::=--force-confold \
    docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin \
    fuse-overlayfs iptables
  echo "Installed: $(docker --version)"
}

configure_docker() {
  # The Cloud Agent VM is itself a container, so use fuse-overlayfs and the
  # legacy iptables backend for the nested Docker daemon.
  sudo mkdir -p /etc/docker
  echo '{ "storage-driver": "fuse-overlayfs", "iptables": true }' \
    | sudo tee /etc/docker/daemon.json >/dev/null
  sudo update-alternatives --set iptables /usr/sbin/iptables-legacy >/dev/null 2>&1 || true
}

start_dockerd() {
  if sudo docker info >/dev/null 2>&1; then
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

install_docker
configure_docker
start_dockerd

echo "Pre-building compose images (this warms the build cache)..."
sudo docker compose build

echo "install.sh complete."
