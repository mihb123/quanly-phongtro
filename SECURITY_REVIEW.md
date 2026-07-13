# Báo cáo Rà soát Bảo mật — Quản lý phòng trọ

- **Phạm vi:** Toàn bộ backend Go (`internal/…`) — auth/JWT/DPoP, thanh toán (PayOS/SePay), Zalo bot/webhook, xử lý file/upload, handler/repository. Frontend chỉ soi các luồng đưa dữ liệu tin cậy xuống backend.
- **Ngày:** 2026-07-09
- **Phương pháp:** đọc trực tiếp code qua CodeGraph + Read, truy vết luồng dữ liệu từ input người dùng tới thao tác nhạy cảm (đánh dấu hoá đơn đã trả, ghi file, cấp quyền).
- **Lưu ý:** Đây là rà soát thủ công theo mã nguồn, không chạy khai thác thực tế. Mức độ tin cậy (confidence) ghi theo thang 1–10.

---

## Tóm tắt

| # | Vấn đề | File | Loại | Mức độ | Confidence |
|---|--------|------|------|--------|-----------|
| 1 | Webhook SePay chấp nhận request **không xác thực** khi manager chưa chọn phương thức auth (mặc định) | `sepay_provider.go:128` | Broken Auth (webhook) | **MEDIUM** | 7 |
| 2 | Bỏ qua kiểm tra secret webhook Zalo với cấu hình "legacy" (secret rỗng) | `zalo_webhook_parser.go:159` | Auth bypass (điều kiện) | **MEDIUM** | 6 |
| 3 | So sánh secret webhook Zalo không dùng hằng-thời-gian | `zalo_webhook_parser.go:166` | Timing / hardening | **LOW** | 5 |
| 4 | Chữ ký URL file `zalo-invoices` là tuỳ chọn (bỏ query là bỏ luôn kiểm tra) | `upload_handler.go:23` | Weakened access control | **LOW** | 5 |

> Theo tiêu chí lọc false-positive nghiêm ngặt (chỉ giữ confidence ≥ 8), **không có finding nào vượt ngưỡng** để coi là lỗ hổng chắc chắn khai thác được. Tuy vậy, vì bạn yêu cầu rà cả "điểm yếu", tôi liệt kê đủ 4 mục dưới dạng weakness cần cân nhắc, kèm điều kiện khai thác trung thực.

---

## Vuln 1 — Webhook SePay chấp nhận request không xác thực (mặc định): `internal/service/payment/sepay_provider.go:128`

- **Severity:** MEDIUM · **Category:** broken_authentication · **Confidence:** 7/10
- **Mô tả:** `authenticateSePayWebhook` chọn cách xác thực theo `Credentials["webhook_auth_method"]`. Nhánh `case "none", "":` **trả về `nil` (chấp nhận)** — tức khi manager lưu cấu hình SePay mà không chọn phương thức auth (giá trị rỗng là mặc định, xem `SaveSePayConfig` handler `payment_handler.go:176` không bắt buộc trường này), endpoint webhook công khai `POST /api/v1/payments/providers/sepay/managers/{managerID}/webhook` sẽ nhận mọi payload không cần chữ ký.
- **Kịch bản khai thác:** Kẻ tấn công biết `managerID` (nằm trên URL webhook) và một `code` (mã thanh toán) trùng với một payment link đang mở của manager đó. Gửi POST giả `sePayWebhookPayload` với `transferType:"in"`, `code:<mã>`, `transferAmount` ≥ số tiền hoá đơn. Luồng `ProcessVerifiedTransaction` (`payment_service.go:222`) sẽ khớp payment link theo order-ref và **đánh dấu hoá đơn đã thanh toán** mà không có giao dịch ngân hàng thật.
- **Yếu tố giảm nhẹ (vì sao không phải HIGH):**
  - `managerID` là UUID người dùng → theo giả định, khó đoán.
  - `code` = `ToUpper(prefix + invoiceID[:8])`; `invoiceID[:8]` là 8 hex của UUID (~32 bit) → cần biết/đoán một order-ref đang hoạt động.
  - Có chốt chặn: chỉ settle khi tồn tại payment link khớp order-ref, `amount ≥ paymentLink.Amount`, và chưa settle (dedup theo transaction id).
