# Thống Nhất Tích Hợp PayOS

Tài liệu này ghi lại các quyết định đã được thống nhất trong quá trình thảo luận để thiết kế kiến trúc và tính năng thanh toán PayOS kết hợp với Zalo Bot.

## 1. Tài liệu đã xem xét
- **PayOS**:
  - Giới thiệu chung: https://payos.vn/docs/
  - API thanh toán: https://payos.vn/docs/api/
  - Cấu trúc Webhook & Chữ ký: https://payos.vn/docs/du-lieu-tra-ve/webhook/
  - Go SDK: https://payos.vn/docs/sdks/back-end/golang/

## 2. Các Quyết định Kỹ thuật (Decisions Selected)
1. **Quản lý Tài khoản (Account Scope):**
   - **Giai đoạn 1 (v1):** Sử dụng 1 bộ API Key (Client ID, API Key, Checksum Key) duy nhất cho toàn bộ hệ thống (app-level). Tiền sẽ về một tài khoản tập trung.
   - **Giai đoạn sau:** Hỗ trợ mỗi quản lý nhập API Key riêng biệt để tiền về trực tiếp tài khoản chủ nhà.

2. **Cách tạo và gửi ảnh QR Code:**
   - Thay vì gửi dạng chữ hoặc phụ thuộc SDK, sử dụng công cụ tạo QR bên thứ 3 (cụ thể là `quickchart.io/qr`) để render chuỗi `qrCode` chuẩn EMVCo mà PayOS trả về thành một file ảnh PNG.
   - Bot Zalo sẽ gửi **2 tin nhắn ảnh liên tiếp**:
     - Tin nhắn 1: Ảnh hóa đơn chi tiết.
     - Tin nhắn 2: Ảnh mã QR thanh toán (kèm theo hướng dẫn).

3. **Logic Cập nhật Hóa Đơn (Webhook Auto-mark PAID):**
   - Webhook của PayOS là nguồn dữ liệu duy nhất đáng tin cậy. Khi có webhook thành công, hệ thống tự động gạch nợ (mark PAID).
   - Bổ sung trường `payment_method` vào bảng `invoices` để phân biệt hóa đơn được thanh toán tự động qua PayOS (`PAYOS`) hay do quản lý gạch bằng tay (`MANUAL`).

4. **Logic Ghép Nối Giao Dịch (Matching Priority):**
   - *Ưu tiên 1:* Trùng khớp mã `orderCode` trong nội dung chuyển khoản.
   - *Ưu tiên 2:* Tên người chuyển khớp với tên Tenant đang hoạt động.
   - *Ưu tiên 3:* Số tiền duy nhất (chỉ có 1 hóa đơn nợ đúng số tiền đó).
   - *Mặc định:* Không tự động mark PAID nếu không chắc chắn 100%, thay vào đó báo quản lý xử lý tay (Unmatched).

5. **Thông Báo Zalo (Notifications):**
   - **Tenant:** Nhận thông báo ngắn gọn: "✅ Hóa đơn phòng [Tên Phòng] tháng [Tháng] ([Số Tiền]) đã thanh toán thành công. Mã GD: [Mã]".
   - **Manager:** Nhận thông báo đối soát chi tiết: "🔔 [PayOS] Khách hàng [Tên Tenant] tại phòng [Tên Phòng] vừa chuyển khoản thành công [Số Tiền]. Hóa đơn đã tự động chuyển sang đã thanh toán."

## 3. Câu Hỏi Triển Khai (Deployment Checklists)
- Cần chuẩn bị tài khoản PayOS (Test & Live) và lấy đầy đủ 3 thông số: `CLIENT_ID`, `API_KEY`, `CHECKSUM_KEY`.
- Cấu hình Webhook URL trên Dashboard của PayOS (ví dụ: `https://api.domain.com/api/v1/payos/webhook`). Đảm bảo cấu hình Webhook URL ở cả môi trường DEV (dùng ngrok/localtunnel) và PROD.
- Đảm bảo biến môi trường `APP_URL` đã có để backend tự build redirect URLs (`returnUrl`, `cancelUrl`).
