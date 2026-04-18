#!/bin/sh
set -eu

STAMP=$(date -u +%Y%m%dT%H%M%SZ)
DEST="/backups/mongo-${STAMP}.archive.gz"
KEEP_DAYS="${BACKUP_KEEP_DAYS:-7}"

echo "[$(date -u +%H:%M:%S)] dump -> ${DEST}"
mongodump --host=mongodb:27017 --db="${MONGO_DB:-kaas}" --archive="${DEST}" --gzip --quiet

echo "[$(date -u +%H:%M:%S)] prune backups older than ${KEEP_DAYS} days"
find /backups -name 'mongo-*.archive.gz' -type f -mtime +"${KEEP_DAYS}" -print -delete || true

echo "[$(date -u +%H:%M:%S)] done"
