# Payment Gateway Integration

## Mục tiêu

Payment hiện được thiết kế theo kiến trúc provider adapter. Backend giữ một service trung lập để điều phối nghiệp vụ hóa đơn, còn từng cổng thanh toán chỉ chịu trách nhiệm gọi SDK/API riêng, tạo link, hủy link và xác thực webhook.

PayOS là provider đầu tiên được triển khai. Các route legacy `/api/v1/payos/*` vẫn được giữ để không làm hỏng cấu hình cũ, nhưng luồng mới nên dùng route generic theo provider và manager.

## Kiến trúc

Các lớp chính:

- `PaymentService`: điều phối nghiệp vụ payment trung lập, bao gồm reuse active link, chọn provider, load credential, lưu event, idempotency, mark invoice paid và gửi thông báo.
- `PaymentProvider`: interface adapter cho từng gateway.
- `PayOSProvider`: adapter PayOS, chỉ chứa logic PayOS SDK, order code, cancel, verify signature và parse webhook.
- `PaymentCredentialService`: đọc/ghi credential theo manager, mã hóa credential trước khi lưu DB.
- `InvoicePaymentRepository`: lưu payment links, provider credentials và webhook events.

Luồng tạo QR hóa đơn:

1. Zalo invoice delivery gọi `PaymentService.CreatePaymentLinkForInvoice`.
2. `PaymentService` kiểm tra active link theo `invoice_id + provider`.
3. Service load credential theo `manager_id + provider`.
4. Provider adapter tạo link thanh toán.
5. Service lưu `invoice_payment_links` với `provider`, `provider_order_ref`, `checkout_url`, `qr_code`.

Luồng webhook:

1. Provider gọi route webhook.
2. Handler truyền `provider`, `managerID`, raw body vào `PaymentService`.
3. Service load đúng credential của manager.
4. Provider adapter verify chữ ký và trả dữ liệu event chuẩn hóa.
5. Service check idempotency bằng `provider + provider_order_ref + transaction_reference`.
6. Service lưu `payment_events`.
7. Nếu match invoice, invoice được chuyển `PAID`, link được chuyển `PAID`, và gửi thông báo Zalo best-effort.

## Database

Bảng `payment_provider_credentials`:

- `manager_id`: manager sở hữu credential.
- `provider`: ví dụ `payos`.
- `encrypted_credentials`: JSON credential đã mã hóa AES-256-GCM.
- `is_active`: credential hiện còn dùng hay không.
- Unique theo `(manager_id, provider)`.

Bảng `invoice_payment_links`:

- `provider`: provider tạo link.
- `provider_order_ref`: mã đơn hàng/reference chuẩn hóa theo provider.
- `order_code`: giữ cho PayOS compatibility.
- `payment_link_id`, `checkout_url`, `qr_code`, `amount`, `status`.

Bảng `payment_events`:

- `provider`, `manager_id`, `invoice_id`.
- `provider_order_ref`, `order_code`.
- `transaction_reference`.
- `raw_payload`, `signature_result`, `matching_method`, `status`.

## API PayOS

Manager APIs:

- `GET /api/v1/payments/providers/payos/config`
- `POST /api/v1/payments/providers/payos/config`
- `DELETE /api/v1/payments/providers/payos/config`
- `GET /api/v1/payments/public-key`

Webhook/redirect APIs:

- `POST /api/v1/payments/providers/payos/managers/{managerID}/webhook`
- `GET /api/v1/payments/providers/payos/return`
- `GET /api/v1/payments/providers/payos/cancel`

Legacy aliases:

- `POST /api/v1/payos/webhook`
- `GET /api/v1/payos/return`
- `GET /api/v1/payos/cancel`

Legacy webhook dùng app-level PayOS env credential. Manager-specific PayOS webhook nên dùng URL có `managerID` để backend load đúng checksum key.

## Security

- Credential từ frontend được mã hóa bằng RSA public key transport trước khi gửi.
- Credential lưu DB được mã hóa bằng AES-256-GCM.
- Env `APP_SECRET_ENCRYPTION_KEY` là key chính cho payment credentials.
- Nếu `APP_SECRET_ENCRYPTION_KEY` không có, app fallback sang `ZALO_BOT_ENCRYPTION_KEY` để không làm hỏng deploy hiện tại.
- PayOS app-level env vẫn được hỗ trợ:
  - `PAYOS_CLIENT_ID`
  - `PAYOS_API_KEY`
  - `PAYOS_CHECKSUM_KEY`

## Thêm Payment Gateway Mới

### 1. Định nghĩa provider key

Thêm constant trong model, ví dụ:

```go
const PaymentProviderMomo = "momo"
```

Nếu gateway cần payment method riêng trên invoice, thêm mapping trong `paymentMethodForProvider`.

### 2. Tạo adapter

Tạo file trong `internal/service`, ví dụ `momo_provider.go`, implement interface:

```go
type PaymentProvider interface {
    Name() string
    CreatePaymentLink(ctx context.Context, input PaymentCreateInput) (*PaymentProviderLink, error)
    CancelPaymentLink(ctx context.Context, input PaymentCancelInput) error
    VerifyWebhook(ctx context.Context, input PaymentWebhookInput) (*VerifiedPaymentEvent, error)
}
```

Adapter chỉ nên chứa logic gateway:

- build request tạo link.
- gọi SDK/API gateway.
- map response sang `PaymentProviderLink`.
- verify webhook signature.
- map webhook sang `VerifiedPaymentEvent`.

Không đưa logic invoice, tenant, Zalo notification, repository query hoặc HTTP response vào adapter.

### 3. Credential schema

Nếu credential khác PayOS, thêm type credential riêng trong `PaymentCredentialService` và map sang `map[string]string` cho adapter.

Credential phải được lưu qua `payment_provider_credentials.encrypted_credentials`, không lưu plaintext.

### 4. Đăng ký provider

Trong `cmd/api/main.go`, thêm provider vào registry:

```go
paymentRegistry := service.NewPaymentProviderRegistry(
    service.NewPayOSProvider(),
    service.NewMomoProvider(),
)
```

### 5. Thêm route config nếu cần UI

Route webhook generic đã có dạng:

```text
POST /api/v1/payments/providers/{provider}/managers/{managerID}/webhook
```

Nếu gateway cần màn hình cấu hình riêng, thêm handler/API tương tự PayOS config:

- get config status.
- save encrypted credential.
- delete/deactivate credential.
- return webhook URL theo manager.

### 6. Test tối thiểu

Mỗi provider mới nên có test cho:

- map request tạo link.
- map response provider sang `PaymentProviderLink`.
- verify webhook valid/invalid signature.
- idempotency event qua `PaymentService`.
- missing manager credential.
- webhook route với manager-specific credential.

Không cần test lại toàn bộ invoice/Zalo nếu provider chỉ thay adapter và dùng lại `PaymentService`.
