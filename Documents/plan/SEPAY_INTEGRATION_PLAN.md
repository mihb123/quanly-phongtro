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
- SePay Webhooks – Xác thực: https://developer.sepay.vn/en/sepay-webhooks/xac-thuc
- SePay Webhooks – Tích hợp/payload: https://developer.sepay.vn/en/sepay-webhooks/tich-hop-webhook
- SePay API v2: https://developer.sepay.vn/vi/sepay-api/v2/gioi-thieu
- VietQR động (qr.sepay.vn / vietqr.app): https://qr.sepay.vn
- Tài liệu workflow hiện có: `Documents/feature/payment_gateway_integration.md`

## Sự kiện kỹ thuật SePay đã xác thực (nguồn sự thật)

> Trích từ tài liệu chính thức SePay (mục Webhooks – Xác thực, Tích hợp webhook, VietQR). Các mục bên dưới tham chiếu phần này thay vì lặp lại; nếu SePay đổi tài liệu, sửa ở đây trước.

### QR động

- Endpoint ảnh QR: `https://qr.sepay.vn/img` (mirror của `https://vietqr.app/img`).
- Param **bắt buộc**: `acc` (số tài khoản nhận), `bank` (short name như `MBBank`/`Vietcombank`, hoặc BIN).
- Param **tùy chọn**: `amount`, `des` (nội dung CK), `template` (`compact`/`qronly`/`standee`), `showinfo`, `download`, `fullacc`, `holder`, `store`.

### Webhook tiền vào

- SePay hỗ trợ **4 phương thức xác thực** cho webhook: **API Key**, **HMAC-SHA256**, **OAuth 2.0**, hoặc **none**. KHÔNG mặc định là HMAC.
  - **API Key** (đơn giản nhất, khuyến nghị mặc định): SePay gửi header `Authorization: Apikey <KEY>`. Server so sánh `<KEY>` với giá trị đã cấu hình.
  - **HMAC-SHA256**: SePay gửi `X-SePay-Signature: sha256={hex}` và `X-SePay-Timestamp: <unix_seconds>`. Chữ ký = HMAC-SHA256(secret, `"{timestamp}.{raw_body}"`), hex-encode. **Phải ký trên raw body bytes**, không phải JSON re-serialize.
  - **OAuth 2.0**: `Authorization: Bearer <token>` (phức tạp, không cần cho phase 1).
- Payload JSON (field chính):
  - `id` (int) — **transaction ID của SePay; bất biến qua retry/replay → đây là KEY chống trùng (dedup)**.
  - `transferType` (string) — `in` = tiền vào, `out` = tiền ra. Chỉ xử lý `in`.
  - `transferAmount` (int) — số tiền VND, **luôn dương** (không phân biệt dấu theo chiều tiền).
  - `code` (string, **nullable**) — mã thanh toán SePay tự tách từ `content` theo **prefix cấu hình trong SePay dashboard** (Company → General settings). `null`/rỗng nếu không match prefix → giao dịch unmatched.
  - `content` (string) — nội dung CK gốc từ ngân hàng.
  - `referenceCode` (string) — mã tham chiếu của ngân hàng (KHÁC `id`).
  - `accountNumber`, `gateway`, `transactionDate` (`YYYY-MM-DD HH:mm:ss`, giờ VN), `subAccount`, `accumulated`, `description`.
- Response server **phải** trả: HTTP **200 hoặc 201**, body **đúng** `{"success": true}`, trong **30 giây**. Trả khác/chậm → SePay retry → bắt buộc idempotency.

### API v2 (đối soát)

