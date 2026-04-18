#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/kaas}"

if [ ! -d "$APP_DIR" ]; then
  echo "ERROR: $APP_DIR not found. Run deploy/bootstrap.sh first." >&2
  exit 1
fi

install -m 0755 "$APP_DIR/deploy/autodeploy.sh" /usr/local/bin/kaas-autodeploy
install -m 0644 "$APP_DIR/deploy/kaas-autodeploy.service" /etc/systemd/system/kaas-autodeploy.service
install -m 0644 "$APP_DIR/deploy/kaas-autodeploy.timer"   /etc/systemd/system/kaas-autodeploy.timer

sed -i 's|ExecStart=.*|ExecStart=/usr/local/bin/kaas-autodeploy|' /etc/systemd/system/kaas-autodeploy.service

systemctl daemon-reload
systemctl enable --now kaas-autodeploy.timer

echo
echo "==> installed. status:"
systemctl status kaas-autodeploy.timer --no-pager || true
echo
echo "==> logs (live):    journalctl -u kaas-autodeploy.service -f"
echo "==> run now:        systemctl start kaas-autodeploy.service"
echo "==> disable:        systemctl disable --now kaas-autodeploy.timer"
