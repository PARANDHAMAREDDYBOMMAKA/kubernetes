#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/kaas}"
BRANCH="${DEPLOY_BRANCH:-main}"
COMPOSE="docker compose -f docker-compose.prod.yml"

cd "$APP_DIR"

git fetch --quiet origin "$BRANCH"
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse "origin/${BRANCH}")

if [ "$LOCAL" = "$REMOTE" ]; then
  echo "$(date -u +%H:%M:%SZ) up-to-date at ${LOCAL:0:7}"
  exit 0
fi

echo "$(date -u +%H:%M:%SZ) deploying ${LOCAL:0:7} -> ${REMOTE:0:7}"

git reset --hard "origin/${BRANCH}"

cd "$APP_DIR/deploy"
COMMIT_SHORT=$(git -C "$APP_DIR" rev-parse --short HEAD)
if grep -q '^GIT_COMMIT=' .env; then
  sed -i "s/^GIT_COMMIT=.*/GIT_COMMIT=${COMMIT_SHORT}/" .env
else
  echo "GIT_COMMIT=${COMMIT_SHORT}" >> .env
fi

$COMPOSE up -d --build --remove-orphans
$COMPOSE ps

echo "$(date -u +%H:%M:%SZ) deployed ${COMMIT_SHORT}"
