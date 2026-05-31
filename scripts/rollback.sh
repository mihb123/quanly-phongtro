#!/usr/bin/env bash
# Rollback hệ thống về version trước.
# Hỗ trợ rollback riêng lẻ hoặc kết hợp: backend, frontend, database.
#
# Sử dụng:
#   ./scripts/rollback.sh --backend                 — Rollback backend binary
#   ./scripts/rollback.sh --frontend                — Rollback frontend
#   ./scripts/rollback.sh --backend --frontend      — Rollback cả backend + frontend
#   ./scripts/rollback.sh --db <file>               — Restore database từ file cụ thể
#   ./scripts/rollback.sh --db                      — Restore database (interactive chọn file)
#   ./scripts/rollback.sh --full                    — Rollback tất cả (db + backend + frontend)
#   ./scripts/rollback.sh --list                    — Liệt kê tất cả backup
#
# Yêu cầu: psql phải có sẵn nếu rollback database, file .env chứa POSTGRES_DSN.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="$PROJECT_DIR/backup"
DB_BACKUP_DIR="$BACKUP_DIR/database"
BINARY_BACKUP_DIR="$BACKUP_DIR/binary"
FRONTEND_BACKUP_DIR="$BACKUP_DIR/frontend"
SERVICE_NAME="quanly-phongtro-api"

# ─── Helper functions ─────────────────────────────────────

# Load biến môi trường từ .env
load_env() {
  if [[ ! -f "$PROJECT_DIR/.env" ]]; then
    echo "❌ File .env không tồn tại tại $PROJECT_DIR/.env"
    exit 1
  fi
  set -a
  source "$PROJECT_DIR/.env"
  set +a
}

# Hiển thị danh sách backup trong một thư mục
list_backups_in_directory() {
  local backup_dir="$1"
  local pattern="$2"
  local label="$3"

  echo "📋 Các bản backup $label:"
  echo "─────────────────────────────────────────────────"
  if [[ -d "$backup_dir" ]] && ls "$backup_dir"/$pattern &>/dev/null; then
    local index=1
    while IFS= read -r file; do
      local filename
      filename=$(basename "$file")
      local size
      size=$(du -h "$file" | cut -f1)
      local modified
      modified=$(date -r "$file" "+%Y-%m-%d %H:%M:%S")
      echo "  [$index] $filename ($size) — $modified"
      index=$((index + 1))
    done < <(ls -t "$backup_dir"/$pattern)
  else
    echo "  (không có bản backup nào)"
  fi
  echo ""
}

# Chọn file backup từ danh sách (interactive), trả kết quả qua biến SELECTED_FILE
select_backup_file() {
  local backup_dir="$1"
  local pattern="$2"
  local label="$3"

  SELECTED_FILE=""

  if [[ ! -d "$backup_dir" ]] || ! ls "$backup_dir"/$pattern &>/dev/null; then
    echo "❌ Không có bản backup $label nào."
    return 1
  fi

  local files=()
  while IFS= read -r file; do
    files+=("$file")
  done < <(ls -t "$backup_dir"/$pattern)

  echo "Chọn bản backup $label để restore:"
  local index=1
  for file in "${files[@]}"; do
    local filename
    filename=$(basename "$file")
    local size
    size=$(du -h "$file" | cut -f1)
    echo "  [$index] $filename ($size)"
    index=$((index + 1))
  done

  read -rp "Nhập số thứ tự (1-${#files[@]}, hoặc 0 để bỏ qua): " choice

  if [[ "$choice" == "0" ]]; then
    echo "⏭️  Bỏ qua restore $label."
    return 1
  fi

  if [[ "$choice" -ge 1 && "$choice" -le "${#files[@]}" ]]; then
    SELECTED_FILE="${files[$((choice - 1))]}"
    return 0
  else
    echo "❌ Lựa chọn không hợp lệ."
    return 1
  fi
}

# ─── Restore functions ────────────────────────────────────

# Restore database từ file backup
restore_database() {
  local backup_file="$1"

  if [[ ! -f "$backup_file" ]]; then
    echo "❌ File backup không tồn tại: $backup_file"
    return 1
  fi

  load_env

  echo "⚠️  SẮP RESTORE DATABASE từ: $(basename "$backup_file")"
  echo "   Toàn bộ dữ liệu hiện tại sẽ bị ghi đè!"
  read -rp "   Xác nhận? (yes/no): " confirm
  if [[ "$confirm" != "yes" ]]; then
    echo "❌ Huỷ restore database."
    return 1
  fi

  local db_name
  db_name=$(echo "$POSTGRES_DSN" | sed -n 's|.*/\([^?]*\).*|\1|p')
  local base_dsn
  base_dsn=$(echo "$POSTGRES_DSN" | sed "s|/$db_name|/postgres|")

  echo "📦 Đang restore database '$db_name'..."

  # Terminate các connection đang active tới database
  psql "$base_dsn" -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$db_name' AND pid <> pg_backend_pid();" >/dev/null 2>&1 || true

  psql "$base_dsn" -c "DROP DATABASE IF EXISTS $db_name;" >/dev/null 2>&1
  psql "$base_dsn" -c "CREATE DATABASE $db_name;" >/dev/null 2>&1

  if gunzip -c "$backup_file" | psql "$POSTGRES_DSN" > /dev/null 2>&1; then
    echo "✅ Restore database thành công"
  else
    echo "❌ Restore database thất bại"
    return 1
  fi
}

