#!/usr/bin/env bash
set -euo pipefail

: "${GITHUB_ENV:?GITHUB_ENV is required}"
: "${GITHUB_PATH:?GITHUB_PATH is required}"
: "${GITHUB_WORKSPACE:?GITHUB_WORKSPACE is required}"

PROJECT_DIR="${PROJECT_DIR:-$GITHUB_WORKSPACE}"
DEPLOY_USER="${DEPLOY_USER:-$(id -un)}"
DEPLOY_GROUP="${DEPLOY_GROUP:-$(id -gn)}"
# File .env thật được giữ lâu dài NGOÀI workspace (workspace bị checkout/agent xóa file untracked).
ENV_FILE_SOURCE="${ENV_FILE_SOURCE:-$HOME/quanly-phongtro/.env}"

# link_env_file trỏ $PROJECT_DIR/.env sang file .env dùng chung để mỗi lần deploy
# đều có .env, kể cả khi file trong workspace bị xóa. Idempotent; không tự-symlink.
link_env_file() {
  local env_link="$PROJECT_DIR/.env"

  # PROJECT_DIR chính là thư mục chứa .env gốc: dùng trực tiếp, tránh symlink trỏ vào chính nó.
  if [[ "$env_link" == "$ENV_FILE_SOURCE" || "$env_link" -ef "$ENV_FILE_SOURCE" ]]; then
    return 0
  fi

  if [[ ! -f "$ENV_FILE_SOURCE" ]]; then
    echo "Không tìm thấy file .env dùng chung: $ENV_FILE_SOURCE" >&2
    return 1
  fi

  mkdir -p "$(dirname "$env_link")"
  ln -sfn "$ENV_FILE_SOURCE" "$env_link"
  echo "Đã trỏ symlink .env: $env_link -> $ENV_FILE_SOURCE"
}

link_env_file

{
  echo "PROJECT_DIR=$PROJECT_DIR"
  echo "DEPLOY_USER=$DEPLOY_USER"
  echo "DEPLOY_GROUP=$DEPLOY_GROUP"
} >> "$GITHUB_ENV"

echo "/usr/local/go/bin" >> "$GITHUB_PATH"
echo "$HOME/go/bin" >> "$GITHUB_PATH"