- Base URL: `https://userapi.sepay.vn/v2`. Auth: `Authorization: Bearer <API_TOKEN>`.
- Liệt kê giao dịch: `GET /v2/transactions`. Filter: `since_id`, `page` + `limit` (tối đa 100), `transaction_date_from`/`transaction_date_to`.
- Rate limit (theo agy đọc tài liệu, cần xác nhận lại trước khi dựa vào): ~3 req/s, vượt → `429`.

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
  "code_prefix": "PT",
  "webhook_auth_method": "apikey",
  "webhook_api_key": "key-for-Authorization-Apikey-header",
  "webhook_secret": "secret-for-hmac-sha256",
  "api_token": "optional-for-reconciliation-v2"
}
```

> Lưu ý quan trọng: SePay có **tới 3 secret khác nhau, đừng gộp**:
> - `webhook_api_key`: dùng khi `webhook_auth_method = "apikey"` — server so sánh với header `Authorization: Apikey ...`.
> - `webhook_secret`: dùng khi `webhook_auth_method = "hmac"` — Secret Key để verify `X-SePay-Signature`.
> - `api_token`: chỉ cho **đối soát API v2** (`Authorization: Bearer ...`), KHÔNG liên quan webhook. Plan cũ mô tả `api_token` mơ hồ — đây là thực thể tách biệt.

Field **bắt buộc** (tạo QR + auto-detect mã):

- `bank_short_name`
- `account_number`
- `account_name`
- `code_prefix` — **phải trùng** prefix cấu hình trong SePay dashboard, nếu không `code` webhook trả về sẽ `null` và mọi giao dịch thành unmatched.

Field **xác thực webhook** (chọn 1 phương thức; xem [Sự kiện kỹ thuật đã xác thực](#webhook-tiền-vào)):

- `webhook_auth_method`: `"apikey"` (mặc định khuyến nghị) | `"hmac"` | `"none"`.
- `webhook_api_key`: bắt buộc nếu method = `apikey`.
- `webhook_secret`: bắt buộc nếu method = `hmac`.

Field **tùy chọn**:

- `api_token`: chỉ cần nếu làm đối soát bằng API v2.

Không cần thêm bảng DB mới cho phase 1. Unique `(manager_id, provider)` trong `payment_provider_credentials` đã đủ.

> Refactor đi kèm: hiện `decryptCredentials`/`decryptPayOSCredentials` đang **gắn cứng kiểu PayOS** (`internal/service/payment_credential_service.go`). Cần thêm `SePayCredentials` struct + `sePayCredentialsMap` và một nhánh decrypt riêng cho `provider == sepay`, không tái dùng struct PayOS.

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
5. Tạo QR URL (xem param đã xác thực ở mục trên):

```text
https://qr.sepay.vn/img?acc={account_number}&bank={bank_short_name}&amount={amount}&des={payment_code}
```

> Ràng buộc khớp mã (rất dễ sai):
> - `des` = **payment_code đầy đủ kèm prefix** (vd `PT1A2B3C`). SePay tách `code` từ nội dung CK theo prefix dashboard và trả lại **cả prefix** (vd payload mẫu: content `SEVN63DC8E5C ...` → `code = "SEVN63DC8E5C"`).
> - Vì vậy `ProviderOrderRef` lưu xuống DB **phải đúng bằng chuỗi** SePay sẽ trả ở field `code`. Đừng lưu phần sau prefix rồi kỳ vọng match.
> - `code_prefix` của credential phải trùng prefix dashboard, nếu không `code` về `null`.
> - URL-encode `des`/`amount` khi build query.

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

Cần thêm header vào `PaymentWebhookInput` (hiện struct chỉ có `Body`, `Credentials`):

```go
type PaymentWebhookInput struct {
    Body        []byte
    Headers     http.Header // MỚI: cần cho Authorization Apikey / X-SePay-Signature
    Credentials map[string]string
}
```

> Thay đổi lan tỏa (đã verify trên code hiện tại):
> - Interface `PaymentService.HandleWebhook(ctx, provider, managerID, body []byte)` **chưa nhận header** → phải đổi chữ ký thành `HandleWebhook(ctx, provider, managerID string, body []byte, headers http.Header)` và truyền tiếp vào `PaymentWebhookInput.Headers`.
> - `internal/handler/payment_handler.go > handleWebhook` đã đọc raw body sẵn; chỉ cần truyền thêm `r.Header`.
> - PayOS provider hiện không dùng header → giữ nguyên hành vi, chỉ nhận thêm tham số.

Verify (theo phương thức cấu hình trong `webhook_auth_method`):

1. Xác thực request TRƯỚC khi parse nghiệp vụ:
   - `apikey`: đọc header `Authorization`, tách `Apikey <key>`, so sánh **constant-time** (`hmac.Equal`) với `webhook_api_key`. Sai → `ErrPaymentWebhookInvalid` (handler trả `400`/`401`).
   - `hmac`: lấy `X-SePay-Signature` (dạng `sha256={hex}`) + `X-SePay-Timestamp`, tính `HMAC-SHA256(webhook_secret, timestamp + "." + string(input.Body))`, hex-encode, so khớp constant-time. **Dùng `input.Body` raw**, không re-marshal. Sai → `ErrPaymentWebhookInvalid`. (Cân nhắc kiểm tra độ lệch `timestamp` để chống replay.)
   - `none`: bỏ qua xác thực (kém an toàn) — log cảnh báo, `SignatureResult = "VALID"` chỉ mang tính danh nghĩa.
2. Parse JSON body.
3. Chỉ xử lý `transferType == "in"`; nếu `"out"` → `ErrPaymentWebhookIgnored`.
4. Nếu `code` rỗng/`null` → `ErrPaymentWebhookIgnored` (hoặc tạo event unmatched tùy policy ở `HandleWebhook` — code rỗng nhưng `id` vẫn unique nên dedup vẫn an toàn).
5. Map event (lưu ý `id` là dedup key, KHÔNG dùng `referenceCode`):

```go
VerifiedPaymentEvent{
    ProviderOrderRef:     webhook.Code,           // = des đã gửi, gồm prefix
    Amount:               webhook.TransferAmount, // luôn dương
    TransactionReference: strconv.Itoa(webhook.ID), // id SePay = dedup key bất biến qua retry
    CounterAccount:       &webhook.AccountNumber, // tài khoản nhận tiền của manager
    RawPayload:           string(input.Body),
    SignatureResult:      paymentSignatureValid,  // hoặc paymentSignatureInvalid
    MatchingMethod:       paymentMatchOrderRef,
}
```

### 4. Cập nhật webhook handler

Trong `internal/handler/payment_handler.go`:

- `handleWebhook` **đã** đọc raw body có giới hạn (`maxPaymentWebhookBodyBytes`) — giữ nguyên.
- **Việc cần làm**: truyền thêm `r.Header` vào `PaymentService.HandleWebhook` (kéo theo đổi chữ ký interface, xem mục VerifyWebhook).
- Response success **đã đúng chuẩn SePay**: `writePaymentWebhookSuccess` đã trả `200 OK` + body `{"success": true}`. SePay chấp nhận `200` hoặc `201` nên không cần sửa. (Plan cũ ghi "sửa response" — thực tế đã xong, bỏ bước này.)
- Map lỗi hiện có đã hợp lý cho SePay: `ErrPaymentWebhookInvalid` → `400`, `ErrPaymentCredentialsNotFound` → `401` → khớp test plan handler.

### 5. Cập nhật `PaymentService.HandleWebhook`

Luồng hiện tại (đã verify) đã có: `CheckProviderEventExists` (dedup) → `GetPaymentLinkByProviderOrderRefForManager` → ghi `payment_event` → nếu có invoice thì `processInvoicePayment`. **Nhưng `processInvoicePayment` đang mark `PAID` vô điều kiện — chưa có amount guard.** Cần bổ sung:

1. Tìm `paymentLink` theo `managerID + provider + provider_order_ref` (đã có).
2. Nếu không tìm thấy, ghi event `UNMATCHED`, không update invoice (đã có).
3. **MỚI**: nếu `verifiedEvent.Amount < paymentLink.Amount`, ghi event nhưng **không** gọi `processInvoicePayment` (không mark `PAID`). Chèn guard này trong `HandleWebhook` ngay trước nhánh gọi `processInvoicePayment`.
4. Nếu amount đủ, mark invoice `PAID` (đã có trong `processInvoicePayment`).
5. Mark payment link `PAID` (đã có).
6. Gửi thông báo tenant/manager (đã có).

Idempotency: dùng unique hiện có `CheckProviderEventExists(provider, provider_order_ref, transaction_reference)`:

```text
sepay + code + id   (id = SePay transaction id, bất biến qua retry/replay)
```

> Vì SePay **retry khi không nhận `{"success": true}` trong 30s**, dedup theo `id` là bắt buộc. Map `TransactionReference = id` (không phải `referenceCode`) để retry cùng giao dịch không bị xử lý lại.

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
  "webhook_auth_method": "apikey",
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
- Code prefix
- Webhook auth method (select: `apikey` mặc định | `hmac`)
- Webhook API key (hiện khi method = `apikey`)
- Webhook secret (hiện khi method = `hmac`)
- API token optional (chỉ cho đối soát v2)

> RSA-encrypt các field bí mật (`webhook_api_key`, `webhook_secret`, `api_token`) qua `/payments/public-key` như PayOS card; `bank_short_name`/`account_number`/`account_name`/`code_prefix` không cần mã hóa.

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

5. Chọn phương thức xác thực webhook và nhập cùng giá trị vào SePaySettingsCard:
   - **API Key** (khuyến nghị, đơn giản): SePay gửi `Authorization: Apikey <key>` → lưu vào `webhook_api_key`.
   - **HMAC-SHA256** (an toàn hơn): lưu Secret Key vào `webhook_secret`.
6. Nếu có option lọc giao dịch không có mã thanh toán, bật lọc để giảm unmatched events.

## Đối soát bằng API v2

Phase 1 có thể chưa cần. Nếu làm thêm (thông số đã xác thực ở mục [API v2](#api-v2-đối-soát)):

1. Lưu `api_token` trong SePay credentials (khác `webhook_api_key`).
2. Thêm service `SePayReconciliationService`. Base `https://userapi.sepay.vn/v2`, header `Authorization: Bearer <api_token>`.
3. Kéo transactions: `GET /v2/transactions` với `since_id` (polling tăng dần) hoặc `transaction_date_from`/`transaction_date_to`; phân trang `page` + `limit` (≤100).
4. Reuse cùng logic match `provider_order_ref` (field `code`) và amount guard như webhook.
5. Chỉ xử lý giao dịch chưa tồn tại trong `payment_events` (dedup theo `id`).

