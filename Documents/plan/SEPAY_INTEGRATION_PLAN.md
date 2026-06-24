# Kế hoạch Tích hợp SePay

## Mục tiêu

Tích hợp SePay vào hệ thống payment provider hiện có để manager có thể cấu hình thông tin SePay riêng, tạo QR chuyển khoản cho hóa đơn, nhận webhook giao dịch tiền vào và tự động chuyển hóa đơn sang `PAID`.

Luồng tạo QR cần ưu tiên provider theo cấu hình của manager:

1. Nếu manager có cấu hình SePay đang active, tạo QR SePay.
2. Nếu manager không có SePay active nhưng có PayOS active, tạo QR PayOS.
3. Nếu không có provider active, không tạo QR, chỉ gửi hóa đơn như hiện tại và log lý do.

Không fallback sang PayOS khi SePay đã active nhưng tạo QR/webhook bị lỗi do cấu hình sai. Trường hợp này cần báo lỗi/log rõ để manager sửa SePay, tránh tạo QR của provider khác ngoài ý muốn.

## Tài liệu tham khảo

- SePay Cổng thanh toán: https://developer.sepay.vn/vi/cong-thanh-toan/gioi-thieu
- SePay Webhooks: https://developer.sepay.vn/vi/sepay-webhooks
- SePay API v2: https://developer.sepay.vn/vi/sepay-api/v2/gioi-thieu
- Tài liệu workflow hiện có: `Documents/feature/payment_gateway_integration.md`

## Kiến trúc hiện có cần tái sử dụng

Hệ thống đã có kiến trúc provider adapter:

- `PaymentService`: điều phối nghiệp vụ payment trung lập provider.
- `PaymentProvider`: adapter interface cho từng provider.
- `PayOSProvider`: adapter PayOS hiện có.
- `PaymentCredentialService`: lưu credential manager theo provider.
- `InvoicePaymentRepository`: lưu payment links, credentials và webhook events.
- Route webhook generic: `POST /api/v1/payments/providers/{provider}/managers/{managerID}/webhook`.
- Bảng DB hiện có:
  - `payment_provider_credentials`
  - `invoice_payment_links`
  - `payment_events`

Vì vậy SePay nên được thêm như provider mới, không tạo luồng payment riêng.

## Quyết định thiết kế

### Phase 1: QR chuyển khoản + webhook SePay

Đây là luồng phù hợp nhất với nhu cầu hiện tại:

1. Hệ thống sinh mã thanh toán riêng cho hóa đơn.
2. Hệ thống tạo QR chuyển khoản SePay/VietQR với:
   - ngân hàng nhận tiền
   - số tài khoản nhận tiền
   - số tiền hóa đơn
   - nội dung chuyển khoản chứa mã thanh toán
3. Tenant chuyển khoản.
4. SePay nhận giao dịch từ ngân hàng và gọi webhook về backend.
5. Backend verify webhook, match mã thanh toán với `invoice_payment_links.provider_order_ref`.
6. Backend ghi `payment_events`, mark invoice `PAID`, mark payment link `PAID`, gửi thông báo best-effort.

### Phase sau: SePay checkout/cổng thanh toán

Chỉ thêm sau nếu cần redirect tenant qua trang checkout SePay. Phase 1 không cần vì workflow chính là chuyển khoản ngân hàng và tự động nhận diện.

### API v2 chỉ dùng cho đối soát

SePay API v2 nên dùng để:

- kéo lại giao dịch bị miss webhook
- đối soát theo khoảng thời gian
- debug các giao dịch unmatched

Không dùng API v2 thay webhook realtime trong phase 1.

## Data model

### Constants

Thêm vào `internal/model/invoice_payment.go`:

```go
const (
    PaymentProviderSePay = "sepay"
)
```

Thêm vào `internal/model/invoice.go`:

```go
const (
    PaymentMethodSePay = "SEPAY"
)
```

Cập nhật `paymentMethodForProvider`:

```go
func paymentMethodForProvider(provider string) string {
    switch provider {
    case model.PaymentProviderPayOS:
        return model.PaymentMethodPayOS
    case model.PaymentProviderSePay:
        return model.PaymentMethodSePay
    default:
        return strings.ToUpper(provider)
    }
}
```

### SePay credential schema

Lưu trong `payment_provider_credentials.encrypted_credentials` dạng JSON đã mã hóa:

