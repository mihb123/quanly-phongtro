#!/bin/bash
# Script to send Telegram notifications

# Load .env if exists
if [ -f "$PROJECT_DIR/.env" ]; then
  set -a
  source "$PROJECT_DIR/.env"
  set +a
elif [ -f "$HOME/quanly-phongtro/.env" ]; then
  set -a
  source "$HOME/quanly-phongtro/.env"
  set +a
fi

if [ -n "$TELEGRAM_BOT_TOKEN" ] && [ -n "$TELEGRAM_CHAT_ID" ]; then
  MESSAGE="$1"
  curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
       -d chat_id="${TELEGRAM_CHAT_ID}" \
       --data-urlencode text="$MESSAGE" > /dev/null
fi
