#!/usr/bin/env bash
. "$(dirname "$0")/env.sh"

umask 077
status="$(jq -n --arg e "$APP_MANAGER_EMAIL" --arg p "$APP_MANAGER_PASSWORD" '{email:$e,password:$p}' |
  curl -sS -m 20 -c "$COOKIE_JAR" -o /dev/null -w '%{http_code}' -H 'content-type: application/json' --data @- "$API/auth/login")"
if [[ "$status" != "200" ]]; then
  echo "Đăng nhập app thất bại (HTTP $status)" >&2
  exit 1
fi

api GET /auth/me | jq -c '.data | {user_id, email, role}'
api GET /payments/providers/sepay/config | jq -c '{has_config, environment, bank_short_name, masked_account_number, code_prefix, webhook_auth_method, webhook_url}'