```json
{
  "bank_short_name": "MBBank",
  "account_number": "123456789",
  "account_name": "NGUYEN VAN A",
  "webhook_secret": "secret-used-for-hmac",
  "code_prefix": "PT",
  "api_token": "optional-for-reconciliation"
}
```

Field bắt buộc:

- `bank_short_name`
- `account_number`
- `account_name`
- `webhook_secret`
- `code_prefix`

Field tùy chọn:

- `api_token`: chỉ cần nếu làm đối soát bằng API v2.

Không cần thêm bảng DB mới cho phase 1. Unique `(manager_id, provider)` trong `payment_provider_credentials` đã đủ.

## Backend plan

### 1. Refactor `PaymentCredentialService` để hỗ trợ nhiều provider

Hiện tại `GetCredentials` đang chỉ chấp nhận PayOS. Cần mở rộng:

- `GetCredentials(ctx, managerID, provider)` trả credentials cho `payos` hoặc `sepay`.
- Giữ fallback app-level PayOS như hiện tại để không phá compatibility cũ.
- Không tạo fallback app-level SePay nếu mỗi manager phải tự cấu hình tài khoản nhận tiền riêng.

Thêm method config riêng cho SePay:

```go
GetSePayConfig(ctx context.Context, managerID, appURL string) (SePayConfigStatus, error)
SaveSePayConfig(ctx context.Context, managerID string, credentials SePayCredentials) error
DeleteSePayConfig(ctx context.Context, managerID string) error
ResolvePreferredProvider(ctx context.Context, managerID string) (string, error)
```

`ResolvePreferredProvider` áp dụng rule:

1. Nếu có active credential `sepay`, return `sepay`.
2. Nếu có active credential `payos` hoặc PayOS fallback env hợp lệ, return `payos`.
3. Nếu không có provider nào, return `ErrPaymentCredentialsNotFound`.

### 2. Thêm provider selection vào `PaymentService`

Hiện `CreatePaymentLinkForInvoice` yêu cầu caller truyền provider. Để tránh hard-code provider trong Zalo delivery, thêm method mới:

```go
CreatePreferredPaymentLinkForInvoice(
    ctx context.Context,
    managerID string,
    invoice *model.InvoiceWithRoom,
    tenantName string,
) (*model.InvoicePaymentLink, error)
```

Implementation:

1. Gọi `credentialService.ResolvePreferredProvider(ctx, managerID)`.
2. Nếu provider là `sepay`, gọi lại `CreatePaymentLinkForInvoice(ctx, managerID, "sepay", invoice, tenantName)`.
3. Nếu provider là `payos`, gọi lại `CreatePaymentLinkForInvoice(ctx, managerID, "payos", invoice, tenantName)`.
4. Nếu không có credential, trả `ErrPaymentCredentialsNotFound`.

Không nên viết logic ưu tiên provider trong `zalo_invoice_delivery.go`, vì file delivery chỉ nên lo việc gửi hóa đơn/QR qua Zalo.

### 3. Tạo `SePayProvider`

Thêm file `internal/service/sepay_provider.go` implement `PaymentProvider`.

#### `Name`

Trả về `model.PaymentProviderSePay`.

#### `CreatePaymentLink`

SePay phase 1 không cần gọi API ngoài. Adapter chỉ tạo thông tin QR:

1. Validate invoice amount > 0.
2. Validate SePay credentials có đủ bank/account/prefix.
3. Sinh payment code unique, ví dụ:
   - `PT` + short code từ invoice ID
   - nếu trùng thì thêm suffix attempt
4. Dùng `ProviderOrderRefExists` để đảm bảo code chưa tồn tại.
5. Tạo QR URL:

```text
https://qr.sepay.vn/img?acc={account_number}&bank={bank_short_name}&amount={amount}&des={payment_code}
```

6. Trả `PaymentProviderLink`:

```go
PaymentProviderLink{
    ProviderOrderRef: paymentCode,
    PaymentLinkID:    paymentCode,
    CheckoutURL:      qrURL,
    QRCode:           qrURL,
    Amount:           amount,
}
```

`OrderCode` có thể để `0` vì PayOS mới cần numeric order code.

#### `CancelPaymentLink`

SePay QR chuyển khoản không có remote payment link để hủy. Adapter có thể no-op và để `PaymentService.CancelPaymentLink` mark local link `CANCELLED`.

