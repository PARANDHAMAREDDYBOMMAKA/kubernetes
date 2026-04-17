#!/usr/bin/env bash
set -euo pipefail

# Run on a fresh Ubuntu 22.04 / 24.04 DigitalOcean droplet as root.
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/PARANDHAMAREDDYBOMMAKA/kubernetes/main/deploy/bootstrap.sh | bash

REPO_URL="https://github.com/PARANDHAMAREDDYBOMMAKA/kubernetes.git"
APP_DIR="/opt/kaas"

echo "==> Installing Docker + compose plugin"
apt-get update -y
apt-get install -y ca-certificates curl git gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" > /etc/apt/sources.list.d/docker.list
apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

echo "==> Cloning repo to ${APP_DIR}"
if [ ! -d "${APP_DIR}" ]; then
  git clone "${REPO_URL}" "${APP_DIR}"
else
  git -C "${APP_DIR}" pull --ff-only
fi

cd "${APP_DIR}/deploy"
if [ ! -f .env ]; then
  cp .env.example .env
  IP=$(curl -fsSL https://ifconfig.me || true)
  if [ -n "$IP" ]; then
    sed -i "s/REPLACE_WITH_DROPLET_IP/${IP}/" .env
  fi
  echo
  echo "==> Created deploy/.env — edit DOCKERHUB_USERNAME, then run:"
  echo "    cd ${APP_DIR}/deploy && docker compose -f docker-compose.prod.yml up -d"
  exit 0
fi

echo "==> Pulling + starting stack"
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d --build

echo
echo "==> Up. Visit: https://$(grep ^SITE_HOST .env | cut -d= -f2)"
