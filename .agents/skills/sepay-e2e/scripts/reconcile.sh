#!/usr/bin/env bash
. "$(dirname "$0")/env.sh"

if [[ $# -ge 2 ]]; then
  body="$(jq -n --arg from "$1" --arg to "$2" '{date_from: $from, date_to: $to}')"
else
  body='{}'
fi
api POST /payments/providers/sepay/reconcile --data "$body"
echo
