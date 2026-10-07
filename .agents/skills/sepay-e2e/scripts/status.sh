#!/usr/bin/env bash
. "$(dirname "$0")/env.sh"

invoice_id="${1:?Dùng: $0 <invoice_id> [phút log, mặc định 10]}"
minutes="${2:-10}"
require_uuid "$invoice_id"
[[ "$minutes" =~ ^[0-9]+$ ]] || { echo "phút phải là số" >&2; exit 1; }

db -c "select id, period, total_amount, status, payment_method from invoices where id='$invoice_id'"
db -c "select provider, provider_order_ref, amount, status, updated_at from invoice_payment_links where invoice_id='$invoice_id' order by created_at"
db -c "select e.transaction_reference, e.amount, e.signature_result, e.matching_method, e.status, e.created_at
       from payment_events e
       where e.invoice_id='$invoice_id'
          or e.provider_order_ref in (select provider_order_ref from invoice_payment_links where invoice_id='$invoice_id')
       order by e.created_at"
echo "--- log API ($minutes phút gần nhất, chỉ dòng liên quan) ---"
journalctl -u quanly-phongtro-api --since "-${minutes} min" --no-pager 2>/dev/null |
  grep -iE "sepay|payment|ERROR|zalo/invoices" | grep -v "GET .*/uploads/" | cut -c1-260 | tail -20
