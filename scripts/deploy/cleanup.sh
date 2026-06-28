#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
start_deploy_step "[Phase 7] Cleanup old backups"

MAX_BACKUPS=5
for dir in backup/binary; do
  if [[ -d "$dir" ]]; then
    FILE_COUNT=$(find "$dir" -maxdepth 1 -type f | wc -l)
    if [[ "$FILE_COUNT" -gt "$MAX_BACKUPS" ]]; then
      DELETE_COUNT=$((FILE_COUNT - MAX_BACKUPS))
      find "$dir" -maxdepth 1 -type f -printf '%T@ %p\n' \
        | sort -n \
        | head -n "$DELETE_COUNT" \
        | awk '{print $2}' \
        | xargs rm -f
      echo "Deleted $DELETE_COUNT old backups in $dir"
    fi
  fi
done

bash "$PROJECT_DIR/scripts/notify-telegram.sh" "Entire deployment process completed successfully!"
