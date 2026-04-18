#!/usr/bin/env bash
set -euo pipefail


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
  SECRET=$(head -c 48 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 48)
  sed -i "s/REPLACE_WITH_LONG_RANDOM_STRING/${SECRET}/" .env
  DGID=$(getent group docker | awk -F: '{print $3}')
  if [ -n "${DGID:-}" ]; then
    sed -i "s/^DOCKER_GID=.*/DOCKER_GID=${DGID}/" .env
  fi
  GIT_COMMIT=$(git -C "${APP_DIR}" rev-parse --short HEAD)
  echo "GIT_COMMIT=${GIT_COMMIT}" >> .env
  echo
  echo "==> Wrote deploy/.env. Edit SITE_HOST if you want a custom domain, then run:"
  echo "    cd ${APP_DIR}/deploy && docker compose -f docker-compose.prod.yml up -d --build"
  exit 0
fi

echo "==> Building + starting stack"
docker compose -f docker-compose.prod.yml up -d --build

echo
echo "==> Up. Visit: https://$(grep ^SITE_HOST .env | cut -d= -f2)"
