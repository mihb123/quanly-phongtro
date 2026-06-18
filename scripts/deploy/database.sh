#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
start_deploy_step "[Phase 5] Database migrations & seed"

set -a
source .env
set +a

"$HOME/go/bin/migrate" -path migrations -database "$POSTGRES_DSN" up
go run ./cmd/seed/main.go
