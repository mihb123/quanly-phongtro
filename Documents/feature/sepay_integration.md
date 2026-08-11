# SePay Integration

## Mục tiêu

Tính năng SePay cho phép manager cấu hình tài khoản nhận chuyển khoản, tạo QR chuyển khoản động cho hóa đơn và tự động đánh dấu hóa đơn `PAID` khi SePay gửi webhook hoặc khi hệ thống đối soát lại giao dịch qua SePay User API v2.

SePay được triển khai như một payment provider trong kiến trúc provider adapter hiện có, không có service payment riêng. Phần nghiệp vụ hóa đơn, audit event, idempotency, cập nhật trạng thái và thông báo Zalo vẫn nằm trong `PaymentService`.

## Phạm vi hiện tại

- Manager cấu hình SePay trong màn Settings.
- Form chỉ cho chọn các ngân hàng có `supported=true` trong danh sách VietQR được tài liệu SePay tham chiếu.
- Khi có API Token, backend xác thực tài khoản đã liên kết và lấy tên chủ tài khoản chuẩn từ SePay API v2 trước khi lưu.
- Backend lưu credential SePay theo từng manager, mã hóa trong DB.
- Khi gửi hóa đơn, hệ thống ưu tiên provider active theo thứ tự: SePay trước, PayOS sau.
- SePay tạo QR bằng URL `https://qr.sepay.vn/img`.
- Webhook SePay tự động xử lý giao dịch vào.
- Manager có thể bấm đối soát thủ công để kéo giao dịch từ SePay API v2.
- Event payment luôn được ghi vào `payment_events` để audit, kể cả unmatched hoặc already-paid.

## Code Map

Backend:

- `internal/service/payment/sepay_provider.go`: adapter SePay, tạo QR, xác thực webhook, map payload sang event chuẩn hóa.
- `internal/service/payment/sepay_client.go`: HTTP client gọi SePay User API v2.
- `internal/service/payment/sepay_reconciliation_service.go`: đối soát giao dịch qua API v2.
- `internal/service/payment/payment_service.go`: orchestration provider-neutral, xử lý link, webhook, event, invoice paid, notification.
- `internal/service/payment/payment_credential_service.go`: credential SePay, trạng thái config và chọn provider ưu tiên.
- `internal/handler/payment/payment_handler.go`: HTTP handler cho config, webhook và reconcile.
- `internal/router/router.go`: đăng ký route payment.
- `internal/model/invoice_payment.go`: model payment link, credential và event.
- `internal/repository/invoice/invoice_payment_repository.go`: persistence cho payment.
- `cmd/api/main.go`: đăng ký `NewSePayProvider`, `NewSePayClient`, `NewSePayReconciliationService`.

Frontend:

- `frontend/src/api/payment.tsx`: API client cho SePay config và reconcile.
- `frontend/src/components/home/settings/SePaySettingsCard.tsx`: card cấu hình SePay trong Settings.
- `frontend/src/components/home/SettingsView.tsx`: mount card SePay.
- `frontend/src/lib/sepay-banks.ts`: snapshot typed của danh sách ngân hàng VietQR được SePay hỗ trợ.
- `frontend/e2e/sepay.e2e.mjs`: Puppeteer E2E trên SePay Test Mode và ứng dụng local.
- `frontend/TESTING.md`: hướng dẫn chạy kiểm tra tĩnh và E2E frontend.

## Kiến trúc

SePay dùng chung interface:

```go
type PaymentProvider interface {
    Name() string
    CreatePaymentLink(ctx context.Context, input PaymentCreateInput) (*PaymentProviderLink, error)
    CancelPaymentLink(ctx context.Context, input PaymentCancelInput) error
    VerifyWebhook(ctx context.Context, input PaymentWebhookInput) (*VerifiedPaymentEvent, error)
}
```

Các trách nhiệm chính:

- `SePayProvider`: chỉ xử lý đặc thù SePay như QR URL, mã thanh toán, webhook auth và mapping payload.
- `PaymentCredentialService`: đọc/ghi credential SePay theo manager, mã hóa AES-256-GCM trong DB.
- `PaymentService`: xử lý nghiệp vụ chung: reuse active link, lưu payment event, idempotency, kiểm tra thiếu tiền, chuyển invoice sang `PAID`, mark link `PAID`, gửi thông báo Zalo best-effort.
- `SePayReconciliationService`: kéo giao dịch SePay API v2 và gọi lại `PaymentService.ProcessVerifiedTransaction`.

## Cấu hình SePay

Credential SePay được lưu trong `payment_provider_credentials.encrypted_credentials` dưới dạng JSON đã mã hóa. Schema typed ở backend:

