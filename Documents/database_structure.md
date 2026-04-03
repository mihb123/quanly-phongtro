# Cấu trúc Database (PostgreSQL)

Dưới đây là phác thảo các bảng chính (Tables) cho hệ thống Quản lý phòng trọ theo yêu cầu.

## 1. Bảng `users`
Lưu trữ thông tin tài khoản đăng nhập của Cả Quản lý (Manager) và Người thuê (Tenant).
- `id`: UUID, Primary Key.
- `email`: Varchar, Unique, Not Null.
- `password_hash`: Varchar, Not Null.
- `role`: Enum ('MANAGER', 'TENANT'). Chỉ có manager mới tự đăng ký, tenant do manager tạo.
- `full_name`: Varchar.
- `phone`: Varchar.
- `is_activated`: Boolean, Default False.
- `created_at`: Timestamp.
- `updated_at`: Timestamp.

## 2. Bảng `houses` (Nhà trọ)
Mỗi user (MANAGER) có thể quản lý nhiều nhà trọ.
- `id`: UUID, Primary Key.
- `manager_id`: UUID, Foreign Key (`users.id`).
- `name`: Varchar (VD: "Nhà trọ Cầu Giấy").
- `address`: Varchar.
- `default_electricity_price`: Decimal (Giá 1 số điện mặc định).
- `default_water_price`: Decimal (Giá 1 khối nước hoặc giá nước mặc định).
- `default_wifi_price`: Decimal.
- `default_parking_price`: Decimal.
- `default_service_price`: Decimal.
- `created_at`: Timestamp.
- `updated_at`: Timestamp.

## 3. Bảng `rooms` (Phòng trọ)
Mỗi nhà trọ có thể chia thành nhiều phòng.
- `id`: UUID, Primary Key.
- `house_id`: UUID, Foreign Key (`houses.id`).
- `name`: Varchar (VD: "Phòng 101").
- `room_price`: Decimal (Giá thuê phòng tính theo tháng).
- `max_tenants`: Integer (Số lượng người ở tối đa).
- `status`: Enum ('AVAILABLE', 'OCCUPIED', 'MAINTENANCE').
- `electricity_price`: Decimal (Tùy chỉnh giá điện riêng cho phòng, nếu có).
- `water_price`: Decimal (Tùy chỉnh giá nước riêng cho phòng, nếu có).
- `wifi_price`: Decimal (Tùy chỉnh giá wifi riêng cho phòng, nếu có).
- `parking_price`: Decimal (Tùy chỉnh giá gửi xe riêng cho phòng, nếu có).
- `service_price`: Decimal (Tùy chỉnh giá dịch vụ riêng cho phòng, nếu có).
- `created_at`: Timestamp.
- `updated_at`: Timestamp.

## 4. Bảng `tenants` (Hồ sơ người thuê)
Lưu thông tin chi tiết về quá trình người thuê ở tại 1 phòng. Có thể liên kết với bảng `users` để họ đăng nhập xem hóa đơn.
- `id`: UUID, Primary Key.
- `room_id`: UUID, Foreign Key (`rooms.id`).
- `created_by`: UUID, Foreign Key (`users.id`).
- `identity_card`: Varchar (CCCD).
- `start_date`: Date (Ngày bắt đầu thuê).
- `end_date`: Date (Ngày kết thúc thuê - Null nếu đang ở).
- `status`: Enum ('ACTIVE', 'INACTIVE').
- `created_at`: Timestamp.
- `updated_at`: Timestamp.

## 5. Bảng `invoices` (Hóa đơn)
Tính hóa đơn hàng tháng cho mỗi phòng.
- `id`: UUID, Primary Key.
- `room_id`: UUID, Foreign Key (`rooms.id`).
- `month`: Integer (Tháng của hóa đơn).
- `year`: Integer (Năm của hóa đơn).
- `room_fee`: Decimal (Tiền phòng).
- `old_electricity_index`: Integer (Số điện cũ).
- `new_electricity_index`: Integer (Số điện mới).
- `electricity_fee`: Decimal (Thành tiền điện = (mới - cũ) * giá).
- `old_water_index`: Integer (Số nước cũ).
- `new_water_index`: Integer (Số nước mới).
- `water_fee`: Decimal.
- `wifi_fee`: Decimal.
- `parking_fee`: Decimal.
- `service_fee`: Decimal.
- `other_fee`: Decimal.
- `discount`: Decimal (Giảm giá nếu có).
- `total_amount`: Decimal (Tổng cộng).
- `status`: Enum ('UNPAID', 'PARTIALLY_PAID', 'PAID').
- `created_at`: Timestamp.