# Restore binary backend từ backup
restore_backend() {
  local backup_file="$1"

  if [[ ! -f "$backup_file" ]]; then
    echo "❌ File backup binary không tồn tại: $backup_file"
    return 1
  fi

  echo "📦 Đang restore backend từ: $(basename "$backup_file")"
  cp "$backup_file" "$PROJECT_DIR/$SERVICE_NAME"
  chmod +x "$PROJECT_DIR/$SERVICE_NAME"

  echo "🔄 Đang restart service..."
  sudo /usr/bin/systemctl restart "$SERVICE_NAME"
  sleep 2
  sudo /usr/bin/systemctl status "$SERVICE_NAME" --no-pager
  echo "✅ Restore backend thành công"
}

# Restore frontend từ backup
restore_frontend() {
  local backup_file="$1"

  if [[ ! -f "$backup_file" ]]; then
    echo "❌ File backup frontend không tồn tại: $backup_file"
    return 1
  fi

  echo "📦 Đang restore frontend từ: $(basename "$backup_file")"
  local frontend_destination="/var/www/quanly-phongtro"
  rm -rf "$frontend_destination"/*
  tar -xzf "$backup_file" -C "$frontend_destination"
  sudo /usr/bin/systemctl reload nginx
  echo "✅ Restore frontend thành công"
}

# ─── Command handlers ─────────────────────────────────────

# Xử lý rollback backend (interactive chọn file)
handle_rollback_backend() {
  echo "── Backend ─────────────────────────────────────────"
  if select_backup_file "$BINARY_BACKUP_DIR" "${SERVICE_NAME}_*" "backend"; then
    restore_backend "$SELECTED_FILE"
  fi
  echo ""
}

# Xử lý rollback frontend (interactive chọn file)
handle_rollback_frontend() {
  echo "── Frontend ────────────────────────────────────────"
  if select_backup_file "$FRONTEND_BACKUP_DIR" "frontend_*.tar.gz" "frontend"; then
    restore_frontend "$SELECTED_FILE"
  fi
  echo ""
}

# Xử lý rollback database (interactive chọn file hoặc từ đường dẫn cụ thể)
handle_rollback_database() {
  local specific_file="${1:-}"

  if [[ -n "$specific_file" ]]; then
    restore_database "$specific_file"
    return
  fi

  echo "── Database ────────────────────────────────────────"
  if select_backup_file "$DB_BACKUP_DIR" "db_*.sql.gz" "database"; then
    restore_database "$SELECTED_FILE"
  fi
  echo ""
}

# Hiển thị header rollback
show_rollback_header() {
  local components="$1"
  echo ""
  echo "🔄 ROLLBACK: $components"
  echo "═══════════════════════════════════════════════════"
  echo ""
}

# Hiển thị footer rollback
show_rollback_footer() {
  echo "═══════════════════════════════════════════════════"
  echo "✅ Rollback hoàn tất."
}

# Hiển thị hướng dẫn sử dụng
show_usage() {
  echo "Sử dụng:"
  echo "  $0 --backend                Rollback backend binary"
  echo "  $0 --frontend               Rollback frontend"
  echo "  $0 --backend --frontend     Rollback cả backend + frontend"
  echo "  $0 --db [file]              Restore database (interactive hoặc từ file cụ thể)"
  echo "  $0 --full                   Rollback tất cả (db + backend + frontend)"
  echo "  $0 --list                   Liệt kê tất cả các bản backup"
  echo "  $0 --help                   Hiển thị trợ giúp"
}

# ─── Main: Parse flags ────────────────────────────────────

DO_BACKEND=false
DO_FRONTEND=false
DO_DATABASE=false
DO_LIST=false
DB_FILE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --backend)
      DO_BACKEND=true
      shift
      ;;
    --frontend)
      DO_FRONTEND=true
      shift
      ;;
    --db)
      DO_DATABASE=true
      # Nếu argument tiếp theo tồn tại và không phải flag, coi là đường dẫn file
      if [[ -n "${2:-}" && "${2:0:2}" != "--" ]]; then
        DB_FILE="$2"
        shift
      fi
      shift
      ;;
    --full)
      DO_BACKEND=true
      DO_FRONTEND=true
      DO_DATABASE=true
      shift
      ;;
    --list)
      DO_LIST=true
      shift
      ;;
    --help | -h)
      show_usage
      exit 0
      ;;
    *)
      echo "❌ Flag không hợp lệ: $1"
      echo ""
      show_usage
      exit 1
      ;;
  esac
done

# Nếu không truyền flag nào → hiển thị help
if ! $DO_BACKEND && ! $DO_FRONTEND && ! $DO_DATABASE && ! $DO_LIST; then
  show_usage
  exit 0
fi

# --list: chỉ liệt kê, không rollback
if $DO_LIST; then
  list_backups_in_directory "$DB_BACKUP_DIR" "db_*.sql.gz" "database"
  list_backups_in_directory "$BINARY_BACKUP_DIR" "${SERVICE_NAME}_*" "backend"
  list_backups_in_directory "$FRONTEND_BACKUP_DIR" "frontend_*.tar.gz" "frontend"
  exit 0
fi

# Xác định label cho header
COMPONENTS=()
$DO_DATABASE && COMPONENTS+=("Database")
$DO_BACKEND && COMPONENTS+=("Backend")
$DO_FRONTEND && COMPONENTS+=("Frontend")
LABEL=$(IFS=" + "; echo "${COMPONENTS[*]}")

show_rollback_header "$LABEL"

# Thực thi rollback theo flags đã chọn
if $DO_DATABASE; then
  handle_rollback_database "$DB_FILE"
fi

if $DO_BACKEND; then
  handle_rollback_backend
fi

if $DO_FRONTEND; then
  handle_rollback_frontend
fi

show_rollback_footer