Cần throttle theo rate limit (~3 req/s theo agy đọc tài liệu — **xác nhận lại trước khi triển khai**); vượt → `429`, cần backoff.

### Đã implement (phase 2)

Contract đã verify từ docs chính thức (`GET https://userapi.sepay.vn/v2/transactions`, `Authorization: Bearer`, response `{status, data[], meta.pagination}`, transaction field snake_case, **`id` là UUID** — KHÁC webhook `id` integer; rate limit 3 req/s).

- `internal/service/sepay_client.go` — `SePayClient.ListTransactions` (Bearer, phân trang `page`/`per_page≤100`, lọc `transaction_date_from/to`).
- `internal/service/sepay_reconciliation_service.go` — `SePayReconciliationService.ReconcileManager(ctx, managerID, dateFrom, dateTo)`: loop trang (throttle 350ms < 3 req/s, cap 100 trang chống loop), chỉ xử lý `transfer_type=in` có `code`, map `amount_in`/`id`(UUID).
- `internal/service/payment_service.go` — tách lõi dùng chung `ProcessVerifiedTransaction` (webhook + reconciliation) + **guard cross-source dedup**: vì v2-id(UUID) ≠ webhook-id(int) không match được, dùng **trạng thái link đã PAID** để bỏ qua settle/notify lại (chỉ ghi audit).
- Endpoint: `POST /api/v1/payments/providers/sepay/reconcile` (manager auth; body optional `date_from`/`date_to`, mặc định 7 ngày gần nhất) → trả `SePayReconcileResult`.
- Tests: paging/filter/mapping, thiếu `api_token`, và guard paid-link.

Chưa có UI trigger (manager đang phải gọi endpoint trực tiếp / qua cron) — tùy chọn thêm nút "Đối soát" trong SePaySettingsCard.

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
  - method `apikey`: header `Authorization: Apikey <đúng>` pass; sai/thiếu → `ErrPaymentWebhookInvalid`
  - method `hmac`: `X-SePay-Signature` đúng (ký trên `timestamp.raw_body`) pass; sai → `ErrPaymentWebhookInvalid`
  - `transferType: "out"` → `ErrPaymentWebhookIgnored`
  - `code` rỗng/null → ignored/unmatched theo policy
  - `TransactionReference` map từ `id` (không phải `referenceCode`)
- `PaymentService.HandleWebhook`:
  - matched SePay event mark invoice `PAID`
  - duplicate transaction không xử lý lại
  - wrong manager không update invoice
  - underpaid không mark `PAID`

### Handler tests

- `POST /api/v1/payments/providers/sepay/managers/{managerID}/webhook` trả `200` với body `{"success": true}` khi event hợp lệ.
- Xác thực sai (`Authorization: Apikey` sai hoặc `X-SePay-Signature` sai) → `400`.
- Missing credentials (manager chưa cấu hình SePay) → `401`.

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
