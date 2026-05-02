# JWT Authentication Flow (Access Token + Refresh Token)

## 1. Overview

Hệ thống sử dụng cơ chế xác thực **JWT stateless** kết hợp với **Refresh Token**.

## 2. Token Definition

### Access Token
- **Dạng:** JWT
- **TTL:** 1 giờ
- **Dùng để:** Authenticate API

### Refresh Token
- **Dạng:** random string (hoặc JWT)
- **TTL:** 30 ngày
- **Lưu trong:** DB (table `refresh_tokens`)

## 3. API Endpoints

### 3.1 Login
**POST** `/api/login`

**Request:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response:**
```json
{
  "access_token": "string",
  "refresh_token": "string",
  "expires_in": 3600
}
```

### 3.2 Call API
**GET** `/api/*`

**Header:**
```
Authorization: Bearer <access_token>
```

### 3.3 Refresh Token
**POST** `/api/token/refresh`

**Request:**
```json
{
  "refresh_token": "string"
}
```

**Response:**
```json
{
  "access_token": "string",
  "refresh_token": "string",
  "expires_in": 3600
}
```

### 3.4 Logout
**POST** `/api/logout`

**Request:**
```json
{
  "refresh_token": "string"
}
```

## 4. Flow chi tiết

### Step 1: Login
1. FE gửi username/password
2. BE:
   - Validate user
   - Tạo access_token (JWT)
   - Tạo refresh_token
   - Lưu refresh_token vào DB
3. Trả về token cho FE

### Step 2: Call API
1. FE gửi request kèm access_token
2. BE:
   - Verify JWT
   - Check expiration
   - Extract user info
3. Nếu hợp lệ → trả data

### Step 3: Token hết hạn
- BE trả: `401 Unauthorized`

### Step 4: Refresh Token
1. FE nhận 401
2. Gọi `/api/token/refresh`
3. BE:
   - Kiểm tra refresh_token:
     - Tồn tại trong DB
     - Chưa hết hạn
     - Chưa bị revoke
   - Tạo access_token mới
   - (Optional) rotate refresh_token
4. Trả token mới

### Step 5: Retry request
- FE gửi lại request cũ với token mới

## 5. Database Design

### Table: `refresh_tokens`

| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | uuid | Primary key |
| `user_id` | uuid | FK user |
| `refresh_token` | string | Refresh token |
| `expires_at` | datetime | Expiration |
| `revoked` | boolean | Đã revoke chưa |
| `created_at` | datetime | Created time |

## 6. Todo
- [ ] Implement `POST /api/login`
- [ ] Apply JWT middleware cho tất cả `/api/*` trừ login
- [ ] Implement `POST /api/token/refresh`
- [ ] Implement `POST /api/logout`

## 7. Security
- [ ] Hash refresh_token trong DB
- [ ] Set short TTL cho access_token
- [ ] Validate expire trong JWT
- [ ] Rate limit login API
- [ ] Limit refresh attempts
