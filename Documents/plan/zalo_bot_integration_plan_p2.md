# Phase 2: Gửi hóa đơn & Nhận ảnh chuyển khoản Zalo Bot

Bản kế hoạch triển khai Phase 2 cho tính năng tích hợp Zalo Bot, cho phép quản lý gửi hóa đơn dạng ảnh vào group Zalo của phòng, đồng thời Bot tự động nhận ảnh chuyển khoản của người thuê, tải về máy chủ và cập nhật trạng thái hóa đơn thành `PENDING_VERIFICATION` (Chờ xác nhận).

## User Review Required

- **Xử lý ảnh Upload**: Kế hoạch sẽ tải ảnh hóa đơn do Tenant gửi từ URL của Zalo (trong Webhook payload) và lưu vào thư mục `uploads/transactions/` ở server local.
- **Trạng thái hóa đơn**: Hóa đơn sẽ được chuyển sang trạng thái "Chờ xác nhận" (`PENDING_VERIFICATION`). Cần xác nhận lại xem đã có sẵn trạng thái này trong DB/Enum chưa, nếu chưa sẽ thêm vào.

## Open Questions

Không có câu hỏi mở ở thời điểm hiện tại, thiết kế đã được làm rõ thông qua quá trình Grill-me.

## Proposed Changes

---

### Database Migration

#### [NEW] migrations/000013_add_invoice_proof.up.sql
```sql
ALTER TABLE invoices ADD COLUMN transaction_image_path VARCHAR(255) NULL;
-- (Nếu cần) Đảm bảo trạng thái PENDING_VERIFICATION hợp lệ trong CHECK constraint của status hoặc enum
```
#### [NEW] migrations/000013_add_invoice_proof.down.sql
```sql
ALTER TABLE invoices DROP COLUMN IF EXISTS transaction_image_path;
```

---

### Go Backend - Model & Repository

#### [MODIFY] internal/model/invoice.go
- Thêm trường `TransactionImagePath *string` vào `Invoice` struct (với tag json, db tương ứng).
- Thêm hằng số cho trạng thái mới: `InvoiceStatusPendingVerification = "PENDING_VERIFICATION"`.

#### [MODIFY] internal/repository/invoice_repository.go
- Cập nhật các câu lệnh `SELECT`, `UPDATE` và struct tham số để hỗ trợ `transaction_image_path`.
- Thêm hàm `GetUnpaidInvoiceByRoomID(ctx, roomID)` để lấy hóa đơn chưa thanh toán mới nhất (hoặc cũ nhất) nhằm tự động map với ảnh chuyển khoản.

---

### Go Backend - Image Generation Refactoring

#### [NEW] internal/service/image_service.go
- Tạo `ImageService` riêng biệt chịu trách nhiệm xử lý các tác vụ liên quan đến hình ảnh để code dễ maintain.
- Di chuyển logic vẽ ảnh (sử dụng thư viện `gg`) từ `DownloadInvoiceImage` trong `invoice_handler.go` sang hàm `GenerateInvoiceImage(ctx context.Context, invoice *model.Invoice) ([]byte, error)` trong service này.
- Hàm này sẽ trả về mảng bytes của ảnh PNG, giúp tái sử dụng cho việc tải xuống và việc gửi qua API.

#### [MODIFY] internal/handler/invoice_handler.go
- Inject `ImageService` vào `InvoiceHandler`.
- Sửa lại `DownloadInvoiceImage` để gọi `GenerateInvoiceImage` từ `ImageService` và ghi mảng bytes trả về ra `ResponseWriter`.

---

### Go Backend - Zalo Bot Client & Service

#### [MODIFY] internal/service/zalo_client.go
- Thêm hàm `SendPhoto(ctx, botToken, chatID, imageBytes []byte, caption string) error`:
  - Sử dụng API gửi tin nhắn kèm file/ảnh của Zalo (thường dùng `multipart/form-data` hoặc upload file API trước rồi lấy `attachment_id` để gửi tin nhắn text). (Sẽ dựa theo doc Zalo Bot API để code trực tiếp).

