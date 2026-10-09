#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BACKUP_DIR="$PROJECT_DIR/backup/database"
UPLOADS_DIR="${UPLOADS_DIR:-$PROJECT_DIR/uploads}"
OFFSITE_HOST="${OFFSITE_HOST:-ocl}"
OFFSITE_DIR="${OFFSITE_DIR:-backup-quanly-phongtro}"
OFFSITE_MAX_BACKUPS="${OFFSITE_MAX_BACKUPS:-5}"
OFFSITE_SSH_KEY="${OFFSITE_SSH_KEY:-$HOME/.ssh/qlpt_backup_ed25519}"
OFFSITE_SSH_PASSFILE="${OFFSITE_SSH_PASSFILE:-$HOME/dotfile/qlpt/backup-ssh.pass}"
SSH_OPTS=(-o BatchMode=yes -o ConnectTimeout=15 -o IdentitiesOnly=yes -i "$OFFSITE_SSH_KEY")

for f in "$OFFSITE_SSH_KEY" "$OFFSITE_SSH_PASSFILE"; do
  if [[ ! -r "$f" ]]; then
    echo "❌ Thiếu $f (key/passphrase SSH cho backup offsite)"
    exit 1
  fi
done

ASKPASS=$(mktemp)
trap 'ssh-agent -k >/dev/null 2>&1 || true; rm -f "$ASKPASS"' EXIT
printf '#!/bin/sh\nexec cat %q\n' "$OFFSITE_SSH_PASSFILE" > "$ASKPASS"
chmod 700 "$ASKPASS"
eval "$(ssh-agent -s)" >/dev/null
SSH_ASKPASS="$ASKPASS" SSH_ASKPASS_REQUIRE=force ssh-add -q "$OFFSITE_SSH_KEY" </dev/null

LATEST_BACKUP=$(find "$BACKUP_DIR" -maxdepth 1 -name "db_*.sql.gz" -type f -printf '%T@ %p\n' 2>/dev/null \
  | sort -n \
  | tail -n 1 \
  | awk '{print $2}')

if [[ -z "$LATEST_BACKUP" ]]; then
  echo "❌ Không tìm thấy bản backup nào trong $BACKUP_DIR"
  exit 1
fi

BACKUP_NAME=$(basename "$LATEST_BACKUP")
echo "☁️  Chuyển backup sang $OFFSITE_HOST:~/$OFFSITE_DIR/$BACKUP_NAME ..."

ssh "${SSH_OPTS[@]}" "$OFFSITE_HOST" "mkdir -p $OFFSITE_DIR"
rsync -a --partial -e "ssh ${SSH_OPTS[*]}" "$LATEST_BACKUP" "$OFFSITE_HOST:$OFFSITE_DIR/.$BACKUP_NAME.part"
ssh "${SSH_OPTS[@]}" "$OFFSITE_HOST" "mv -f $OFFSITE_DIR/.$BACKUP_NAME.part $OFFSITE_DIR/$BACKUP_NAME && gzip -t $OFFSITE_DIR/$BACKUP_NAME"
echo "✅ Đã chuyển backup sang $OFFSITE_HOST"

ssh "${SSH_OPTS[@]}" "$OFFSITE_HOST" \
  "cd $OFFSITE_DIR && ls -1t db_*.sql.gz | tail -n +$((OFFSITE_MAX_BACKUPS + 1)) | xargs -r rm -f && echo '📋 Backup trên $OFFSITE_HOST:' && ls -lh db_*.sql.gz"

if [[ -d "$UPLOADS_DIR" ]]; then
  echo "☁️  Đồng bộ ảnh upload sang $OFFSITE_HOST:~/$OFFSITE_DIR/uploads/ ..."
  rsync -a --partial -e "ssh ${SSH_OPTS[*]}" "$UPLOADS_DIR"/ "$OFFSITE_HOST:$OFFSITE_DIR/uploads/"
  echo "✅ Đã đồng bộ ảnh upload sang $OFFSITE_HOST"
else
  echo "⚠️  Không có thư mục upload $UPLOADS_DIR, bỏ qua"
fi
