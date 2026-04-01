# House & Room Management APIs

## 1. Overview
Triển khai hệ thống API cho phép tự quản lý thông tin nhà trọ (House) và phòng trọ (Room) dành cho Manager. Hệ thống hỗ trợ đầy đủ các thao tác CRUD cùng với khả năng tìm kiếm, lọc và sắp xếp dữ liệu linh hoạt.

## 2. API Endpoints - House Management

### 2.1 Get Houses List
**GET** `/api/houses`
**Query Parameters:**
- `page`: int (default: 1)
- `limit`: int (default: 10)
- `search`: string (Tìm theo `name`, `address`)
- `sort`: string (VD: `created_at:desc`, `name:asc`)

### 2.2 Get House Details
**GET** `/api/houses/:id`

### 2.3 Create House
**POST** `/api/houses`
**Request Body:**
```json
{
  "name": "string",
  "address": "string",
  "default_electricity_price": 4000,
  "default_water_price": 25000,
  "default_wifi_price": 100000,
  "default_parking_price": 50000,
  "default_service_price": 50000
}
```

### 2.4 Update House
**PUT** `/api/houses/:id`
**Request Body:** (Tương tự Create House, cho phép update từng phần hoặc toàn bộ trường thông tin)

### 2.5 Delete House
**DELETE** `/api/houses/:id`

## 3. API Endpoints - Room Management

### 3.1 Get Rooms List
**GET** `/api/rooms`
**Query Parameters:**
- `house_id`: uuid (Bắt buộc hoặc tùy chọn để lọc theo nhà)
- `page`: int (default: 1)
- `limit`: int (default: 10)
- `search`: string (Tìm theo `name`)
- `status`: string ('AVAILABLE', 'OCCUPIED', 'MAINTENANCE')
- `sort`: string (VD: `room_price:asc`, `name:desc`)

### 3.2 Get Room Details
**GET** `/api/rooms/:id`

### 3.3 Create Room
**POST** `/api/rooms`
**Request Body:**
```json
{
  "house_id": "uuid",
  "name": "string",
  "room_price": 3000000,
  "max_tenants": 2,
  "status": "AVAILABLE"
}
```

### 3.4 Update Room
**PUT** `/api/rooms/:id`
**Request Body:** (Các trường muốn update)

### 3.5 Delete Room
**DELETE** `/api/rooms/:id`

## 4. Database Interaction
- **Houses Table**: Quản lý dựa trên `manager_id` (lấy từ JWT token của User hiện tại). Đảm bảo Manager chỉ xem và sửa được thông tin House của mình (Row-level authorization).
- **Rooms Table**: Ràng buộc Foreign Key `house_id` trỏ tới `houses.id`. Khi thao tác tạo/cập nhật/xóa Room, phải tiếp tục verify quyền sở hữu của Manager đối với bảng House chứa Room đó.

## 5. Flow chi tiết (Ví dụ: Create Room)
1. **Frontend** gửi request payload tạo Room (POST `/api/rooms`).
2. **Backend**:
   - Verify JWT Token qua middleware, trích xuất `user_id` hiện hành.
   - Validate payload dữ liệu đầu vào.
   - Truy xuất giá trị `house_id` từ request body.
   - Query DB lấy thông tin House dựa trên `house_id`. Kiểm tra `manager_id` của House này có trùng với `user_id` hiện hành hay không (Security Check).
   - Nếu điều kiện hợp lệ: Insert record mới vào bảng `rooms`.
3. Trả về thông tin Room đã được tạo (Kèm status HTTP 201 Created).

## 6. Todo
- [ ] Định nghĩa các Route group cho House và Room trong router.
- [ ] Implement request validation (e.g. Validator/Binding logic).
- [ ] Implement DB queries và Repository pattern cho House (List, Detail, Create, Update, Delete).
- [ ] Implement DB queries và Repository pattern cho Room.
- [ ] Thêm middleware check Role `MANAGER` bao bọc bên ngoài.
- [ ] Xây dựng common struct/helper hỗ trợ pagination, sorting, search filter.

## 7. Security
- [ ] Bắt buộc kiểm tra quyền sở hữu đối với từng thao tác đọc - ghi - xóa trên House và Room. Không cho phép thay đổi `house_id` sang một id không thuộc quyền sở hữu.
- [ ] Cẩn trọng với SQL Injection khi map các parameters `search`, `sort` trực tiếp vào SQL statement. Nên dùng query builder an toàn hoặc strict allow-list.
