#!/usr/bin/env bash
set -euo pipefail

: "${GITHUB_WORKSPACE:?GITHUB_WORKSPACE is required}"
: "${PROJECT_DIR:?PROJECT_DIR is required}"
: "${SERVICE_NAME:?SERVICE_NAME is required}"

source "$GITHUB_WORKSPACE/scripts/setup-trap.sh" "[Phase 1] Prepare source code"

if [[ "$PROJECT_DIR" != "$GITHUB_WORKSPACE" ]]; then
  mkdir -p "$PROJECT_DIR"
  if command -v rsync &> /dev/null; then
    rsync -a --delete \
      --exclude='.git' \
      --exclude='.env' \
      --exclude='backup/' \
      --exclude="$SERVICE_NAME" \
      --exclude="${SERVICE_NAME}.new" \
      "$GITHUB_WORKSPACE"/ "$PROJECT_DIR"/
  else
    tar \
      --exclude='./.git' \
      --exclude='./.env' \
      --exclude='./backup' \
      --exclude="./$SERVICE_NAME" \
      --exclude="./${SERVICE_NAME}.new" \
      -C "$GITHUB_WORKSPACE" -cf - . | tar -C "$PROJECT_DIR" -xf -
  fi
fi

cd "$PROJECT_DIR"
git -C "$GITHUB_WORKSPACE" rev-parse --short HEAD
