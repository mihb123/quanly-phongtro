# SePay Integration — Session Handoff Summary

> Bản compact ngữ cảnh phiên tích hợp SePay (Claude điều phối + agy thực thi). Dùng để nạp lại ngữ cảnh ở phiên sau mà không cần đọc lại transcript. Plan gốc: `Documents/plan/SEPAY_INTEGRATION_PLAN.md`.

## 1. Mục tiêu & bối cảnh
- Dự án `quanly-phongtro` (Go backend + React frontend). Tích hợp SePay: tạo QR chuyển khoản động cho hóa đơn, tự động `PAID` qua webhook.
- Ưu tiên QR: SePay active → PayOS active → gửi hóa đơn không QR. KHÔNG fallback âm thầm khi provider đã chọn lỗi cấu hình.
- Tái dùng kiến trúc provider adapter: `PaymentProvider`, `PaymentCredentialService`, `InvoicePaymentRepository`; bảng `payment_provider_credentials`/`invoice_payment_links`/`payment_events`; route webhook generic `/api/v1/payments/providers/{provider}/managers/{managerID}/webhook`.

## 2. Sự thật kỹ thuật SePay đã verify (nguồn sự thật)
- **QR**: `https://qr.sepay.vn/img?acc=&bank=&amount=&des=` (des = mã thanh toán gồm prefix).
- **Webhook auth (4 cách)**: API Key `Authorization: Apikey <key>` (mặc định) | HMAC-SHA256 ký `{timestamp}.{raw_body}`, header `X-SePay-Signature: sha256={hex}` + `X-SePay-Timestamp` | OAuth | none.
- **Webhook payload**: `id` (integer) = dedup key bất biến qua retry; `transferType` in/out; `transferAmount` (dương); `code` (nullable, mã thanh toán theo prefix dashboard); `referenceCode`.
- **Webhook response bắt buộc**: HTTP 200/201 + body `{"success": true}` trong 30s.
- **API v2**: base `https://userapi.sepay.vn/v2`, `Authorization: Bearer <api_token>`, `GET /v2/transactions` (`page`/`per_page≤100`, `transaction_date_from/to`). Response `{status, data[], meta.pagination{has_more}}`. Transaction `id` là **UUID** (KHÁC webhook integer id), `amount_in`/`amount_out`, `code`, `transfer_type`. Rate limit **3 req/s**.

## 3. Quyết định kiến trúc đã chốt
- `SePayProvider` (adapter mới), không tạo luồng payment riêng.
- Credential JSON tách 3 secret: `webhook_api_key` (apikey) / `webhook_secret` (hmac) / `api_token` (API v2). Bắt buộc: `bank_short_name`, `account_number`, `account_name`, `code_prefix` (phải trùng prefix dashboard).
- Mã thanh toán uppercase toàn bộ (chống prefix chữ thường bị filter → mã `null`).
- **Lõi dùng chung `ProcessVerifiedTransaction`** (webhook + reconciliation) trong `payment_service.go`.
- **Guard cross-source dedup**: v2-id (UUID) ≠ webhook-id (int) → không dedup chéo bằng transaction ref; thay vào đó nếu payment link đã `PAID` thì chỉ ghi audit, KHÔNG settle/notify lại.

## 4. Đã hoàn thành (build/vet/test ✅)
**Backend P1**: constants `PaymentProviderSePay`/`PaymentMethodSePay`; refactor `PaymentCredentialService` multi-provider + `ResolvePreferredProvider`; `SePayCredentials`; config endpoints `GET/POST/DELETE /api/v1/payments/providers/sepay/config`; `SePayProvider.CreatePaymentLink` (QR); `VerifyWebhook` apikey/HMAC constant-time + `PaymentWebhookInput.Headers` + đổi chữ ký `HandleWebhook(...,headers)`; amount guard.
**Frontend**: `frontend/src/components/home/settings/SePaySettingsCard.tsx` (form cấu hình + nút "Đối soát giao dịch" gọi `reconcileSePay`), `SettingsView.tsx`, `frontend/src/api/payment.tsx` (config APIs + `reconcileSePay`/`SePayReconcileResult`).
**Backend P2 (reconciliation)**: `internal/service/sepay_client.go` (`SePayClient.ListTransactions`), `internal/service/sepay_reconciliation_service.go` (`ReconcileManager`, throttle 350ms, cap 100 trang), endpoint `POST /api/v1/payments/providers/sepay/reconcile` (setter `SetSePayReconciler` ở `PaymentHandler`).
**Tests**: `sepay_provider_test.go`, `sepay_reconciliation_service_test.go` (11/11 pass). Frontend `tsc -b` + lint pass.

## 5. Phân vai & lỗi agy đã sửa
- agy: boilerplate backend P1 + card React. Claude: verify doc (WebFetch), crypto webhook, service đối soát, toàn bộ tests, review/sửa diff.
- Claude sửa lỗi agy: (a) agy xóa nhầm khai báo `interface PaymentCredentialService` + code mồ côi → vỡ build; (b) shadow const + uppercase prefix; (c) frontend `any` → `SePayConfigPayload`.

## 6. Còn dang dở / next steps
- [x] Nút "Đối soát giao dịch" trong `SePaySettingsCard` → gọi `POST /providers/sepay/reconcile` (xong, tsc/lint pass).
- [ ] Cron tự động gọi `ReconcileManager` cho manager có `api_token`.
- [ ] Manual E2E với tài khoản SePay thật (ngrok/dev webhook).
- [ ] **Chưa git commit/push** (nhánh `develop`). Lệnh gợi ý:
  ```bash
  git add -A && git commit -m "feat(payment): tích hợp cổng SePay (QR + webhook + API v2 đối soát)" && git push origin develop
  ```

## 7. Kết quả Code Review (agy)
Dựa trên skill `review-dif`, dưới đây là các lỗi logic/kiến trúc cần lưu ý (chưa tự động fix):
1. **Lỗi parse JSON bị ẩn (swallow error) trong `ReconcileSePay`**: Ở `internal/handler/payment_handler.go`, kết quả của `json.NewDecoder(r.Body).Decode(&req)` bị bỏ qua `_ =`. Nếu body truyền lên là JSON không hợp lệ, lỗi sẽ bị bỏ qua và API vẫn chạy tiếp với giá trị mặc định thay vì trả về `400 Bad Request`. Cần xử lý lỗi tường minh (bỏ qua `io.EOF`).
2. **Vòng lặp đối soát bị gián đoạn toàn bộ nếu 1 giao dịch lỗi**: Ở `internal/service/sepay_reconciliation_service.go`, hàm `ReconcileManager` sẽ return và dừng batch sync ngay lập tức nếu `processTransaction` trả về lỗi. Nên log lỗi này và dùng `continue` để xử lý tiếp các giao dịch khác, tránh hỏng cả một phiên đối soát chỉ vì 1 giao dịch lỗi.
