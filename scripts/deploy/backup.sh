#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
start_deploy_step "[Phase 4] Backup"
require_env SERVICE_NAME

chmod +x scripts/backup-db.sh
./scripts/backup-db.sh

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
GIT_HASH=$(git rev-parse --short HEAD~1 2>/dev/null || echo "unknown")

mkdir -p backup/binary
if [[ -f "$SERVICE_NAME" ]]; then
  cp "$SERVICE_NAME" "backup/binary/${SERVICE_NAME}_${TIMESTAMP}_${GIT_HASH}"
  echo "Binary backup successful"
else
  echo "Current binary not found, skipping backup"
fi
