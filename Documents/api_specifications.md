# Yêu cầu API (Backend API Specifications)

Hệ thống cung cấp các API chuẩn RESTful. Phản hồi mạc định dùng JSON format.
Tất cả các API yêu cầu xác thực (trừ Đăng ký/Đăng nhập Manager) sẽ cần truyền `Authorization: Bearer <token>` trên Header.

## 1. Auth APIs
- `POST /api/v1/auth/register`: Đăng ký tài khoản Manager. Mặc định gán role là MANAGER.
- `POST /api/v1/auth/login`: Đăng nhập, trả về JWT Token và thông tin User.
- `POST /api/v1/auth/verify-email`: Kích hoạt tài khoản bằng mã (OTP) gửi qua Email.

## 2. House Management APIs (Dành cho Manager)
- `GET /api/v1/houses`: Lấy danh sách các nhà trọ. Hỗ trợ sort (`name`, `created_at`) và filter (`name` search).
- `POST /api/v1/houses`: Tạo mới một nhà trọ. Body: `name`, `address` và các thiết lập giá default (điện, nước, dịch vụ...).
- `GET /api/v1/houses/:id`: Lấy chi tiết nhà trọ.
- `PUT /api/v1/houses/:id`: Cập nhật cấu hình nhà trọ.
- `DELETE /api/v1/houses/:id`: Xóa nhà trọ.

## 3. Room Management APIs (Dành cho Manager)
- `GET /api/v1/houses/:house_id/rooms`: Danh sách phòng trong 1 nhà. Hỗ trợ sort (`name`, `room_price`) và filter (`status`, `name` search).
- `POST /api/v1/houses/:house_id/rooms`: Tạo phòng mới.
- `GET /api/v1/rooms/:id`: Chi tiết phòng (kèm danh sách người thuê hiện tại).
- `PUT /api/v1/rooms/:id`: Sửa thông tin phòng.
- `DELETE /api/v1/rooms/:id`: Xóa phòng.

## 4. Tenant Management APIs (Dành cho Manager)
- `POST /api/v1/rooms/:room_id/tenants`: Thêm người thuê mới vào phòng. API này sẽ làm 2 việc:
   1. Tạo tài khoản đăng nhập (role TENANT) tạo mật khẩu mặc định, gửi thông tin đăng nhập qua Email.
   2. Tạo Record Tenant liên kết User vừa tạo với Phòng.
- `GET /api/v1/rooms/:room_id/tenants`: Danh sách người thuê của phòng. Hỗ trợ sort (`start_date`) và filter (`status`).
- `PUT /api/v1/tenants/:id`: Chuyển trạng thái người thuê rời đi.

## 5. Invoice Management APIs
- `GET /api/v1/invoices`: Danh sách hóa đơn. Hỗ trợ filter (`house_id`, `room_id`, `month`, `year`, `status`) và sort (`created_at`, `total_amount`).
- `POST /api/v1/rooms/:room_id/invoices`: Khởi tạo hóa đơn tháng cho 1 phòng.
  Body: `month`, `year`, `new_electricity_index`, `new_water_index`, `other_fee`.
  *Logic tính toán: Lấy chỉ số mới trừ chỉ số cũ (của tháng trước), nhân với giá thiết lập tại House hoặc Room.*
- `GET /api/v1/invoices/:id`: Xem chi tiết hóa đơn.
- `PUT /api/v1/invoices/:id/pay`: Cập nhật trạng thái thanh toán của hóa đơn thành PAID.

## 6. Tenant APIs (Người thuê login)
- `GET /api/v1/tenant/my-invoices`: Danh sách hóa đơn cá nhân. Hỗ trợ sort (`month`, `year`, `created_at`) và filter (`status`).
- `GET /api/v1/tenant/my-room`: Lấy thông tin phòng đang ở.
