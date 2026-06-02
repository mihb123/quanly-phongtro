#!/bin/bash

STEP_NAME="$1"
LOG_FILE="/tmp/deploy_step.log"

# Clear old log
> "$LOG_FILE"

# Redirect stdout/stderr to tee
exec > >(tee "$LOG_FILE") 2>&1

trap "
  EXIT_CODE=\$?
  sleep 1
  if [ \$EXIT_CODE -ne 0 ]; then
    LOG_CONTENT=\$(tail -n 30 \"$LOG_FILE\")
    bash \"$PROJECT_DIR/scripts/notify-telegram.sh\" \"❌ $STEP_NAME thất bại!
Log:
\$LOG_CONTENT\"
  else
    bash \"$PROJECT_DIR/scripts/notify-telegram.sh\" \"✅ $STEP_NAME thành công\"
  fi
" EXIT
