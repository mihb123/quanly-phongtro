# Báo cáo Refactor — Quản lý phòng trọ

- **Mục tiêu:** chỉ ra các đoạn code rườm rà, khó bảo trì và đề xuất refactor **giữ nguyên hành vi** (surgical). Không phải báo cáo bug/bảo mật (xem `SECURITY_REVIEW.md`).
- **Ngày:** 2026-07-09
- **Ghi chú độ tin cậy:** Mục ✅ = đã đọc trực tiếp code và xác nhận. Mục 🔎 = ứng viên phát hiện qua kích thước/độ phức tạp file, nên xem lại trước khi làm.

---

## Ưu tiên CAO

### 1. ✅ `HandleWebhook` quá dài & lồng sâu — `internal/service/zalo/zalo_webhook_handler.go:22`
- **Vấn đề:** một hàm ~200 dòng ôm nhiều trách nhiệm: verify secret → linh tinh nhánh event → **liên kết tài khoản** (manager/tenant, phone parsing) → **định tuyến lệnh hoá đơn** → **xử lý ảnh giao dịch nhóm**. Lồng `if/else` tới 5–6 cấp (đặc biệt khối `95→203`).
- **Vì sao quan trọng:** đây là điểm vào từ internet cho toàn bộ luồng Zalo; khó đọc = dễ lọt lỗi logic/bảo mật khi sửa.
- **Đề xuất (giữ hành vi):** tách theo trách nhiệm, dùng early-return:
  - `verifyAndLoadManager()` (đã có `verifyWebhookSecret`, gom phần load user).
  - `handleAccountLinking(ctx, managerID, webhookCtx)` — toàn bộ khối liên kết theo phone/role.
  - `routeInvoiceCommand(...)` — khối `85–93` (đã tách một phần).
  - `handleGroupTransactionImage(...)` — khối `206–214` (gọi `processTransactionImage` đã có sẵn).
  - `HandleWebhook` chỉ còn điều phối tuần tự, mỗi nhánh `return` sớm.

### 2. ✅ Lặp khuôn mẫu handler thanh toán — `internal/handler/payment/payment_handler.go`
- **Vấn đề:** các cặp `GetPayOSConfig / SavePayOSConfig / DeletePayOSConfig` và `GetSePayConfig / SaveSePayConfig / DeleteSePayConfig` lặp cùng một khung: `authenticatedManagerID` → decode body → `decrypt…Config` → validate trường bắt buộc → gọi service → `WriteHeader + encode {success:true}`.
- **Vì sao quan trọng:** file 451 dòng, sửa 1 chỗ (vd format response) phải sửa nhiều nơi; dễ lệch.
- **Đề xuất:** trích helper dùng chung:
  - `writeSuccess(w)` cho `{"success": true}` (đang lặp ≥6 lần).
  - `decodeJSON(w, r, &req) bool` cho decode + lỗi 400 (đang lặp khắp handler).
  - Giữ phần validate/decrypt riêng theo provider (đúng, không nên gộp cưỡng ép).

### 3. 🔎 `RevenueView.tsx` (551 dòng) & `InvoiceView.tsx` (547 dòng) — `frontend/src/components/home/`
- **Vấn đề:** component "view" quá lớn, thường trộn: fetch/state, tính toán tổng hợp, bảng/filter, và JSX.
- **Đề xuất:** tách hook dữ liệu (`useRevenueSummary`, `useInvoiceList`) khỏi phần trình bày; tách bảng/hàng thành component con (`InvoiceRow`, `RevenueTable`). Không đổi hành vi, chỉ đổi cấu trúc file. Tuân convention hiện có (shared components + token trong `ui-conventions`).

---

## Ưu tiên TRUNG BÌNH

### 4. 🔎 `auth_service.go` (549) & `auth_handler.go` (487) — `internal/service/auth/`, `internal/handler/auth/`
- **Vấn đề:** file lớn nhất backend; khả năng cao trộn nhiều luồng (register, login, refresh, verify-email, location/geo) trong ít hàm.
- **Đề xuất:** xem hàm `Login`/`RefreshToken` — nếu ôm cả validate + phát token + tạo session + geo thì tách theo bước. Xác nhận trước khi cắt.

### 5. ✅ Phân nhánh QR theo provider nằm rải rác — `internal/service/zalo/zalo_invoice_delivery.go`
- **Vấn đề:** logic "SePay QR là ảnh sẵn, PayOS là chuỗi VietQR phải render" (`fetchPaymentQRImage:202`) tách biệt với chỗ gửi ảnh; dễ bỏ sót khi thêm provider.
- **Đề xuất:** dồn về một điểm quyết định theo `provider` (map provider → cách lấy QR), tránh so sánh chuỗi provider ở nhiều nơi.

### 6. 🔎 `zalo_invoice_command_core.go` (448) + nhóm `zalo_invoice_command_*` — `internal/service/zalo/`
- **Vấn đề:** state máy lệnh hoá đơn trải nhiều file; core 448 dòng dễ chứa hàm điều phối dài.
- **Đề xuất:** rà các hàm > ~60 dòng, tách bước parse → validate → build → phản hồi. (Đã tách file theo pending/handlers/flows là hướng đúng — tiếp tục theo trách nhiệm.)

### 7. 🔎 `tenant_repository.go` (416) / `invoice_repository.go` (361) — `internal/repository/`
- **Vấn đề:** repository lớn thường có câu SQL dài lặp cột.
- **Đề xuất:** trích hằng danh sách cột (`const tenantColumns = "..."`) dùng lại cho SELECT/scan, giảm lệch cột giữa các query. (Không đổi truy vấn.)

---

## Việc dọn nhỏ (Low)

- **✅ So sánh secret dùng `!=`** ở `zalo_webhook_parser.go:166` → đổi sang `hmac.Equal` (vừa nhất quán vừa an toàn hơn — xem SECURITY_REVIEW #3).
- **🔎 Chuỗi thông báo tiếng Việt hardcode** rải trong `zalo_webhook_handler.go` (hàng chục `SendMessage(..., "…")`). Gom vào hằng/`messages.go` để dễ chỉnh và test.
- **🔎 `map[string]interface{}` cho response** trong các handler Zalo → cân nhắc struct response có tên trường, nhất quán với `httpx.ResData`.

---

## Nguyên tắc khi thực hiện
1. Mỗi refactor **một trách nhiệm/lần**, chạy `go test ./...` (backend) và `pnpm exec tsc --noEmit` + `pnpm exec eslint .` (frontend) sau mỗi bước.
2. Không đổi chữ ký public/API, không gộp logic khác tầng chỉ để "gọn".
3. Ưu tiên #1 và #2 — rủi ro thấp, lợi ích bảo trì cao nhất.

> Các mục 🔎 nên được xác nhận bằng cách đọc lại hàm cụ thể trước khi cắt — chúng được nhận diện qua kích thước/độ phức tạp, chưa soi từng dòng như các mục ✅.
