#!/usr/bin/env bash
set -euo pipefail

: "${GITHUB_ENV:?GITHUB_ENV is required}"
: "${GITHUB_PATH:?GITHUB_PATH is required}"
: "${GITHUB_WORKSPACE:?GITHUB_WORKSPACE is required}"

PROJECT_DIR="${PROJECT_DIR:-$GITHUB_WORKSPACE}"
DEPLOY_USER="${DEPLOY_USER:-$(id -un)}"
DEPLOY_GROUP="${DEPLOY_GROUP:-$(id -gn)}"

{
  echo "PROJECT_DIR=$PROJECT_DIR"
  echo "DEPLOY_USER=$DEPLOY_USER"
  echo "DEPLOY_GROUP=$DEPLOY_GROUP"
} >> "$GITHUB_ENV"

echo "/usr/local/go/bin" >> "$GITHUB_PATH"
echo "$HOME/go/bin" >> "$GITHUB_PATH"
