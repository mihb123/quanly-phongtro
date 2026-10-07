#!/usr/bin/env bash
. "$(dirname "$0")/env.sh"

invoice_id="${1:?Dùng: $0 <invoice_id>}"
require_uuid "$invoice_id"

api POST "/zalo/invoices/$invoice_id/send"
echo
db -c "select provider, provider_order_ref, amount, status, qr_code from invoice_payment_links where invoice_id='$invoice_id' order by created_at"