- **Khuyến nghị:**
  1. Từ chối lưu cấu hình SePay với `webhook_auth_method` rỗng/`none` (bắt buộc `hmac` hoặc `apikey`), hoặc
  2. Nếu vẫn cho phép "none", chặn side-effect tài chính: chỉ ghi nhận audit, không tự settle hoá đơn khi webhook không được xác thực.
  3. Tối thiểu: hiển thị cảnh báo rõ ràng trên UI và log ở mức WARN kèm managerID.

---

## Vuln 2 — Bỏ qua kiểm tra secret webhook Zalo với config "legacy": `internal/service/zalo/zalo_webhook_parser.go:159`

- **Severity:** MEDIUM · **Category:** authentication_bypass (điều kiện) · **Confidence:** 6/10
- **Mô tả:**
  ```go
  func (s *zaloServiceImpl) verifyWebhookSecret(user *model.User, secretTokenHeader string) error {
      if secretTokenHeader == "" { return errors.New("missing webhook secret token") }
      if user.ZaloWebhookSecret == nil || *user.ZaloWebhookSecret == "" {
          return nil // <-- chấp nhận BẤT KỲ secret token nào khi chưa lưu secret
      }
      ...
  }
  ```
  Khi `ZaloWebhookSecret` rỗng (cấu hình cũ tạo trước khi tính năng secret ra đời), hàm chấp nhận mọi header `X-Bot-Api-Secret-Token` khác rỗng.
- **Kịch bản khai thác:** Với manager có config legacy, kẻ tấn công biết `managerID` (trên URL webhook `/api/v1/zalo/webhooks/{managerID}`) gửi POST giả với một header secret tuỳ ý → được xử lý như webhook hợp lệ (liên kết tài khoản Zalo, kích hoạt luồng lệnh hoá đơn…).
- **Yếu tố giảm nhẹ:** `SaveConfig` hiện tại **luôn** sinh secret 12 ký tự (`zalo_handler.go:90`), nên chỉ config cũ mới dính; `managerID` là UUID khó đoán.
- **Khuyến nghị:** Bỏ nhánh "cho phép khi secret rỗng". Nếu cần tương thích ngược, backfill secret cho toàn bộ user đang bật Zalo và bắt buộc secret != rỗng mới xử lý webhook.

---

## Vuln 3 — So sánh secret webhook Zalo không hằng-thời-gian: `internal/service/zalo/zalo_webhook_parser.go:166`

- **Severity:** LOW · **Category:** timing_side_channel / hardening · **Confidence:** 5/10
- **Mô tả:** `if plainSecret != secretTokenHeader` dùng so sánh chuỗi thường (early-exit), khác với các nơi thanh toán đã dùng `hmac.Equal` (`sepay_provider.go:153,177`). Về lý thuyết lộ độ dài khớp qua thời gian phản hồi.
- **Đánh giá:** Rủi ro thực tế thấp (nhiễu mạng, secret entropy cao). Đưa vào mục hardening cho nhất quán.
- **Khuyến nghị:** Dùng `hmac.Equal([]byte(plainSecret), []byte(secretTokenHeader))`.

---

## Vuln 4 — Chữ ký URL file `zalo-invoices` là tuỳ chọn: `internal/router/upload_handler.go:23`

