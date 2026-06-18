#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
start_deploy_step "[Phase 3] Build"
require_env SERVICE_NAME

echo "Building backend..."
go build -o "${SERVICE_NAME}.new" ./cmd/api

echo "Building frontend..."
cd "$PROJECT_DIR/frontend"
if command -v pnpm &> /dev/null; then
  pnpm install --frozen-lockfile
  pnpm lint
  pnpm build
else
  npm install
  npm run lint
  npm run build
fi
