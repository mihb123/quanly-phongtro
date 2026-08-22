#!/usr/bin/env bash
# Backup PostgreSQL database trước mỗi lần deploy.
# Giữ tối đa MAX_BACKUPS bản backup gần nhất, tự động xoá bản cũ.
# Sử dụng: ./scripts/backup-db.sh
# Yêu cầu: pg_dump phải có sẵn, file .env chứa POSTGRES_DSN.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="$PROJECT_DIR/backup/database"
MAX_BACKUPS=5

# Load biến môi trường từ .env
if [[ ! -f "$PROJECT_DIR/.env" ]]; then
  echo "❌ File .env không tồn tại tại $PROJECT_DIR/.env"
  exit 1
fi

set -a
source "$PROJECT_DIR/.env"
set +a

if [[ -z "${POSTGRES_DSN:-}" ]]; then
  echo "❌ Biến POSTGRES_DSN không được thiết lập trong .env"
  exit 1
fi

mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
GIT_SHORT_HASH=$(cd "$PROJECT_DIR" && git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BACKUP_FILE="$BACKUP_DIR/db_${TIMESTAMP}_${GIT_SHORT_HASH}.sql.gz"

echo "📦 Bắt đầu backup database..."
echo "   Thời gian : $TIMESTAMP"
echo "   Git commit: $GIT_SHORT_HASH"
echo "   Đích      : $BACKUP_FILE"

# Thực hiện backup, nén bằng gzip để tiết kiệm dung lượng
if pg_dump "$POSTGRES_DSN" | gzip > "$BACKUP_FILE"; then
  BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
  echo "✅ Backup thành công ($BACKUP_SIZE)"
else
  echo "❌ Backup thất bại"
  rm -f "$BACKUP_FILE"
  exit 1
fi

# Xoá các bản backup cũ, chỉ giữ lại MAX_BACKUPS bản mới nhất
BACKUP_COUNT=$(find "$BACKUP_DIR" -name "db_*.sql.gz" -type f | wc -l)
if [[ "$BACKUP_COUNT" -gt "$MAX_BACKUPS" ]]; then
  DELETE_COUNT=$((BACKUP_COUNT - MAX_BACKUPS))
  echo "🗑️  Xoá $DELETE_COUNT bản backup cũ (giữ lại $MAX_BACKUPS bản mới nhất)..."
  find "$BACKUP_DIR" -name "db_*.sql.gz" -type f -printf '%T@ %p\n' \
    | sort -n \
    | head -n "$DELETE_COUNT" \
    | awk '{print $2}' \
    | xargs rm -f
fi

echo "📋 Danh sách backup hiện tại:"
ls -lh "$BACKUP_DIR"/db_*.sql.gz 2>/dev/null || echo "   (trống)"