#### `VerifyWebhook`

Cần parse raw body SePay webhook và map sang `VerifiedPaymentEvent`.

Cần thay đổi `PaymentWebhookInput` để có header:

```go
type PaymentWebhookInput struct {
    Body        []byte
    Headers     http.Header
    Credentials map[string]string
}
```

Handler truyền `r.Header` vào service/provider.

Verify:

1. Parse JSON body.
2. Verify HMAC theo SePay docs nếu webhook secret được cấu hình.
3. Chỉ xử lý giao dịch tiền vào.
4. Nếu payload không có mã thanh toán (`code`) thì return `ErrPaymentWebhookIgnored` hoặc tạo event unmatched tùy policy.
5. Map event:

```go
VerifiedPaymentEvent{
    ProviderOrderRef:     webhook.Code,
    Amount:               webhook.TransferAmount,
    TransactionReference: webhook.ID or webhook.ReferenceCode,
    PayerAccount:         optional payer account,
    CounterAccount:       receiving account,
    RawPayload:           string(input.Body),
    SignatureResult:      paymentSignatureValid,
    MatchingMethod:       paymentMatchOrderRef,
}
```

### 4. Cập nhật webhook handler

Trong `internal/handler/payment_handler.go`:

- `handleWebhook` đọc raw body như hiện tại.
- Truyền header vào `PaymentService.HandleWebhook`.
- Sửa response success thành JSON hợp lệ:

```json
{"success": true}
```

Nếu SePay yêu cầu HTTP `200` hoặc `201`, giữ `200 OK` là đủ.

### 5. Cập nhật `PaymentService.HandleWebhook`

Thêm guard trước khi mark paid:

1. Tìm `paymentLink` theo `managerID + provider + provider_order_ref`.
2. Nếu không tìm thấy, ghi event `UNMATCHED`, không update invoice.
3. Nếu `verifiedEvent.Amount < paymentLink.Amount`, ghi event nhưng không mark `PAID`.
4. Nếu amount đủ, mark invoice `PAID`.
5. Mark payment link `PAID`.
6. Gửi thông báo tenant/manager.

Cần đảm bảo idempotency bằng unique hiện có:

```text
provider + provider_order_ref + transaction_reference
```

### 6. Cập nhật Zalo invoice delivery

Hiện tại `internal/service/zalo_invoice_delivery.go` hard-code PayOS:

```go
CreatePaymentLinkForInvoice(ctx, managerID, model.PaymentProviderPayOS, invoice, mainTenantName)
```

Cần đổi sang:

```go
CreatePreferredPaymentLinkForInvoice(ctx, managerID, invoice, mainTenantName)
```

Behavior mong muốn:

1. Manager có SePay active: tạo QR SePay và gửi ảnh QR SePay.
2. Manager không có SePay, có PayOS: tạo QR PayOS và gửi ảnh QR PayOS.
3. Không có provider: vẫn gửi ảnh hóa đơn, không gửi QR.
4. Nếu provider selected bị lỗi cấu hình: log lỗi, không fallback âm thầm sang provider khác.

### 7. Routes backend

Thêm route config vào group authenticated manager:

```text
GET    /api/v1/payments/providers/sepay/config
POST   /api/v1/payments/providers/sepay/config
DELETE /api/v1/payments/providers/sepay/config
```

Webhook dùng route generic đã có:

```text
POST /api/v1/payments/providers/sepay/managers/{managerID}/webhook
```

Response `GET /config` nên trả:

```json
{
  "has_config": true,
  "is_active": true,
  "provider": "sepay",
  "masked_account_number": "****6789",
  "bank_short_name": "MBBank",
  "code_prefix": "PT",
  "webhook_url": "https://domain/api/v1/payments/providers/sepay/managers/{managerID}/webhook"
}
```

## Frontend plan

### 1. API layer

Cập nhật `frontend/src/api/payment.tsx`:

- `SePayConfigPayload`
- `SePayConfigStatus`
- `getSePayConfig`
- `saveSePayConfig`
- `deleteSePayConfig`

Tất cả HTTP calls vẫn đi qua `apiClient`.

### 2. Settings UI

Thêm `frontend/src/components/home/settings/SePaySettingsCard.tsx`.

Fields:

- Bank short name
- Account number
- Account name
- Webhook secret
- Code prefix
- API token optional

UI behavior:

