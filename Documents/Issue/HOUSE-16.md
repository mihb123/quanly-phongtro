# Tenant Management APIs

## 1. Overview
Quản lý hợp đồng, hồ sơ và thông tin người thuê (Tenant). Tự động khởi tạo tài khoản hệ thống cho người thuê mới và tích hợp gửi thông tin (tài khoản, mật khẩu) qua Email. Quản lý thay đổi trạng thái và thời hạn lưu trú của người thuê theo từng phòng trọ.

## 2. API Endpoints

### 2.1 Get Tenants List
**GET** `/api/tenants`
**Query Parameters:**
- `page`: int
- `limit`: int
- `room_id`: uuid (Lọc theo phòng nhất định)
- `house_id`: uuid (Lọc theo toàn bộ khu nhà)
- `status`: string ('ACTIVE', 'INACTIVE')
- `search`: string (Tìm theo CCCD, số điện thoại, tên của người thuê)

### 2.2 Create Tenant (Thêm người thuê)
**POST** `/api/tenants`
**Request Body:**
```json
{
  "room_id": "uuid",
  "full_name": "string",
  "phone": "string",
  "email": "string",
  "identity_card": "string",
  "start_date": "YYYY-MM-DD"
}
```

### 2.3 Update Tenant Status (Thay đổi trạng thái người thuê - Kết thúc hợp đồng)
**PUT** `/api/tenants/:id/status`
**Request Body:**
```json
{
  "status": "INACTIVE",
  "end_date": "YYYY-MM-DD"
}
```

### 2.4 Get Tenant Details
**GET** `/api/tenants/:id`

## 3. Flow chi tiết: Thêm Người Thuê

1. **Manager** gọi API `POST /api/tenants` và truyền đầy đủ thông tin người thuê.
2. **Backend** thực thi hệ thống nghiệp vụ thông qua **Database Transaction**:
   - **Validation Phase**: 
     - Kiểm tra `room_id` có tồn tại và Manager này có quyền quản lý phòng đó không. 
     - Kiểm tra trạng thái phòng, nếu số lượng `ACTIVE` tenants hiện tại chưa vượt quá `max_tenants` của phòng thì mới xử lý tiếp.
   - **User Accounts Creation Phase**: 
     - Insert record vào bảng `users` với role `TENANT`.
     - `email` hoặc `phone` được dùng để đăng nhập phải là unique. 
     - Generate mật khẩu tự động ngẫu nhiên (hoặc dựa trên một quy tắc cụ thể) và được mã hóa bcrypt (`password_hash`) trước khi lưu.
   - **Tenant Record Creation Phase**:
     - Insert record vào bảng `tenants` liên kết với `room_id`. 
     - Lưu ý cần liên kết bản ghi `tenants` với `user_id` vừa tạo để cấp quyền đăng nhập cho người đó. (Do đó, có thể cần update cấu trúc của bảng `tenants` thêm trường `user_id` làm FK trỏ tới bảng `users`).
   - **Room Status Update Phase**: 
     - Chuyển `status` của phòng sang `OCCUPIED` nếu là người thuê đầu tiên đến thay vì `AVAILABLE`.
   - **3rd Party Integration (Email SMTP)**:
     - Đẩy thông tin email chứa URL, thông tin đăng nhập và Mật Khẩu qua Email Service đến email của người thuê. Việc gọi API này có thể xử lý Asynchronous thông qua message queue hoặc goroutines để không block HTTP request response.
3. Trả về kết quả khởi tạo thành công và trạng thái gửi Email.

## 4. Flow chi tiết: Thay đổi trạng thái Người Thuê
1. Manager thao tác loại bỏ hoặc chốt hạn hợp đồng cho Tenant thông qua `PUT /api/tenants/:id/status` với param `status = "INACTIVE"`.
2. Backend:
   - Ghi lại `end_date` vào bản ghi của `tenants`.
   - Cập nhật record status lên `INACTIVE`.
   - Trigger tính toán tự động: Kiểm tra phòng liên quan hiện tại còn `ACTIVE` tenant nào không. Nếu bằng 0 người, cập nhật tự động room `status` thành `AVAILABLE`.

## 5. Đề xuất bổ sung cấu trúc DB 
Bảng `tenants` nên bổ sung một column: `tenant_user_id` (UUID - Foreign key reference tới `users.id`) để hệ thống biết hồ sơ thuê thuộc tải khoản user nào mà sinh quyền đăng nhập ở mục API cho Tenant về sau.

## 6. Todo
- [ ] Bổ sung/chỉnh sửa DB Schema cho bảng `tenants` (cần ref tới user account).
- [ ] Viết mock service cho Email Notification (In-memory logging trước khi có SMTP config thật).
- [ ] Implement logic `POST /api/tenants` sử dụng DB Transaction an toàn.
- [ ] Implement `PUT /api/tenants/:id/status`.
- [ ] Implement Search List API.
- [ ] Test luồng cập nhật tự động `status` của Room khi số lượng tenants thay đổi.

## 7. Security & Rules
- [ ] Số điện thoại/Email phải duy nhất trong hệ thống `users`. Tránh đụng độ Duplicate Constraint gây lỗi.
- [ ] Hash password trước khi lưu (luôn luôn bắt buộc).
- [ ] Áp dụng đúng Authorization: Người quản lý chỉ có thể thêm/gỡ người thuê trong phạm vi `room_id` của các phòng thuộc hệ thống nhà (`house_id`) mình sở hữu.
