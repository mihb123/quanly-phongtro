#!/usr/bin/env bash
set -euo pipefail

: "${GITHUB_ENV:?GITHUB_ENV is required}"
: "${GITHUB_PATH:?GITHUB_PATH is required}"
: "${GITHUB_WORKSPACE:?GITHUB_WORKSPACE is required}"

PROJECT_DIR="${PROJECT_DIR:-$GITHUB_WORKSPACE}"
DEPLOY_USER="${DEPLOY_USER:-$(id -un)}"
DEPLOY_GROUP="${DEPLOY_GROUP:-$(id -gn)}"
FRONTEND_DIR="${FRONTEND_DIR:-/var/www/quanly-phongtro}"
NGINX_SITE_NAME="${NGINX_SITE_NAME:-quanly-phongtro}"
NGINX_AVAILABLE_DIR="${NGINX_AVAILABLE_DIR:-/etc/nginx/sites-available}"
NGINX_ENABLED_DIR="${NGINX_ENABLED_DIR:-/etc/nginx/sites-enabled}"

{
  echo "PROJECT_DIR=$PROJECT_DIR"
  echo "DEPLOY_USER=$DEPLOY_USER"
  echo "DEPLOY_GROUP=$DEPLOY_GROUP"
  echo "FRONTEND_DIR=$FRONTEND_DIR"
  echo "NGINX_SITE_NAME=$NGINX_SITE_NAME"
  echo "NGINX_AVAILABLE_DIR=$NGINX_AVAILABLE_DIR"
  echo "NGINX_ENABLED_DIR=$NGINX_ENABLED_DIR"
} >> "$GITHUB_ENV"

echo "/usr/local/go/bin" >> "$GITHUB_PATH"
echo "$HOME/go/bin" >> "$GITHUB_PATH"