- **Severity:** LOW · **Category:** weakened_access_control · **Confidence:** 5/10
- **Mô tả:** `signedUploadFileHandler` chỉ verify chữ ký **khi** request có query `exp`/`sig` (`hasSignedUploadQuery`). Bỏ hết query string → phục vụ file trực tiếp không kiểm tra. Nghĩa là cơ chế "URL ký, hết hạn" không có tác dụng bắt buộc.
- **Đánh giá / vì sao LOW:** Đây gần như **theo thiết kế**: ảnh hoá đơn được gửi cho Zalo bằng URL **không ký** để server Zalo tự tải (`zalo_invoice_delivery.go:195`), và tên file chứa 2 UUID (`{invoiceID}_{uuid}.png`) nên khó đoán. Path traversal đã được chặn tốt ở `safeUploadFilePath` (loại `/ \`, `filepath.Clean`, kiểm tra `Dir==root`).
- **Rủi ro còn lại:** URL ảnh (chứa thông tin hoá đơn: tên khách, phòng, số tiền — PII) là **vĩnh viễn công khai** với ai có link, thay vì hết hạn như ý định của cơ chế ký.
- **Khuyến nghị:** Nếu muốn thực thi hết hạn, bắt buộc chữ ký cho các file không phục vụ trực tiếp cho Zalo; hoặc tách 2 nhóm: file "public cho Zalo fetch" và file "chỉ truy cập có chữ ký".

---

## Các điểm đã kiểm tra và **an toàn** (để thấy độ phủ)

- **JWT (`security/jwt.go:120`)**: `Parse` ghim cứng `HS256`, từ chối alg khác → chặn alg-confusion/`none`. Access & refresh dùng secret riêng. ✔
- **DPoP proxy trust (`security/dpop_url.go`)**: `X-Forwarded-Proto/Host` chỉ được tin khi peer nằm trong `TRUSTED_PROXIES` CIDR; mặc định trống = không tin proxy nào → không spoof được origin cho htu. ✔
- **Path traversal upload (`shared/upload_path.go`, `router/upload_handler.go:47`)**: chuẩn hoá `\`→`/` rồi loại mọi `/`, `..`; `filepath.Clean` + kiểm tra thư mục cha == root. ✔ (khớp các auto-fix gần đây trong git log)
- **SePay HMAC/APIKey**: dùng `hmac.Equal` (constant-time), ký trên **raw body** đúng cách, không re-marshal. ✔
- **Phân quyền route (`router/router.go`)**: các nhóm nghiệp vụ đều qua `authMiddleware` + `requireRole("MANAGER")`; service scope theo `managerID` từ claims (không nhận ID owner từ client). ✔
- **Chốt chặn thanh toán (`payment_service.go:222`)**: dedup theo transaction id, không settle nếu thiếu payment link khớp, chặn underpaid, không re-settle link đã Paid. ✔
- **Liên kết tài khoản Zalo**: có kiểm tra chống hijack số đã liên kết, chặn liên kết tài khoản manager của người khác, tenant phải thuộc manager. ✔

---

## Kết luận

Bề mặt tấn công chính (JWT, DPoP, path traversal, chữ ký thanh toán, phân quyền cross-tenant) được xử lý **đúng và cẩn thận**. Không tìm thấy lỗ hổng khai thác trực tiếp mức HIGH với confidence ≥ 8.

Ưu tiên xử lý theo thứ tự: **Vuln 1** (buộc bật auth webhook SePay để không tự-settle hoá đơn khi không xác thực) → **Vuln 2** (bỏ nhánh bypass secret Zalo legacy) → **Vuln 3, 4** (hardening).

---

## Nhận định triage — review lại ngày 2026-07-10

> Đối chiếu lại từng finding với code thực tế (`sepay_provider.go`, `payment_handler.go`, `zalo_webhook_parser.go`, `zalo_handler.go`, `upload_handler.go`, `SePaySettingsCard.tsx`). Cả 4 finding đều mô tả đúng code, nhưng mức độ "cần fix" khác nhau rõ rệt.

| Finding | Verdict | Lý do ngắn gọn |
|---|---|---|
| Vuln 2 — Zalo legacy secret bypass | ✅ **Fix** | Auth bypass thật, fix rẻ, ít rủi ro regression |
| Vuln 1 — SePay auth method rỗng | ⚠️ **Fix một nửa** | Chỉ cần validate ở `SaveSePayConfig`; `"none"` là lựa chọn sản phẩm hợp lệ |
| Vuln 3 — timing compare | 🔧 Tuỳ | 1 dòng, tiện sửa cùng Vuln 2 thì làm |
| Vuln 4 — signed URL tuỳ chọn | ❌ **Không fix** | By design — Zalo cần URL không ký để tải ảnh |

### Vuln 2 — cần fix thật (ưu tiên cao nhất)

Đã xác nhận: khi `ZaloWebhookSecret` rỗng, hàm chấp nhận **bất kỳ** header secret nào khác rỗng — đây là auth bypass thật, không phải hardening. Webhook giả có thể kích hoạt liên kết tài khoản Zalo và luồng lệnh hoá đơn, tức có side-effect nghiệp vụ thật.

Vì `SaveConfig` (`zalo_handler.go:90`) luôn sinh secret 12 ký tự, chỉ config tạo trước khi có tính năng secret mới dính → fix rất an toàn:
1. Backfill secret cho các user đang bật Zalo mà secret rỗng (hoặc bắt các user đó re-save config).
2. Xoá nhánh `return nil` khi secret rỗng.

### Vuln 1 — chỉ cần fix phần validate backend

Báo cáo gốc thiếu một chi tiết làm giảm mức độ: frontend (`SePaySettingsCard.tsx:30`) đã bắt buộc chọn phương thức (`z.string().min(1)`), và `"none"` là **lựa chọn có chủ đích trên UI** ("Không xác thực") — user chọn "none" là quyết định sản phẩm, không phải lỗi.

Cái thật sự cần fix là **backend không validate**: `SaveSePayConfig` (`payment_handler.go:187-194`) chỉ check key/secret khi method là `apikey`/`hmac`, không reject method rỗng hoặc giá trị lạ. Gọi thẳng API (bỏ qua UI) vẫn lưu được config với method `""` → webhook nhận mọi payload.

- **Fix:** reject nếu `WebhookAuthMethod` không thuộc `{apikey, hmac, none}` khi lưu config.
- **Đừng vội chặn `""` ở phía webhook** (`authenticateSePayWebhook`): config cũ lưu trước khi có trường này sẽ chết lặng lẽ — hoá đơn không tự settle nữa. Nếu muốn siết, backfill data trước rồi mới tách `""` khỏi `"none"`.
- Việc cấm hẳn `"none"` (khuyến nghị 1 của báo cáo gốc) là quyết định sản phẩm, không phải bug — exploit cần biết cả managerID (UUID trong URL webhook, chỉ manager thấy) lẫn một payment code đang mở, rào cản thực tế khá cao.

### Vuln 3 — fix rẻ, tiện tay thì làm

Rủi ro thực tế gần như bằng 0 (nhiễu mạng, secret 12 ký tự random), nhưng fix chỉ 1 dòng (`hmac.Equal`) và làm code nhất quán với SePay. Nếu đã sửa Vuln 2 trong file này thì sửa luôn; không thì bỏ qua cũng không sao.

### Vuln 4 — không cần fix, chấp nhận được

Đây là **by design** (chính báo cáo gốc cũng thừa nhận): ảnh hoá đơn phải phục vụ **không ký** để server Zalo tải được (`zalo_invoice_delivery.go:195`). Bắt buộc chữ ký sẽ làm hỏng tính năng gửi hoá đơn qua Zalo. Các lớp bảo vệ còn lại đều ổn: tên file chứa 2 UUID (không đoán được), path traversal đã chặn kỹ ở `safeUploadFilePath`. Rủi ro còn lại (link ảnh chứa PII sống vĩnh viễn với ai có link) là trade-off chấp nhận được — xử lý triệt để đòi đổi kiến trúc lưu/phục vụ file, không đáng so với giá trị.

### Kế hoạch hành động đề xuất

1. **Vuln 2:** backfill secret Zalo rỗng + bỏ nhánh bypass. *(nhỏ, an toàn)*
2. **Vuln 1:** thêm validate `WebhookAuthMethod ∈ {apikey, hmac, none}` trong `SaveSePayConfig`. *(nhỏ, an toàn)*
3. **Vuln 3:** đổi sang `hmac.Equal` cùng lượt sửa Vuln 2. *(1 dòng)*
4. **Vuln 4:** không làm gì — ghi nhận là trade-off có chủ đích.
