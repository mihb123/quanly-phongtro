# Invoice & Tenant Role APIs

## 1. Overview
Hệ thống quản lý hóa đơn (Invoice) giúp tự động tính toán số tiền phòng, điện nước và các phụ phí khác dựa trên chỉ số đầu cuối mỗi tháng. Quản trị viên quản lý quy trình ra hóa đơn, cập nhật thanh toán.
Bên cạnh đó cung cấp các API chuyên biệt (Tenant APIs) để người dùng với quyền (role) `TENANT` truy cập để thu thập thông tin hóa đơn cá nhân và kiểm tra chi tiết phòng nghỉ.

## 2. API Endpoints - Invoice Management (Dành cho MANAGER)

### 2.1 Get Invoices List
**GET** `/api/invoices`
**Query Parameters:**
- `room_id`: uuid (Tùy chọn)
- `house_id`: uuid (Tùy chọn)
- `month`: int
- `year`: int
- `status`: string ('UNPAID', 'PARTIALLY_PAID', 'PAID')

### 2.2 Create & Calculate Invoice
**POST** `/api/invoices`
Tính hóa đơn có thể được tạo thủ công hoặc tự động.
**Request Body:**
```json
{
  "room_id": "uuid",
  "month": "integer",
  "year": "integer",
  "new_electricity_index": "integer",
  "new_water_index": "integer",
  "discount": "decimal",
  "other_fee": "decimal"
}
```

### 2.3 Update Invoice Payment Status
**PATCH** `/api/invoices/:id/status`
Cập nhật trạng thái thanh toán hàng tháng sau khi người quản lý nhận được tiền.
**Request Body:**
```json
{
  "status": "PAID"
}
```

### 2.4 Get Invoice Details
**GET** `/api/invoices/:id`

## 3. API Endpoints - Tenant Dashboard (Dành riêng cho TENANT)

(Những APIs này yêu cầu Token của một User có role đăng nhập là `TENANT`).

### 3.1 Get My Invoices
**GET** `/api/tenant/my-invoices`
- Lấy danh sách hóa đơn lịch sử và hiện tại, lọc tự động dựa vào `user_id` đang login. Người dùng tự xem mình nợ tiền hay đã đóng đủ.

### 3.2 Get My Room & Contract Info
**GET** `/api/tenant/my-room`
- Trả về thông tin trạng thái phòng, bảng giá tiền điện, nước cơ bản mà hợp đồng đang sử dụng và chi tiết các người cùng phòng hiện tại.

## 4. Flow chi tiết: Tính toán và Sinh Hóa Đơn

1. **Manager** nhập chỉ số điện nước cuối tháng cho 1 phòng qua API tạo Invoice.
2. **Backend**:
   - Truy vấn Invoice của tháng/kỳ gần nhất để lấy `old_electricity_index` và `old_water_index` hiện hữu (Hoặc có thể tra lại bản ghi Room có thuộc tính cache các index cũ). Nếu là lần ghi đầu tiên, số index cũ có thể bằng 0 hoặc được cấp tay lúc set up.
   - Truy vấn thông tin cấu hình giá từ bảng `houses` gắn liền với phòng này: `default_electricity_price`, `default_water_price`, `default_wifi_price`...
   - Tính toán Logic:
     - Tiền điện = `(new_electricity_index - old_electricity_index) * default_electricity_price`
     - Tiền nước = `(new_water_index - old_water_index) * default_water_price`
     - Các loại tiền khác = `room_fee` + `wifi_fee` + `parking_fee` + ...
     - Tổng tiền = Tổng các số trên trừ đi `discount` cộng thêm `other_fee`.
   - Transaction: Insert bản ghi hoá đơn vào DB (`invoices`).
3. (Optional): Kích hoạt event gửi thông báo có hóa đơn mới về Email cho người thuê (Tenant) được định danh ở phòng.
4. Return 201 Created cùng details về hóa đơn.

## 5. Todo
- [ ] Viết UseCase/Service xử lý thuật toán tính `InvoiceCalculator`.
- [ ] Implement các APIs phục vụ chức năng quản lý, listing và xem chi tiết quản trị Invoice (`internal/handler`, `internal/service`).
- [ ] Phân luồng Middlewares: Xây dựng check authorization riêng phục vụ cho Role `TENANT` truy xuất các endpoint dạng `/api/tenant/*`.
- [ ] Implement logic Endpoint cho `my-invoices` ngầm lọc điều kiện qua relationship mapping: `User -> Tenant Record -> Room -> Invoice`.
- [ ] Implement Endpoint `my-room`.

## 6. Security & Permissions
- [ ] **Manager**: Phải xác minh được `room_id` tạo hóa đơn thực sự nằm trong `houses` thuộc quyền sở hữu của chính User manager này gửi token, chống nhầm lẫn/chiếm đoạt thay đổi Invoice nhà khác.
- [ ] **Tenant**: Cách ly dữ liệu thông qua định danh JWT. Hóa đơn chỉ dành cho phòng có Tenant ID tương ứng với `user_id` JWT. Bảo vệ endpoint cẩn thận chống truy cập chéo hóa đơn của các room khác.