```go
type SePayCredentials struct {
    Environment       string `json:"environment"`
    BankShortName     string `json:"bank_short_name"`
    AccountNumber     string `json:"account_number"`
    AccountName       string `json:"account_name"`
    CodePrefix        string `json:"code_prefix"`
    WebhookAuthMethod string `json:"webhook_auth_method"`
    WebhookAPIKey     string `json:"webhook_api_key"`
    WebhookSecret     string `json:"webhook_secret"`
    APIToken          string `json:"api_token"`
}
```

Ý nghĩa field:

- `environment`: `production` hoặc `sandbox`; credential cũ không có field này mặc định dùng production.
- `bank_short_name`: tên ngắn ngân hàng truyền vào QR SePay, được chọn từ danh sách `supported=true`, ví dụ `MBBank`.
- `account_number`: số tài khoản nhận tiền.
- `account_name`: tên chủ tài khoản, dùng cho cấu hình/quản trị.
- `code_prefix`: tiền tố mã thanh toán, phải khớp prefix đã cấu hình trên SePay dashboard để webhook trả về `code`.
- `webhook_auth_method`: `apikey`, `hmac` hoặc `none`.
- `webhook_api_key`: secret dùng với `Authorization: Apikey <key>`.
- `webhook_secret`: secret dùng để verify HMAC.
- `api_token`: Bearer token cho SePay User API v2, gửi bằng `Authorization: Bearer <token>` và chỉ bắt buộc khi chạy đối soát.

Khi lưu hoặc xóa cấu hình SePay, repository đánh dấu các payment link SePay đang `ACTIVE` của manager thành `STALE`. Việc này buộc hệ thống tạo QR mới theo credential mới, tránh tiếp tục dùng QR cũ.

## API

Manager APIs, yêu cầu auth và role `MANAGER`:

- `GET /api/v1/payments/public-key`
- `GET /api/v1/payments/providers/sepay/config`
- `POST /api/v1/payments/providers/sepay/config`
- `DELETE /api/v1/payments/providers/sepay/config`
- `POST /api/v1/payments/providers/sepay/reconcile`

Webhook public:

- `POST /api/v1/payments/providers/sepay/managers/{managerID}/webhook`

`GET /config` trả về trạng thái đã mask:

```json
{
  "has_config": true,
  "is_active": true,
  "provider": "sepay",
  "environment": "sandbox",
  "masked_account_number": "****1234",
  "bank_short_name": "MBBank",
  "code_prefix": "PT",
  "webhook_auth_method": "apikey",
  "webhook_url": "https://example.com/api/v1/payments/providers/sepay/managers/{managerID}/webhook"
}
```

`POST /config` nhận field thường cho thông tin ngân hàng và field secret đã RSA-encrypt base64 cho `webhook_api_key`, `webhook_secret`, `api_token`. Backend decrypt bằng private key runtime trong `PaymentHandler`, rồi mã hóa lại bằng app secret trước khi lưu DB.

Nếu request có `api_token`, handler gọi `GET /v2/bank-accounts` trên đúng environment, lọc và khớp chính xác `bank_short_name + account_number`. Tài khoản không thuộc công ty của token sẽ bị từ chối; tài khoản hợp lệ được lưu với `account_holder_name` chuẩn do SePay trả về. Response thành công cho biết kết quả xác thực:

```json
{
  "success": true,
  "bank_account_verified": true,
  "account_holder_name": "CONG TY TNHH DEMO"
}
```

Không có API Token thì cấu hình vẫn được lưu với `bank_account_verified=false`. SePay không công bố API tra cứu tên cho một tài khoản bất kỳ; endpoint này chỉ đọc tài khoản đã liên kết với công ty sở hữu token.

`POST /reconcile` nhận body tùy chọn:

```json
{
  "date_from": "2026-07-01 00:00:00",
  "date_to": "2026-07-07 23:59:59"
}
```

Nếu không truyền window, backend mặc định đối soát 7 ngày gần nhất. Response backend:

```json
{
  "pages_fetched": 1,
  "scanned": 25,
  "processed": 3,
  "failed": 0,
  "truncated": false
}
```

Frontend hiện hiển thị `scanned`, `processed` và cảnh báo `truncated`.

Client chọn base URL theo `environment`: production dùng `https://userapi.sepay.vn/v2`, Test Mode dùng `https://userapi-sandbox.sepay.vn/v2`. Host được chọn từ enum cố định, không nhận URL tùy ý từ người dùng.

## Luồng Tạo QR