- Dùng React Hook Form + Zod như PayOS card.
- Dùng `/payments/public-key` để RSA encrypt secret fields trước khi save.
- Hiện webhook URL để manager copy vào SePay dashboard.
- Hiện status "SePay đang được ưu tiên tạo QR" khi config active.
- Nếu SePay chưa có nhưng PayOS có, PayOS vẫn là provider tạo QR.

Cập nhật `frontend/src/components/home/SettingsView.tsx` để render:

1. Zalo settings
2. SePay settings
3. PayOS settings

Đặt SePay trước PayOS để đúng với rule ưu tiên.

## Dashboard SePay setup cho manager

Trong SePay dashboard, manager cần:

1. Kết nối tài khoản ngân hàng nhận tiền.
2. Cấu hình mã thanh toán tự động với prefix trùng với `code_prefix`, ví dụ `PT`.
3. Tạo webhook tiền vào.
4. Gắn webhook URL:

```text
https://domain/api/v1/payments/providers/sepay/managers/{managerID}/webhook
```

5. Bật HMAC/webhook secret nếu SePay dashboard hỗ trợ.
6. Nếu có option lọc giao dịch không có mã thanh toán, bật lọc để giảm unmatched events.

## Đối soát bằng API v2

Phase 1 có thể chưa cần. Nếu làm thêm:

1. Lưu `api_token` trong SePay credentials.
2. Thêm service `SePayReconciliationService`.
3. Kéo transactions theo thời gian hoặc `since_id`.
4. Reuse cùng logic match `provider_order_ref`.
5. Chỉ xử lý giao dịch chưa tồn tại trong `payment_events`.

Cần throttle request theo rate limit của SePay API v2.

## Test plan

### Backend unit tests

- `ResolvePreferredProvider`:
  - SePay active + PayOS active -> `sepay`
  - SePay inactive + PayOS active -> `payos`
  - only PayOS fallback env -> `payos`
  - no provider -> `ErrPaymentCredentialsNotFound`
- `SePayProvider.CreatePaymentLink`:
  - sinh payment code đúng prefix
  - QR URL đúng bank/account/amount/des
  - retry khi code trùng
  - reject amount <= 0
- `SePayProvider.VerifyWebhook`:
  - valid signature
  - invalid signature
  - outgoing transfer ignored
  - missing code ignored/unmatched theo policy
- `PaymentService.HandleWebhook`:
  - matched SePay event mark invoice `PAID`
  - duplicate transaction không xử lý lại
  - wrong manager không update invoice
  - underpaid không mark `PAID`

### Handler tests

- `POST /api/v1/payments/providers/sepay/managers/{managerID}/webhook` trả `200` với body `{"success": true}` khi event hợp lệ.
- Invalid HMAC trả `400`.
- Missing credentials trả `401`.

### Frontend tests/manual QA

- Save SePay config encrypt secret fields.
- Copy webhook URL.
- Delete SePay config làm payment links SePay active cũ thành `STALE`.
- Khi SePay active, invoice Zalo message có QR SePay.
- Khi xóa SePay nhưng PayOS còn active, invoice Zalo message có QR PayOS.

## Thứ tự implement để giảm rủi ro

1. Thêm model constants và SePay credential types.
2. Refactor `PaymentCredentialService` hỗ trợ multi-provider và provider priority.
3. Thêm config handler/routes cho SePay.
4. Thêm `SePayProvider.CreatePaymentLink`.
5. Thêm `CreatePreferredPaymentLinkForInvoice` và đổi Zalo delivery sang method mới.
6. Thêm header vào webhook input và implement `SePayProvider.VerifyWebhook`.
7. Thêm amount guard/idempotency tests cho webhook.
8. Thêm frontend API và `SePaySettingsCard`.
9. Manual test với ngrok/dev webhook.
10. Thêm đối soát API v2 nếu cần.

## Acceptance criteria

- Manager có SePay active thì hóa đơn Zalo tạo QR SePay.
- Manager chỉ có PayOS active thì hóa đơn Zalo tạo QR PayOS.
- Tenant chuyển khoản đúng mã SePay thì invoice tự động `PAID`.
- Giao dịch trùng lặp không tạo notification/update lặp.
- Giao dịch sai manager hoặc sai code không update invoice.
- Giao dịch thiếu tiền không mark `PAID`.
- Credential không lưu plaintext trong DB.
- Webhook event lưu đủ raw payload để audit.
