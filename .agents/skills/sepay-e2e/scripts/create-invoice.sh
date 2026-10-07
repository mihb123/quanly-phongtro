#!/usr/bin/env bash
. "$(dirname "$0")/env.sh"

if [[ $# -lt 3 ]]; then
  echo "Dùng: $0 <room_id> <period YYYY-MM> <amount VND>" >&2
  exit 1
fi
room_id="$1" period="$2" amount="$3"
require_uuid "$room_id"
[[ "$period" =~ ^[0-9]{4}-(0[1-9]|1[0-2])$ ]] || { echo "period phải dạng YYYY-MM" >&2; exit 1; }
[[ "$amount" =~ ^[0-9]+$ ]] || { echo "amount phải là số nguyên" >&2; exit 1; }

existing="$(db -At -c "select status from invoices where room_id='$room_id' and period='$period'")"
if [[ -n "$existing" ]]; then
  echo "Phòng đã có hóa đơn kỳ $period ($existing). Chọn kỳ khác (vd 2099-01) để không đụng dữ liệu thật." >&2
  exit 1
fi

create() {
  jq -n --arg room "$room_id" --arg period "$period" --argjson other "$1" '{
    room_id: $room, period: $period,
    old_electricity_index: 0, new_electricity_index: 0, old_water_index: 0, new_water_index: 0,
    other_fees: [{name: "Test SePay", amount: $other}],
    vehicle_count: 0, tenant_count: 1, exclude_room_fee: true
  }' | api POST /invoice/ --data @-
}

base="$(create 0 | jq -r '.data.total_amount // empty')"
[[ -n "$base" ]] || { echo "Không tạo được hóa đơn" >&2; exit 1; }
other=$(( amount - ${base%.*} ))
if (( other < 0 )); then
  echo "Phí cố định của nhà (${base%.*}) lớn hơn $amount; chọn phòng khác hoặc số tiền lớn hơn." >&2
  exit 1
fi

result="$(create "$other" | jq -c '.data | {id, period, status, total_amount}')"
echo "$result"
echo "$result" | jq -r .id >> "$TRACK_FILE"
