#!/usr/bin/env bash
# Convenience helper to run the KaaS web UI locally.
# Usage: ./scripts/web-dev.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEB_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)/web"

cd "${WEB_DIR}"

if [ ! -d node_modules ]; then
  echo "[web-dev] Installing dependencies..."
  npm install
fi

echo "[web-dev] Starting Vite dev server on http://localhost:5173"
echo "[web-dev] Backend API expected at \$VITE_API_BASE_URL (default http://localhost:8080)"
exec npm run dev
