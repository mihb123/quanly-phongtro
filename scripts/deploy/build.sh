#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
start_deploy_step "[Phase 3] Build"
require_env SERVICE_NAME

# Build frontend trước để có bản dist mới nhất, vì binary sẽ nhúng thư mục này.
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

# Nhúng bản build frontend vào package web rồi build ra một binary duy nhất.
echo "Embedding frontend and building backend..."
cd "$PROJECT_DIR"
rm -rf internal/web/dist
cp -r frontend/dist internal/web/dist
go build -o "${SERVICE_NAME}.new" ./cmd/api