1. Luồng gửi hóa đơn gọi `PaymentService.CreatePreferredPaymentLinkForInvoice`.
2. `PaymentCredentialService.ResolvePreferredProvider` chọn SePay nếu manager có credential SePay active; nếu không có thì dùng PayOS nếu available.
3. `PaymentService.CreatePaymentLinkForInvoice` kiểm tra active link theo `invoice_id + provider`.
4. Nếu active link cùng amount còn tồn tại, service reuse link.
5. Nếu amount đổi, link cũ bị mark `STALE`.
6. `SePayProvider.CreatePaymentLink` tạo mã thanh toán và QR URL.
7. `PaymentService` lưu record vào `invoice_payment_links` với `provider = "sepay"`.

QR URL có dạng:

```text
https://qr.sepay.vn/img?acc={account_number}&bank={bank_short_name}&amount={amount}&des={payment_code}
```

Mã thanh toán được build từ `code_prefix + invoice_id`, uppercase, bỏ ký tự không thuộc `[A-Z0-9]`, chỉ lấy tối đa 8 ký tự đầu của invoice ID sau khi bỏ dấu `-`. Nếu trùng `provider_order_ref`, hệ thống thêm suffix attempt cho tới khi không còn trùng.

## Luồng Webhook

1. SePay POST webhook đến URL có `managerID`.
2. `PaymentHandler.handleWebhook` đọc raw body tối đa 1 MB và truyền cả header sang `PaymentService.HandleWebhook`.
3. `PaymentService` load credential SePay theo manager.
4. `SePayProvider.VerifyWebhook` xác thực request theo `webhook_auth_method`.
5. Adapter parse payload SePay, bỏ qua giao dịch không phải tiền vào hoặc không có `code`, và từ chối giao dịch gửi cho tài khoản khác cấu hình.
6. Adapter map payload sang `VerifiedPaymentEvent`.
7. `PaymentService.ProcessVerifiedTransaction` xử lý idempotency, ghi event, match payment link và settle invoice nếu đủ điều kiện.
8. Handler trả HTTP 200 body `{"success": true}` khi xử lý thành công hoặc webhook bị ignore hợp lệ.

Payload SePay webhook đang được map:

- `id`: transaction reference cho webhook, dùng dedup trong cùng nguồn webhook.
- `code`: `provider_order_ref`, dùng để match `invoice_payment_links.provider_order_ref`.
- `transferType`: chỉ xử lý `in`.
- `transferAmount`: amount nhận được.
- `accountNumber`: lưu vào `counter_account`.
- Raw body được lưu vào `payment_events.raw_payload`.

Với HMAC, server ký/kiểm tra chuỗi `{timestamp}.{raw_body}` và từ chối timestamp lệch quá 5 phút để hạn chế replay.

Auth webhook:

- `apikey`: yêu cầu header `Authorization: Apikey <key>`, so sánh constant-time với `webhook_api_key`.
- `hmac`: yêu cầu `X-SePay-Timestamp` và `X-SePay-Signature`; chữ ký là HMAC-SHA256 trên chuỗi `{timestamp}.{raw_body}`.
- `none` hoặc rỗng: chấp nhận webhook và ghi log cảnh báo.

## Luồng Đối Soát API v2

Đối soát dùng để xử lý trường hợp webhook bị miss hoặc cần audit lại giao dịch.

1. Frontend gọi `POST /api/v1/payments/providers/sepay/reconcile`.
2. Handler lấy `managerID`, default date window là 7 ngày gần nhất nếu body rỗng.
3. `SePayReconciliationService` load credential SePay và yêu cầu `api_token`.
4. `SePayClient.ListTransactions` gọi `GET https://userapi.sepay.vn/v2/transactions`.
5. Mỗi page dùng `per_page = 100`; giữa các page throttle 350 ms để giữ dưới limit 3 req/s.
6. Service dừng khi `meta.pagination.has_more = false` hoặc chạm cap 100 page.
7. Mỗi transaction `transfer_type = in`, `amount_in > 0`, có `code` sẽ được map thành `VerifiedPaymentEvent`.
8. Event được đưa vào `PaymentService.ProcessVerifiedTransaction`, dùng chung logic settle với webhook.

Điểm quan trọng về dedup:

- Webhook SePay dùng `id` dạng integer.
- API v2 dùng `id` dạng UUID.
- Cùng một giao dịch ngân hàng có thể có transaction reference khác nhau giữa webhook và API v2.
- Vì vậy dedup chéo không dựa hoàn toàn vào transaction reference; `ProcessVerifiedTransaction` còn kiểm tra payment link đã `PAID` để chỉ ghi audit, không settle/notify lại.

## Xử Lý Invoice Và Event

`PaymentService.ProcessVerifiedTransaction` xử lý theo thứ tự:

1. Kiểm tra event đã tồn tại bằng `provider + provider_order_ref + transaction_reference`.
2. Tìm payment link theo `managerID + provider + provider_order_ref`.
3. Ghi `payment_events`.
4. Nếu không tìm thấy link, event có trạng thái `UNMATCHED`, không cập nhật invoice.
5. Nếu link đã `PAID`, chỉ giữ audit event, không xử lý lại invoice.
6. Nếu amount nhận được nhỏ hơn amount trên link, ghi event nhưng không mark paid.
7. Nếu hợp lệ, invoice được update sang `PAID`, payment method theo provider, link chuyển `PAID`.
8. Gửi thông báo Zalo best-effort cho tenant và manager nếu có Zalo user ID.

## Database

Bảng `payment_provider_credentials`:

- Lưu credential theo `manager_id + provider`.
- `encrypted_credentials` chứa JSON credential đã mã hóa.
- `is_active` cho biết credential còn dùng hay không.
- Upsert theo unique `(manager_id, provider)`.

Bảng `invoice_payment_links`:

- `provider = "sepay"`.
- `provider_order_ref` là mã thanh toán trong nội dung chuyển khoản.
- `payment_link_id` và `qr_code` đều dùng QR/payment code SePay.
- `amount` là số tiền VND dạng integer.
- `status`: `ACTIVE`, `STALE`, `CANCELLED`, `PAID`.

Bảng `payment_events`:

- Ghi mọi giao dịch đã verify hoặc đã map qua reconcile.
- `transaction_reference` là webhook integer id hoặc API v2 UUID.
- `matching_method` là `ORDER_CODE` nếu match link, `UNMATCHED` nếu không match.
- `status` là `PROCESSED` khi có invoice, `UNMATCHED` khi không tìm được link.
- `raw_payload` lưu raw webhook body hoặc JSON transaction API v2.

## Frontend

`SePaySettingsCard` cung cấp:

- Trạng thái đã cấu hình/chưa cấu hình.
- Webhook URL kèm nút copy.
- Form cấu hình ngân hàng, số tài khoản, chủ tài khoản, code prefix.
- Chọn auth method: API Key, HMAC hoặc none.
- Nhập secret webhook và API token với nút ẩn/hiện.
- Lưu cấu hình qua RSA public key transport.
- Xóa cấu hình.
- Bấm đối soát thủ công.

Lưu ý vận hành UI: khi cập nhật config, nếu chọn `apikey` hoặc `hmac`, manager cần nhập lại secret tương ứng vì backend không trả plaintext secret về frontend.

## Bảo mật

- Frontend lấy public key từ `GET /api/v1/payments/public-key`.
- Secret gửi lên được encrypt RSA-OAEP SHA-256 và base64.
- Backend decrypt bằng private key runtime của `PaymentHandler`.
- Credential lưu DB tiếp tục được encrypt bằng app-level AES key.
- Webhook auth nên dùng `apikey` hoặc `hmac`; `none` chỉ phù hợp môi trường kiểm thử hoặc khi có lớp bảo vệ khác.
- Webhook body bị giới hạn 1 MB bằng `http.MaxBytesReader`.
- HMAC ký raw body, không ký JSON đã parse lại.

## Vận hành

Checklist cấu hình SePay:

1. Manager cấu hình `bank_short_name`, `account_number`, `account_name`, `code_prefix`.
2. `code_prefix` phải trùng prefix trên SePay dashboard để SePay extract được `code`.
3. Copy webhook URL trong Settings sang SePay dashboard.
4. Chọn auth method và cấu hình secret tương ứng ở cả hệ thống lẫn SePay.
5. Nếu cần đối soát, cấu hình thêm `api_token`.
6. Tạo/gửi thử một hóa đơn mới để sinh QR SePay.
7. Chuyển khoản với đúng nội dung QR, kiểm tra invoice chuyển `PAID`.
8. Nếu webhook không chạy, dùng nút đối soát và kiểm tra `payment_events`.

Các lỗi thường gặp:

- Webhook trả `payment credentials not found`: manager chưa có credential SePay active hoặc webhook URL sai `managerID`.
- Giao dịch không settle: webhook/API v2 không có `code`, prefix SePay dashboard không khớp `code_prefix`, hoặc nội dung chuyển khoản bị sửa.
- Event `UNMATCHED`: không tìm thấy active/stored payment link theo `provider_order_ref`.
- Underpaid: amount nhận được nhỏ hơn `invoice_payment_links.amount`, event được ghi nhưng invoice không chuyển `PAID`.
- Đối soát trả lỗi thiếu credential: thiếu `api_token`.
- Reconcile `truncated = true`: một lần chạy chạm cap 100 page; cần chạy lại với date window hẹp hơn.

Không chạy migration/build/deploy nếu chỉ cập nhật tài liệu.
