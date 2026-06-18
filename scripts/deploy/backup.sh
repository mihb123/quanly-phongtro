#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
start_deploy_step "[Phase 4] Backup"
require_env SERVICE_NAME
require_env FRONTEND_DIR

chmod +x scripts/backup-db.sh
./scripts/backup-db.sh

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
GIT_HASH=$(git rev-parse --short HEAD~1 2>/dev/null || echo "unknown")

mkdir -p backup/binary
if [[ -f "$SERVICE_NAME" ]]; then
  cp "$SERVICE_NAME" "backup/binary/${SERVICE_NAME}_${TIMESTAMP}_${GIT_HASH}"
  echo "Backup binary thanh cong"
else
  echo "Khong tim thay binary hien tai, bo qua backup"
fi

mkdir -p backup/frontend
if [[ -d "$FRONTEND_DIR" ]] && [[ -n "$(ls -A "$FRONTEND_DIR" 2>/dev/null)" ]]; then
  tar -czf "backup/frontend/frontend_${TIMESTAMP}_${GIT_HASH}.tar.gz" -C "$FRONTEND_DIR" .
  echo "Backup frontend thanh cong"
else
  echo "Thu muc frontend rong hoac khong ton tai, bo qua backup"
fi
