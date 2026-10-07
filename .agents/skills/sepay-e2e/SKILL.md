---
name: sepay-e2e
description: Test E2E luồng thanh toán SePay của quanly-phongtro trên SePay Test Mode — tạo hóa đơn test, gửi QR qua Zalo, giả lập giao dịch, kiểm tra webhook/PAID, đối soát và dọn dữ liệu test. Dùng khi người dùng nói "test sepay", "test thanh toán", "giả lập giao dịch", "gửi QR test", "kiểm tra webhook sepay", "đối soát sepay".
---

# SePay E2E (Test Mode)

Mọi thao tác chạy trên **production** `https://quanly.ptro.site` (cùng máy, service `quanly-phongtro-api`) nhưng tiền là **giả lập** trong SePay Test Mode. Script nằm trong `scripts/`, tất cả đều `source scripts/env.sh`.

## Quy tắc bắt buộc

- Credential ở `~/.config/quanly-phongtro/sepay-test.env` (chmod 600). **Không** `cat`, `echo`, in ra log, ghi vào repo/memory. Script tự nạp và đẩy output nhạy cảm vào `/dev/null`.
- Không trích xuất Webhook Secret / API key từ trang SePay để dán sang chỗ khác. Nếu cần cấu hình lại secret, nhờ người dùng tự nhập trong Settings → SePay của app.
- Hóa đơn test luôn dùng kỳ `2099-xx` cho phòng không có nhóm Zalo và tenant chưa liên kết Zalo, để chỉ manager nhận tin.
- Test xong **phải** chạy `cleanup.sh` (xoá hóa đơn, payment link, payment event, ảnh Zalo đã tạo). `cleanup.sh` từ chối xoá hóa đơn không thuộc kỳ `2099-xx`.
- Gửi Zalo là gửi tin thật tới người thật: chỉ gửi khi người dùng yêu cầu.

## Quy trình

```bash
S=.agents/skills/sepay-e2e/scripts
$S/sepay-login.sh                                   # mở playwright-cli session "sepay" (persistent), đăng nhập + vào Test Mode
$S/app-login.sh                                     # đăng nhập manager, in cấu hình SePay (đã mask)
$S/create-invoice.sh <room_id> 2099-01 1000000      # tạo hóa đơn đúng số tiền, ghi id vào file theo dõi
$S/send-zalo.sh <invoice_id>                        # gửi ảnh hóa đơn + QR qua Zalo, in payment link (mã PHxxxxxxxx)
$S/simulate.sh <payment_code> 1000000               # giả lập tiền vào trên SePay → SePay bắn webhook
$S/status.sh <invoice_id>                           # hóa đơn, payment link, payment_events + log API
$S/reconcile.sh ["2026-10-01 00:00:00" "2026-10-08 23:59:59"]
$S/cleanup.sh [invoice_id ...]                      # không truyền id → dọn mọi hóa đơn trong file theo dõi
```

Tìm phòng phù hợp (không nhóm Zalo, tenant chưa liên kết Zalo):

```sql
select r.id, r.name, h.name from rooms r join houses h on h.id = r.house_id
where h.manager_id = '<manager_id>' and coalesce(r.group_chat_id, '') = ''
  and not exists (select 1 from tenants t join users u on u.id = t.user_id
                  where t.room_id = r.id and coalesce(u.zalo_user_id, '') <> '');
```

## Kết quả mong đợi

| Bước | Dấu hiệu đúng |
|---|---|
| send-zalo | `{"success":true}` + 1 dòng `invoice_payment_links` provider `sepay`, `ACTIVE`, QR `qr.sepay.vn` |
| simulate | Bảng "Quy trình xử lý" nhận diện đúng mã `PH...`, "Gửi webhook" |
| status | log `POST .../sepay/managers/<id>/webhook ... 200`, invoice `PAID`, link `PAID`, event `PROCESSED` |
| reconcile | `processed` ≥ 0, không lỗi `sepay credentials not found` |

## Dữ liệu Test Mode hiện có

- Tài khoản nhận: Vietcombank `0000000001` "CONG TY TNHH TEST D838"; Vietcombank bắt buộc chọn VA (script tự chọn VA đầu tiên).
- Prefix mã thanh toán: `PH` (khớp `code_prefix` của manager).
- Webhook #2376 HMAC-SHA256 → `https://quanly.ptro.site/api/v1/payments/providers/sepay/managers/<manager_id>/webhook`.
- API key sandbox: `quanly-phongtro-local-env` (token lưu ở `~/.config/quanly-phongtro/sepay-sandbox-api-token`).
- Trang hữu ích: `/testmode/webhook#tab-logs` (lịch sử gửi webhook), `/testmode/transaction`, `/testmode/companyapi`.

## Xử lý lỗi thường gặp

- `sepay webhook signature mismatch`: secret HMAC lưu trong app khác secret của webhook trên SePay → người dùng nhập lại secret trong Settings → SePay. SePay tự gửi lại tối đa 7 lần khi server trả lỗi.
- `sepay webhook account mismatch`: số tài khoản trong cấu hình app khác `accountNumber` webhook gửi.
- `sepay credentials not found` khi đối soát: cấu hình chưa có API token.
- Không có payment link sau khi gửi Zalo: xem `status.sh`, tìm dòng `create preferred payment link failed`.
- `sepay-login.sh` báo chưa vào Test Mode: có thể dính captcha/OTP → `playwright-cli -s=sepay show` để người dùng tự xử lý.

Tài liệu SePay: https://docs.sepay.vn/test-mode.html, https://docs.sepay.vn/gia-lap-giao-dich.html, xác thực webhook: https://developer.sepay.vn/vi/sepay-webhooks/xac-thuc. Module tích hợp: `pkg/sepay/guide.md`.