#### [MODIFY] internal/service/zalo_service.go
- **Tính năng 1: Gửi Hóa Đơn**: Thêm hàm `SendInvoiceToZalo(ctx, managerID, invoiceID)`:
  - Lấy thông tin hóa đơn.
  - Lấy `group_chat_id` từ phòng của hóa đơn, báo lỗi nếu chưa liên kết.
  - Gọi `imageService.GenerateInvoiceImage` để lấy bytes ảnh.
  - Gọi `ZaloClient.SendPhoto` để gửi ảnh kèm dòng caption thông báo "Hóa đơn tiền nhà...".
- **Tính năng 2: Bắt sự kiện có ảnh (Webhook)**: Cập nhật hàm `HandleWebhook`:
  - Kiểm tra nếu payload là tin nhắn gửi từ group (`group_chat_id` tồn tại).
  - Kiểm tra xem tin nhắn có chứa ảnh hay không (qua attachments).
  - Lấy URL ảnh, dùng HTTP GET để tải ảnh về buffer.
  - Lưu file ảnh vào thư mục `uploads/transactions/` và lưu đường dẫn.
  - Tìm phòng dựa trên `group_chat_id`.
  - Tìm hóa đơn `UNPAID` ưu tiên nhất của phòng đó, cập nhật status thành `PENDING_VERIFICATION` + `transaction_image_path`.
  - Phản hồi lại group: "Đã nhận được ảnh chuyển khoản. Đang chờ quản lý xác nhận."

---

### Go Backend - Handlers & Routing

#### [MODIFY] internal/handler/zalo_handler.go
- Thêm endpoint `POST /api/v1/zalo/send-invoice/{id}`:
  - Validate phân quyền và gọi `zaloService.SendInvoiceToZalo`.
- Sửa `HandleWebhook` để gọi các luồng xử lý ảnh mới.

#### [MODIFY] internal/router/router.go
- Thêm route cho endpoint `POST /api/v1/zalo/send-invoice/{id}`.

---

### Frontend - UI

#### [MODIFY] frontend/src/api/zalo.tsx
- Export hàm gọi API `sendInvoiceToZalo(invoiceId: string)`.

#### [MODIFY] frontend/src/components/invoices/... (InvoiceList / InvoiceRow)
- Thêm Button "Gửi Zalo" (kèm icon Zalo) hiển thị ở các hóa đơn trạng thái `UNPAID`.
- Handle sự kiện onClick: Gọi `sendInvoiceToZalo`, hiển thị Toast thông báo đang gửi và thành công/thất bại.
- Hiển thị badge trạng thái "Chờ xác nhận" màu vàng/cam cho `PENDING_VERIFICATION`.
- Thêm nút "Xem ảnh GD" (View Proof) cho manager nếu `transaction_image_path` tồn tại, mở popup/modal xem ảnh.
- Tại giao diện, nếu hóa đơn "Chờ xác nhận", Manager có thể bấm "Xác nhận đã thanh toán" để chuyển trạng thái sang `PAID`.

---

## Verification Plan

### Automated/Local Tests
- Chạy thử hàm `GenerateInvoiceImage` đảm bảo xuất byte chuẩn.
- Dùng Postman giả lập gửi payload Webhook của Zalo chứa URL ảnh, kiểm tra server có tải file về thư mục `uploads` và đổi trạng thái hóa đơn thành `PENDING_VERIFICATION` không.

### Manual Verification
- Ở Frontend: Bấm nút "Gửi Zalo" trên 1 hóa đơn, kiểm tra group Zalo test xem bot có gửi ảnh hóa đơn + text vào đúng group không.
- Ở Group Zalo: Lấy nick phụ (đóng vai Tenant) gửi 1 ảnh hóa đơn vào group. Kiểm tra UI Frontend xem hóa đơn có tự chuyển sang "Chờ xác nhận" và hiển thị nút xem ảnh không. Mở ảnh lên có đúng ảnh tenant vừa gửi không.
