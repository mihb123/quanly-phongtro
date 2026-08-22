# Yêu cầu API (Backend API Specifications)

Hệ thống cung cấp các API chuẩn RESTful. Phản hồi mạc định dùng JSON format.
Tất cả các API yêu cầu xác thực (trừ Đăng ký/Đăng nhập Manager) sẽ cần truyền `Authorization: Bearer <token>` trên Header.

## 1. Auth APIs
- `POST /api/v1/auth/register`: Đăng ký tài khoản Manager. Mặc định gán role là MANAGER.
- `POST /api/v1/auth/login`: Đăng nhập, trả về JWT Token và thông tin User.
- `POST /api/v1/auth/verify-email`: Kích hoạt tài khoản bằng mã (OTP) gửi qua Email.

## 2. House Management APIs (Dành cho Manager)
- `GET /api/v1/house`: Lấy danh sách các nhà trọ. Hỗ trợ sort và filter qua query params.
- `POST /api/v1/house/create`: Tạo mới một nhà trọ. Body: `name`, `address`, các thiết lập giá default (điện, nước, cấu hình hóa đơn, phụ phí, ...) và (tùy chọn) thông tin thuê nguyên căn từ chủ nhà.
- `GET /api/v1/house/{id}`: Lấy chi tiết nhà trọ.
- `POST /api/v1/house/{id}`: Cập nhật cấu hình nhà trọ. Body kèm thông tin thuê nguyên căn: `owner_name`, `owner_phone`, `owner_rent_price`, `owner_deposit`, `rent_start_date`, `rent_end_date` (ngày định dạng `YYYY-MM-DD`, chuỗi rỗng = xóa ngày).
- `PATCH /api/v1/house/{id}/documents`: Upload CCCD chủ nhà và hợp đồng thuê nguyên căn (`multipart/form-data`, tối đa 10 file mỗi loại, chấp nhận `.jpg/.jpeg/.png/.pdf`).
   - File mới: `owner_cccd_file`, `owner_contract_file` (lặp lại field cho nhiều file).
   - Giữ file cũ: `kept_owner_cccd_paths`, `kept_owner_contract_paths` (danh sách path ngăn cách bởi dấu phẩy). Muốn xóa hết một nhóm thì gửi kèm `kept_owner_cccd_paths_empty=true` / `kept_owner_contract_paths_empty=true`; không gửi field nào của nhóm thì giữ nguyên.
   - File được phục vụ lại qua `GET /api/v1/tenant/files/{filename}` như CCCD khách thuê và hợp đồng phòng.
- `DELETE /api/v1/house/{id}`: Xóa nhà trọ.

## 3. Room Management APIs (Dành cho Manager)
- `GET /api/v1/room?house_id={id}`: Danh sách phòng trong 1 nhà.
- `POST /api/v1/room`: Tạo phòng mới.
- `GET /api/v1/room/{id}?house_id={id}`: Chi tiết phòng.
- `PATCH /api/v1/room/{id}`: Sửa thông tin phòng. Tự động cập nhật các hóa đơn UNPAID nếu giá phòng thay đổi.
- `DELETE /api/v1/room/{id}?house_id={id}`: Xóa phòng.

## 4. Tenant Management APIs (Dành cho Manager)
- `POST /api/v1/rooms/:room_id/tenants`: Thêm người thuê mới vào phòng. API này sẽ làm 2 việc:
   1. Tạo tài khoản đăng nhập (role TENANT) tạo mật khẩu mặc định, gửi thông tin đăng nhập qua Email.
   2. Tạo Record Tenant liên kết User vừa tạo với Phòng.
- `GET /api/v1/rooms/:room_id/tenants`: Danh sách người thuê của phòng. Hỗ trợ sort (`start_date`) và filter (`status`).
- `PUT /api/v1/tenants/:id`: Chuyển trạng thái người thuê rời đi.

## 5. Invoice Management APIs
- `GET /api/v1/invoice`: Danh sách hóa đơn. Hỗ trợ filter qua query params.
- `POST /api/v1/invoice`: Khởi tạo/Cập nhật hóa đơn tháng cho 1 phòng. (Upsert dựa trên `room_id` và `period`).
- `GET /api/v1/invoice/{id}`: Xem chi tiết hóa đơn.
- `PATCH /api/v1/invoice/{id}/pay`: Cập nhật trạng thái thanh toán của hóa đơn thành PAID.
- `PATCH /api/v1/invoice/{id}/unpay`: Hoàn tác trạng thái thanh toán về UNPAID.

## 6. Tenant APIs (Người thuê login)
- `GET /api/v1/tenant/my-invoices`: Danh sách hóa đơn cá nhân. Hỗ trợ sort (`month`, `year`, `created_at`) và filter (`status`).
- `GET /api/v1/tenant/my-room`: Lấy thông tin phòng đang ở.
