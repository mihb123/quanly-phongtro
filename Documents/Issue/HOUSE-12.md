# Email OTP Verification Flow

## 1. Overview

Hệ thống sử dụng **Email (SMTP)** để gửi mã OTP xác thực qua thư điện tử. Flow này cho phép người dùng đăng nhập hoặc đăng ký bằng email, sau đó nhận mã xác thực qua Email để kích hoạt (enable) tài khoản.

## 2. Token / OTP Definition

### OTP Code
- **Dạng:** Chuỗi số ngẫu nhiên (VD: 6 chữ số)
- **TTL:** 5 phút
- **Dùng để:** Xác thực quyền sở hữu email và kích hoạt tài khoản
- **Lưu trong:** DB (table `user_otps`) hoặc Cache (Redis)

## 3. API Endpoints

### 3.1 Request OTP (Đăng nhập / Gửi lại mã)
**POST** `/api/request-otp`

**Request:**
```json
{
  "email": "string"
}
```

**Response:**
```json
{
  "message": "Mã OTP đã được gửi qua Email",
  "expires_in": 300
}
```

### 3.2 Verify Email OTP (Kích hoạt tài khoản)
**POST** `/api/verify-email`

**Request:**
```json
{
  "email": "string",
  "otp_code": "string"
}
```

**Response:**
```json
{
  "message": "Tài khoản đã được kích hoạt thành công",
  "access_token": "string",
  "refresh_token": "string",
  "expires_in": 3600
}
```
*(Nếu là luồng đăng nhập, API có thể trực tiếp trả về `access_token` và `refresh_token` sau khi verify thành công).*

## 4. Flow chi tiết

### Step 1: Đăng nhập bằng Email & Request OTP
1. FE gửi `email` (có thể gửi cả `password` tuỳ design)
2. BE:
   - Validate thông tin user (nếu account chưa active thì tiến hành gửi OTP).
   - Random sinh ra mã `otp_code`.
   - Lưu `otp_code` vào DB (kèm `email` và `expires_at`).
   - Gọi Email Service (SMTP) để gửi thư điện tử chứa mã OTP đến email đó.
3. Trả về thông báo thành công cho FE.

### Step 2: Nhập mã OTP
1. Người dùng mở Email, lấy mã OTP và nhập vào app/web.
2. FE gửi thông tin đến `POST /api/verify-email`.

### Step 3: BE Verify OTP
1. BE nhận request và truy vấn `otp_code` theo `email`:
   - Nếu không tồn tại -> Báo lỗi.
   - Nếu `expires_at` < hiện tại -> Báo `OTP đã hết hạn`.
   - Nếu thông tin trùng khớp:
     - Cập nhật trạng thái account là đã kích hoạt (`is_active = true`).
     - Delete / Mark as used mã OTP để tránh reuse.
     - Tạo `access_token` và `refresh_token` (giống cơ chế ở HOUSE-11).
2. Trả về kết quả và JWT tokens cho FE.

### Step 4: OTP hết hạn hoặc sai mã (Error Handling)
- BE trả: `400 Bad Request` hoặc `401 Unauthorized` kèm theo câu lỗi chi tiết cho người dùng.

## 5. Database Design

### Table: `user_otps`

| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | uuid | Primary key |
| `email` | string | Email nhận OTP |
| `otp_code` | string | Mã OTP |
| `expires_at` | datetime | Thời gian hết hạn của OTP |
| `is_used` | boolean | Đã sử dụng hay chưa |
| `created_at` | datetime | Created time |


### Table: `users` (Cập nhật)

| Field | Type | Description |
| :--- | :--- | :--- |
| `is_active` | boolean | Trạng thái xác thực (true: đã xác thực, false: chưa xác thực) |


## 6. Todo
- [ ] Implement `POST /api/request-otp`
- [ ] Tích hợp SMTP Email Service để thực hiện gửi thư.
- [ ] Implement `POST /api/verify-email`
- [ ] Implement logic tạo access_token/refresh_token sau khi xác thực thành công (tái sử dụng utility từ lệnh POST `/api/login`).

## 7. Security
- [ ] **Rate Limiting OTP:** Giới hạn API Request OTP (ví dụ 1 email chỉ được gửi tối đa 3 lần/5 phút) để tránh bị spam mail.
- [ ] **Short TTL:** Thời gian sống của mã OTP nên ngắn (3 đến 5 phút).
- [ ] **Brute-force Prevention:** Giới hạn số lần verify sai. (Ví dụ: sai OTP quá 5 lần thì khoá tính năng nhập của user đó trong 15 phút).
- [ ] Xoá hoặc vô hiệu hoá (Invalidate) OTP ngay sau khi verify thành công.
