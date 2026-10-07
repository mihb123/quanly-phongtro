#!/usr/bin/env bash
set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_DIR="$(cd "$SKILL_DIR/../../.." && pwd)"

SEPAY_CRED_FILE="${SEPAY_CRED_FILE:-$HOME/.config/quanly-phongtro/sepay-test.env}"
if [[ ! -r "$SEPAY_CRED_FILE" ]]; then
  echo "Thiếu $SEPAY_CRED_FILE (SEPAY_LOGIN_EMAIL, SEPAY_LOGIN_PASSWORD, APP_MANAGER_EMAIL, APP_MANAGER_PASSWORD)" >&2
  exit 1
fi
set -a
# shellcheck disable=SC1090
. "$SEPAY_CRED_FILE"
set +a

APP_BASE_URL="${APP_BASE_URL:-https://quanly.ptro.site}"
API="$APP_BASE_URL/api/v1"
PW_SESSION="${PW_SESSION:-sepay}"
SEPAY_TESTMODE_URL="https://my.sepay.vn/testmode"
SEPAY_TEST_ACCOUNT="${SEPAY_TEST_ACCOUNT:-0000000001}"
UPLOADS_DIR="${UPLOADS_DIR:-/home/dell/actions-runner/_work/quanly-phongtro/quanly-phongtro/uploads/zalo-invoices}"

STATE_DIR="${XDG_RUNTIME_DIR:-/tmp}/sepay-e2e-$USER"
mkdir -p "$STATE_DIR"
chmod 700 "$STATE_DIR"
COOKIE_JAR="$STATE_DIR/app-cookies.txt"
TRACK_FILE="$STATE_DIR/created-invoices.txt"

if [[ -z "${POSTGRES_DSN:-}" ]]; then
  POSTGRES_DSN="$(grep -m1 '^POSTGRES_DSN=' "$REPO_DIR/.env" | cut -d= -f2- | tr -d "'\"")"
fi

db() {
  psql "$POSTGRES_DSN" -X -v ON_ERROR_STOP=1 "$@"
}

api() {
  local method="$1" path="$2"
  shift 2
  curl -sS -m 60 -b "$COOKIE_JAR" -c "$COOKIE_JAR" -X "$method" -H 'content-type: application/json' "$@" "$API$path"
}

pw() {
  playwright-cli -s="$PW_SESSION" "$@"
}

require_uuid() {
  [[ "$1" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]] || { echo "UUID không hợp lệ: $1" >&2; exit 1; }
}
