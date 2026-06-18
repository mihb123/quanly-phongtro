# Kế hoạch triển khai DPoP (Demonstrating Proof of Possession)

Kế hoạch này nhằm nâng cấp tính bảo mật cho hệ thống xác thực của dự án `quanly-phongtro` bằng cách sử dụng cơ chế DPoP. Mục tiêu chính là khóa chặt Access Token và Refresh Token với Public Key của thiết bị người dùng. Kẻ tấn công dù có trộm được token cũng không thể sử dụng nếu không có Private Key nằm trên thiết bị thật.

## User Review Required

> [!IMPORTANT]
> - **Cấu trúc Database**: Chúng ta sẽ nâng cấp bảng `jwt_refresh_tokens` hiện tại thành bảng `auth_sessions`. Bảng mới sẽ có thêm các cột `ip_address`, `user_agent`, `jkt` (DPoP Thumbprint) và `expires_at`. Điều này giúp quản lý rủi ro thiết bị hiệu quả và hỗ trợ tính năng "Đăng xuất khỏi các thiết bị khác". (Ghi chú: Không cần thêm `last_login_ip` vào bảng `users` vì có thể truy vấn trực tiếp từ bảng `auth_sessions`).
> - **Thư viện JWT Backend**: Backend sẽ tận dụng `golang-jwt/jwt/v5` hiện có để parse JWK và verify chữ ký ECDSA từ DPoP header, không cần cài thêm thư viện nặng như `jwx/v2`.
> - **Phạm vi bảo mật**: DPoP (và tham số `ath` - hash của token) sẽ được áp dụng cho **tất cả API**, kể cả các request `multipart/form-data` upload file để đảm bảo an toàn tuyệt đối 100%. Axios Interceptor phía Frontend sẽ tự động handle việc đính kèm.

## Proposed Changes

---

### Backend (Go)

Thay đổi chính nằm ở Database Schema, middleware và các hàm xử lý JWT.

#### [MODIFY] Database Schema & Models
- Đổi tên bảng `jwt_refresh_tokens` thành `auth_sessions`.
- Cập nhật struct `model.JWTRefreshToken` (hoặc đổi thành `AuthSession`) để có thêm các trường: `IPAddress`, `UserAgent`, `JKT`, `ExpiresAt`.
- Migration script để cập nhật database schema.

#### [NEW] `internal/security/dpop.go`
- Viết package `dpop` để chuyên parse và verify DPoP proof.
- Chứa logic: Verify signature bằng Public Key JWK lấy từ header.
- Chứa logic In-memory cache (sử dụng package `patrickmn/go-cache` hoặc map có lock) để lưu trữ `jti` (chống Replay Attack) với TTL = 5 phút.

#### [MODIFY] `internal/security/jwt.go`
- Thêm trường `Cnf map[string]string` vào struct `Claims` (để chứa `jkt`).
- Thay đổi hàm `GenerateAccessToken` và `GenerateRefreshToken` để nhận thêm tham số `jkt` (JWK Thumbprint). Đưa `jkt` này vào token claims.

#### [MODIFY] `internal/handler/auth_handler.go`
- Tại hàm `Login` và `Register`, backend sẽ đọc header `DPoP` để lấy `jkt` từ public key do Frontend sinh ra.
- Truyền `jkt` này xuống `AuthService` để issue token có khóa `jkt`.
- Hàm `RefreshToken` cũng cần verify DPoP và check `jkt`.

#### [MODIFY] `internal/router/auth_middleware.go`
- Đọc thêm header `DPoP`.
- Xác thực Access Token xong, lấy `cnf.jkt` từ claim.
- Pass `DPoP` string và `cnf.jkt` vào hàm verify của `internal/security/dpop.go`.
- Kiểm tra `htm` (phương thức HTTP, vd GET, POST) và `htu` (đường dẫn URL hiện tại) xem có khớp với request đang đi vào không.

---

### Frontend (React)

Thay đổi chủ yếu nằm ở interceptor và hàm khởi tạo key.

#### [NEW] `frontend/src/utils/dpop.ts`
- Sử dụng Web Crypto API để tạo cặp key ECDSA P-256.
- Lưu Private Key với thuộc tính `extractable: false` vào IndexedDB (ví dụ qua thư viện `idb`).
- Hàm `getDPoPProof(url, method, accessToken?)` để sinh ra DPoP header dạng JWT được ký bởi Private Key.
- Trả về `jkt` (Thumbprint của Public Key).

#### [MODIFY] `frontend/src/api/client.tsx`
- Bổ sung Axios Interceptor: Trước mỗi request, interceptor sẽ gọi `getDPoPProof()` để tính chữ ký và đính kèm vào header `DPoP`.

#### [MODIFY] `frontend/src/api/auth.tsx`
- Đảm bảo rằng key pair được sinh ra sẵn trước khi gọi hàm `login` hoặc `register`. `DPoP` proof sẽ được gắn ngay từ API đầu tiên này để backend có thể bind được token.

---

## Verification Plan

### Automated Tests
- Cập nhật test trong `internal/router/auth_middleware_test.go` để giả lập request có DPoP và không có DPoP.
- Test in-memory `jti` replay cache: gửi 2 request với cùng 1 `jti` phải bị reject ở request thứ 2.

### Manual Verification
1. Login từ UI React: Kiểm tra DevTools Network xem header `DPoP` có xuất hiện ở API `/auth/login` không.
2. Kiểm tra Payload JWT của `access_token` xem có node `cnf: { jkt: "..." }` không.
3. Reload trang F5: Interceptor lấy lại key từ IndexedDB và ký cho `/auth/me`, đảm bảo không lỗi.
4. Lấy cắp Cookie `access_token` dán vào Postman và call API mà không có `DPoP` header hợp lệ -> Server phải báo lỗi `401 Unauthorized`.
