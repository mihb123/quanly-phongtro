# SePay module — hướng dẫn tích hợp

Module độc lập để nhận thanh toán chuyển khoản qua [SePay](https://sepay.vn): sinh QR VietQR có mã thanh toán, xác thực webhook (HMAC-SHA256 / API Key), gọi SePay User API v2 (giao dịch, tài khoản đã liên kết) và đối soát giao dịch bị miss webhook.

- Go: chỉ dùng **standard library**, không phụ thuộc DB/ORM/framework.
- Không giữ state: lưu credential, payment link, event là việc của app (xem `sql/schema.sql`).
- Đang chạy thật trong `quanly-phongtro`; toàn bộ phần nối với app nằm trong một file `internal/service/payment/sepay_adapter.go`, dùng làm ví dụ tham chiếu.

## 1. Cấu trúc thư mục

```
sepay/
├── guide.md              # tài liệu này
├── config.go             # Credentials (+ Map/CredentialsFromMap, MergeCredentials, Validate), Environment, AuthMethod
├── errors.go             # ErrInvalidWebhook, ErrIgnoredWebhook, ErrMissingCredentials, ErrAccountNotLinked
├── payment_code.go       # CreatePayment, BuildPaymentCode, GenerateUniquePaymentCode, BuildQRURL
├── webhook.go            # VerifyWebhook, AuthenticateWebhook, SignHMAC, WebhookPayload
├── client.go             # Client (API v2): ListTransactions, ListBankAccounts, FindLinkedAccount
├── reconcile.go          # Reconciler: phân trang + throttle + cap trang
├── *_test.go             # go test ./... (không cần mạng)
├── sql/schema.sql        # bảng credential / payment link / payment event mẫu (PostgreSQL)
└── web/
    ├── sepay-api.ts      # createSePayApi(transport) + fetchTransport + mã hoá RSA-OAEP secret
    └── sepay-banks.ts    # danh sách ngân hàng SePay/VietQR hỗ trợ (supported=true)
```

## 2. Copy sang dự án khác

```bash
cp -r pkg/sepay <du-an-moi>/pkg/sepay
cd <du-an-moi> && go test ./pkg/sepay/...
```

Import theo module path của dự án mới, ví dụ `github.com/acme/shop/pkg/sepay`. Không cần sửa code bên trong vì package không import gì ngoài stdlib.

Frontend: copy `web/*.ts` vào `src/lib/` (TypeScript, không phụ thuộc framework; cần `fetch` + WebCrypto).

## 3. Chuẩn bị trên SePay

1. Tạo tài khoản tại https://my.sepay.vn, liên kết tài khoản ngân hàng nhận tiền. Muốn thử trước thì bật **Test Mode** (https://docs.sepay.vn/test-mode.html).
2. **Cấu hình → Cấu hình mã thanh toán**: bật nhận diện, đặt tiền tố (2–5 chữ cái, ví dụ `PH`). Tiền tố này phải trùng `Credentials.CodePrefix`, nếu không webhook sẽ có `code` rỗng và bị bỏ qua.
3. **Tích hợp Webhook → Thêm webhook**:
   - URL: endpoint của bạn, nên chứa id chủ tài khoản, ví dụ `https://app.example.com/api/v1/payments/providers/sepay/owners/{ownerID}/webhook`.
   - Loại giao dịch: Tiền vào (hoặc Tất cả, module tự bỏ qua tiền ra).
   - Định dạng: JSON.
   - Bảo mật: **HMAC-SHA256** (khuyến nghị) và copy Secret Key, hoặc API Key.
   - Bật "Tự động gửi lại khi server trả lỗi" (SePay thử lại tối đa 7 lần khi HTTP ngoài 2xx).
4. **API Access**: tạo API Token (Bearer) nếu muốn xác thực tài khoản ngân hàng khi lưu cấu hình và chạy đối soát.

## 4. Lưu cấu hình

`sepay.Credentials` có JSON tag sẵn, lưu nguyên struct (đã mã hoá) vào `payment_provider_credentials.encrypted_credentials`:

```go
creds := sepay.Credentials{
    Environment:       sepay.EnvironmentSandbox, // hoặc EnvironmentProduction (mặc định khi rỗng)
    BankShortName:     "Vietcombank",            // giá trị shortName trong web/sepay-banks.ts
    AccountNumber:     "0000000001",
    AccountName:       "CONG TY TNHH TEST",
    CodePrefix:        "PH",
    WebhookAuthMethod: sepay.AuthMethodHMAC,
    WebhookSecret:     secretFromForm,
    APIToken:          tokenFromForm, // tuỳ chọn
}
```

Nếu app lưu credential dạng `map[string]string`, dùng `creds.Map()` / `sepay.CredentialsFromMap(m)` (key = JSON tag).

Khuyến nghị khi lưu:

- Form sửa cấu hình cho phép để trống số tài khoản/secret/token: backend gộp với bản đã lưu bằng `sepay.MergeCredentials(update, existing)` rồi gọi `creds.Validate()` (thiếu trường bắt buộc hoặc thiếu secret theo phương thức xác thực → `ErrMissingCredentials`). API trạng thái chỉ trả cờ `has_webhook_secret` / `has_webhook_api_key` / `has_api_token`, không trả giá trị.

- Nếu có `APIToken`: gọi `client.ListBankAccounts` rồi `sepay.FindLinkedAccount` để chắc tài khoản thuộc đúng công ty sở hữu token, và lấy `AccountHolderName` chuẩn từ SePay.
- Mã hoá credential bằng khoá AES riêng (AES-256-GCM), không dùng chung khoá với tính năng khác.
- Frontend gửi secret đã mã hoá RSA-OAEP SHA-256 (`web/sepay-api.ts` → `encryptRSA`), backend giải mã bằng private key runtime.
- Khi đổi/xoá cấu hình: đánh dấu các payment link SePay `ACTIVE` của chủ đó thành `STALE` để QR cũ không còn được dùng.
- API không bao giờ trả plaintext secret về frontend, chỉ trả trạng thái đã mask.

```go
client := sepay.NewClient(nil) // nil = http.Client timeout 15s
resp, err := client.ListBankAccounts(ctx, creds.APIToken, sepay.ListBankAccountsParams{
    Environment: creds.Environment, BankShortName: creds.BankShortName, AccountNumber: creds.AccountNumber,
})
if err != nil { return err }
account, err := sepay.FindLinkedAccount(resp.Data, creds.BankShortName, creds.AccountNumber)
if errors.Is(err, sepay.ErrAccountNotLinked) { /* báo lỗi cho người dùng */ }
creds.AccountName = account.AccountHolderName
```

## 5. Tạo QR thanh toán

```go
payment, err := sepay.CreatePayment(ctx, creds, order.ID, amountVND,
    func(ctx context.Context, code string) (bool, error) {
        return repo.PaymentLinkExists(ctx, "sepay", code) // unique (provider, provider_order_ref)
    })
if err != nil { return err } // ErrMissingCredentials khi thiếu ngân hàng/số TK/prefix
// lưu payment_links: provider="sepay", provider_order_ref=payment.Code, qr_code=payment.QRURL, amount=payment.Amount, status=ACTIVE
```

Cần tách bước thì dùng trực tiếp `sepay.GenerateUniquePaymentCode` + `sepay.BuildQRURL`.

- Mã = `prefix + 8 ký tự đầu của id (bỏ "-")`, viết hoa, chỉ giữ `[A-Z0-9]`; trùng thì thêm hậu tố `1, 2, ...`.
- `QRURL` là ảnh PNG (`https://qr.sepay.vn/img?...`), có thể nhúng `<img>` hoặc tải về gửi qua chat.
- Tái sử dụng link `ACTIVE` nếu số tiền không đổi; đổi số tiền thì đánh dấu link cũ `STALE` rồi tạo mới.
- Để DB tự sinh `id` (đừng insert chuỗi rỗng vào cột uuid, xem mục 10).

## 6. Webhook

```go
func (h *Handler) SePayWebhook(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20)) // giữ RAW body để kiểm HMAC
    if err != nil { http.Error(w, "invalid body", http.StatusBadRequest); return }

    creds, err := h.store.SePayCredentials(r.Context(), chi.URLParam(r, "ownerID"))
    if err != nil { http.Error(w, "credentials not found", http.StatusBadRequest); return }

    transfer, err := sepay.VerifyWebhook(r.Header, body, creds, time.Now())
    switch {
    case errors.Is(err, sepay.ErrIgnoredWebhook):
        writeJSON(w, http.StatusOK, map[string]bool{"success": true}) // tiền ra / không có mã: trả 200 để SePay không gửi lại
        return
    case err != nil: // ErrInvalidWebhook, ErrMissingCredentials
        http.Error(w, "invalid webhook", http.StatusBadRequest)
        return
    }

    if err := h.payments.Settle(r.Context(), "sepay", ownerID, *transfer); err != nil {
        http.Error(w, "process failed", http.StatusInternalServerError) // SePay sẽ retry
        return
    }
    writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
```

Module đảm nhận:

- HMAC: header `X-SePay-Timestamp` (Unix giây) và `X-SePay-Signature: sha256=<hex>` với `hex = HMAC_SHA256(secret, "{timestamp}.{raw_body}")`; từ chối lệch giờ quá ±5 phút; so sánh constant-time.
- API Key: header `Authorization: Apikey <key>`.
- `none`: chấp nhận nhưng log cảnh báo (chỉ dùng khi thử nghiệm).
- Bỏ qua giao dịch tiền ra hoặc không có `code`; từ chối webhook có `accountNumber` khác tài khoản đã cấu hình.

App phải tự làm trong `Settle` (một transaction DB):

1. **Idempotency**: nếu đã có event `(provider, provider_order_ref, transaction_reference)` thì trả OK, không xử lý lại.
2. Tìm payment link theo `(owner, provider, provider_order_ref = transfer.PaymentCode)`.
3. Luôn ghi `payment_events` (kể cả không khớp: `UNMATCHED`) để audit, `raw_payload = transfer.RawPayload`.
4. Link đã `PAID` → chỉ ghi audit, không settle lại (chống trùng giữa webhook và đối soát, xem mục 7).
5. `transfer.Amount < link.amount` → ghi event, không đánh dấu đã trả.
6. Đủ tiền → cập nhật đơn `PAID`, link `PAID`, rồi gửi thông báo (best-effort, không làm fail webhook).

## 7. Đối soát (API v2)

Dùng khi webhook bị miss (server down, lỗi chữ ký...). Có thể gắn vào nút "Đối soát" hoặc cron.

```go
reconciler := sepay.NewReconciler(sepay.NewClient(nil)) // throttle 350ms/trang (giới hạn 3 req/s), tối đa 100 trang
result, err := reconciler.Reconcile(ctx, creds, "2026-10-01 00:00:00", "2026-10-07 23:59:59",
    func(ctx context.Context, t sepay.IncomingTransfer) error {
        return payments.Settle(ctx, "sepay", ownerID, t)
    })
// result: PagesFetched, Scanned, Processed, Failed, Truncated (true → thu hẹp khoảng thời gian rồi chạy lại)
```

Lưu ý: `id` giao dịch trong webhook là **số nguyên**, còn trong API v2 là **UUID**. Cùng một giao dịch ngân hàng sẽ có hai `transaction_reference` khác nhau, nên không dedup chéo bằng reference được. Phải dựa vào guard "link đã PAID thì chỉ ghi audit" ở bước 4 mục 6.

## 8. Frontend

`createSePayApi` nhận một **transport**, nên app tự quyết định gọi HTTP bằng gì (fetch, axios có interceptor auth/DPoP...).

```ts
import { createSePayApi, fetchTransport } from '@/lib/sepay-api'
import { SEPAY_SUPPORTED_BANKS, isSePaySupportedBank } from '@/lib/sepay-banks'

// Cách 1: fetch + cookie
const sepayApi = createSePayApi(fetchTransport('/api/v1'))

// Cách 2: axios có sẵn interceptor auth
const sepayApi = createSePayApi(<T,>(method, url, data) => apiClient.request<T>({ method, url, data }).then((r) => r.data))

await sepayApi.saveConfig({
  environment: 'sandbox', bank_short_name: 'Vietcombank', account_number: '0000000001',
  account_name: 'CONG TY TNHH TEST', code_prefix: 'PH', webhook_auth_method: 'hmac',
  webhook_secret: form.secret, api_token: form.apiToken,
}) // truyền plaintext: tự lấy public key và mã hoá RSA-OAEP các field secret
const status = await sepayApi.getConfig()   // hiển thị webhook_url để người dùng dán sang SePay
const result = await sepayApi.reconcile()   // mặc định 7 ngày gần nhất
```

- Router khác đường dẫn mặc định thì truyền tham số thứ hai `createSePayApi(transport, { publicKey, config, reconcile })`.
- Ô ngân hàng nên là Select từ `SEPAY_SUPPORTED_BANKS` (giá trị `shortName`), không cho nhập tự do.
- Khi cập nhật cấu hình, người dùng phải nhập lại secret vì backend không trả plaintext.
- Muốn import trực tiếp từ thư mục module (không copy) như quanly-phongtro: thêm alias Vite `'@sepay': path.resolve(__dirname, '../pkg/sepay/web')`, `server.fs.allow` thêm thư mục đó, và `paths` `"@sepay/*": ["../pkg/sepay/web/*"]` + `include` trong tsconfig.

## 9. Kiểm thử

- Unit: `go test ./pkg/sepay/...` (giả lập HTTP bằng `httptest`, không gọi mạng).
- Ký thử webhook: `sepay.SignHMAC(secret, timestamp, body)` tạo đúng header `X-SePay-Signature`.
- E2E trên SePay Test Mode (https://docs.sepay.vn/gia-lap-giao-dich.html):
  1. Test Mode có sẵn tài khoản giả lập; Vietcombank bắt buộc chọn **tài khoản ảo (VA)** khi giả lập.
  2. Tạo đơn, lấy mã thanh toán, vào **Mô phỏng giao dịch**: chọn tài khoản + VA, Tiền vào, số tiền, nội dung chứa mã.
  3. Bảng "Quy trình xử lý" cho biết SePay có nhận diện được mã và đã gửi webhook chưa; xem chi tiết ở **Tích hợp Webhook → Lịch sử gửi**.
  4. Token Test Mode chỉ dùng được với `https://userapi-sandbox.sepay.vn/v2` (`Environment = sandbox`).
- quanly-phongtro có sẵn script E2E: `.agents/skills/sepay-e2e/`.

## 10. Lỗi hay gặp

| Triệu chứng | Nguyên nhân / cách xử lý |
|---|---|
| Webhook 400 `signature mismatch` | Secret trong app khác Secret Key của webhook trên SePay; hoặc body bị middleware đọc/parse trước khi kiểm HMAC (phải ký trên raw bytes). |
| Webhook 400 `timestamp expired` | Đồng hồ server lệch hơn 5 phút, cần đồng bộ NTP. |
| Webhook 200 nhưng đơn không PAID | `code` rỗng (prefix không khớp, nội dung chuyển khoản bị sửa), event `UNMATCHED`, hoặc chuyển thiếu tiền. |
| `account mismatch` | Số tài khoản cấu hình khác `accountNumber` SePay gửi. |
| Không tạo được payment link: `invalid input syntax for type uuid: ""` | ORM insert cột `id` rỗng. Với bun dùng `ExcludeColumn("id", ...)` + `Returning("id, ...")`, để DB tự sinh. |
| Đối soát `missing credentials` | Chưa lưu API Token. |
| Đối soát `truncated: true` | Chạm cap 100 trang, chạy lại với khoảng ngày hẹp hơn. |
| HTTP 401 từ API v2 | Token sai môi trường (token Test Mode gọi production hoặc ngược lại) hoặc đã bị thu hồi. |
