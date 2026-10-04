#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
start_deploy_step "[Phase 4.1] Offsite backup (ocl)"

bash "$PROJECT_DIR/scripts/backup-db-offsite.sh"
