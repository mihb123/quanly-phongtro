#!/usr/bin/env bash
. "$(dirname "$0")/env.sh"

ids=("$@")
if [[ ${#ids[@]} -eq 0 && -s "$TRACK_FILE" ]]; then
  mapfile -t ids < <(sort -u "$TRACK_FILE")
fi
if [[ ${#ids[@]} -eq 0 ]]; then
  echo "Không có hóa đơn test nào để dọn." >&2
  exit 0
fi

for invoice_id in "${ids[@]}"; do
  require_uuid "$invoice_id"
  period="$(db -At -c "select period from invoices where id='$invoice_id'")"
  if [[ -n "$period" && "$period" != 2099-* ]]; then
    echo "Bỏ qua $invoice_id: kỳ $period không phải kỳ test 2099-xx" >&2
    continue
  fi
  db -At -c "delete from payment_events
             where invoice_id='$invoice_id'
                or provider_order_ref in (select provider_order_ref from invoice_payment_links where invoice_id='$invoice_id')" |
    sed "s/^/payment_events $invoice_id: /"
  if [[ -n "$period" ]]; then
    echo "invoice $invoice_id: HTTP $(api DELETE "/invoice/$invoice_id" -o /dev/null -w '%{http_code}')"
  fi
  if [[ -d "$UPLOADS_DIR" ]]; then
    find "$UPLOADS_DIR" -maxdepth 1 -type f -name "${invoice_id}_*" -print -delete | sed 's/^/xoá ảnh: /'
  fi
  if [[ -f "$TRACK_FILE" ]]; then
    sed -i "/^$invoice_id\$/d" "$TRACK_FILE"
  fi
done
