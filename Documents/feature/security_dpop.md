# Tài Liệu Kỹ Thuật: Cơ Chế Bảo Mật DPoP (Demonstrating Proof-of-Possession)

Tài liệu này giải thích chi tiết về cơ chế bảo mật xác thực của dự án, nhằm giúp các lập trình viên mới (cả Frontend và Backend) dễ dàng nắm bắt được luồng hoạt động, cấu trúc dữ liệu và cách hệ thống bảo vệ thông tin người dùng khỏi các cuộc tấn công.

---

## 1. Vấn Đề Của JWT Thông Thường (Bearer Token)
Theo cách truyền thống, hệ thống sử dụng **Bearer Token** (thường là JWT). Đặc tính của Bearer token là *"Ai cầm được token thì người đó có quyền sử dụng"*.
- Nếu hacker đánh cắp được `access_token` (ví dụ thông qua lỗ hổng XSS trên trình duyệt hoặc qua mạng), hacker có thể dùng token đó để gọi API dưới quyền của bạn.
- Tương tự, nếu `refresh_token` bị lộ, hacker có thể tạo ra vô số `access_token` mới.

**Giải pháp:** Áp dụng **DPoP (Demonstrating Proof-of-Possession)**.
Cơ chế này "trói" (bind) token vào một thiết bị cụ thể. Dù hacker có ăn cắp được `access_token` hay `refresh_token`, họ cũng **không thể** sử dụng được nếu không có "chìa khóa riêng" (Private Key) được lưu ẩn giấu trên thiết bị của bạn.

---

## 2. Giải Thích Các Thuật Ngữ Dành Cho Người Mới

- **Key Pair (Cặp khóa):** Bao gồm một **Public Key** (Khóa công khai) và một **Private Key** (Khóa bí mật). Private Key dùng để ký điện tử, Public Key dùng để kiểm tra chữ ký đó.
- **JWK (JSON Web Key):** Là định dạng JSON để biểu diễn các khóa mật mã (Public Key).
- **JWK Thumbprint (`jkt`):** Là một chuỗi băm (hash) đại diện duy nhất cho một Public Key. Thay vì lưu toàn bộ Public Key lớn, hệ thống chỉ cần lưu `jkt` cho nhẹ.
- **DPoP Proof:** Là một chuỗi JWT đặc biệt do Frontend tạo ra và gửi kèm trong Header `DPoP` mỗi khi gọi API. Nó chứa thông tin chứng minh rằng *"Tôi đang nắm giữ Private Key"*.
- **`cnf` (Confirmation):** Là một trường (claim) đặc biệt bên trong `access_token` (payload). Trường `cnf` chứa giá trị `jkt`, giúp Backend biết token này chỉ được phép dùng với trình duyệt có Public Key tương ứng.
- **`jti` (JWT ID):** Mã định danh duy nhất của một DPoP Proof. Giúp Backend chặn **Replay Attack** (hacker copy nguyên xi request cũ và gửi lại).

---

## 3. Luồng Hoạt Động (Workflow)

### Bước 1: Khởi Tạo Khóa Tại Frontend (Trình duyệt/App)
Khi người dùng mở ứng dụng, Frontend (cụ thể trong `frontend/src/utils/dpop.tsx`) sẽ tự động tạo một cặp khóa thuật toán ECDSA P-256 thông qua Web Crypto API.

**Cách Frontend lưu trữ và bảo vệ Private Key:**
- **Lưu trữ ở đâu?** Khóa được lưu trữ tại **IndexedDB** của trình duyệt (sử dụng thư viện `idb-keyval`). IndexedDB được bảo vệ bởi cơ chế **Same-Origin Policy**, nghĩa là các trang web khác (domain khác) hoàn toàn không thể truy cập vào kho chứa khóa này.
- **Làm sao để không bị đánh cắp?** Điểm mấu chốt là khi khởi tạo khóa bằng `window.crypto.subtle.generateKey`, Frontend đã thiết lập tham số `extractable = false`.
  - Thiết lập này ra lệnh cho trình duyệt: **"Tuyệt đối không cho phép bất kỳ mã lệnh nào đọc được dữ liệu gốc (raw bytes) của Private Key"**.
  - Nếu ứng dụng chẳng may bị dính lỗ hổng **XSS (Cross-Site Scripting)**, hacker có thể chạy mã JS độc hại để truy cập IndexedDB, nhưng chúng chỉ lấy ra được một đối tượng `CryptoKey` dạng tham chiếu.
  - Hacker có thể xài đối tượng này để ký request *ngay lúc đó* trên trình duyệt của nạn nhân, nhưng **không thể trích xuất (export)** để mang Private Key về máy của chúng. Khi nạn nhân đóng trình duyệt, hacker mất quyền kiểm soát. Đây là lớp khiên cực kỳ mạnh mẽ bảo vệ khóa ngay cả khi ứng dụng có lỗ hổng.

- **Public Key:** Ngược lại với Private Key, khóa công khai được chuyển thành định dạng chuẩn **JWK** và thoải mái đính kèm vào các DPoP Proof để gửi lên Backend xác thực.

### Bước 2: Đăng Nhập / Đăng Ký (Login/Register)
1. **Frontend:** 
   - Lấy email, password.
   - Tạo ra một **DPoP Proof** (JWT) chứa Public Key (JWK) và ký nó bằng Private Key.
   - Gửi Request lên API:
     ```http
     POST /auth/login
     DPoP: <chuỗi_dpop_proof_jwt>
     
     { "email": "...", "password": "..." }
     ```
2. **Backend:**
   - Trích xuất Public Key từ header `DPoP`.
   - Tính toán **JWK Thumbprint (`jkt`)**.
   - Nếu đăng nhập đúng, Backend sinh ra `access_token`. Trong payload của `access_token`, Backend nhét thêm `jkt` vào claim `cnf`.
   - Tạo một **Auth Session** mới cho `refresh_token` và lưu vào Database (Bảng `auth_sessions`). Bảng này lưu thêm: `jkt`, `ip_address`, và `user_agent` của thiết bị.

### Bước 3: Truy Cập Các API Yêu Cầu Xác Thực
Khi Frontend muốn lấy danh sách phòng trọ (`GET /houses`), Frontend phải:
1. **Tạo DPoP Proof mới:**
   - Chứa URL đang gọi: `htu` = `https://api.domain.com/houses`
   - Chứa HTTP Method: `htm` = `GET`
   - Chứa thời gian tạo: `iat`
   - Chứa ID ngẫu nhiên: `jti`
   - Ký bằng **Private Key**.
2. **Gửi Request:**
   ```http
   GET /houses
   Authorization: DPoP <access_token>
   DPoP: <chuỗi_dpop_proof_mới_tạo>
   ```
3. **Backend Xác Nhận (Middleware):**
   - Lấy `access_token` và giải mã, lấy ra `jkt` từ trường `cnf`.
   - Lấy header `DPoP` ra và verify (xác thực chữ ký) để thu được Public Key của Frontend.
   - Tính `jkt` từ Public Key thu được, so sánh xem có **trùng khớp** với `jkt` nằm trong `access_token` không.
   - Kiểm tra xem URL (`htu`) và Method (`htm`) trong DPoP Proof có khớp với request hiện tại không.
   - Kiểm tra xem `jti` đã từng được sử dụng trong 5 phút qua chưa (chống Replay Attack).
   - Nếu tất cả hợp lệ -> Cho phép truy cập.

---

## 4. Chi Tiết Payload Của Các Token

### 1. DPoP Proof Header (Frontend gửi lên)
Header của JWT này phải có tham số `typ` là `dpop+jwt` và chứa `jwk`.
```json
// DPoP Proof Header
{
  "typ": "dpop+jwt",
  "alg": "ES256",
  "jwk": {
    "kty": "EC",
    "crv": "P-256",
    "x": "...",
    "y": "..."
  }
}
```

```json
// DPoP Proof Payload
{
  "jti": "-Bnd2wgG_...",    // Chuỗi ngẫu nhiên duy nhất cho request này
  "htm": "GET",              // Method HTTP
  "htu": "https://...",      // URL đích
  "iat": 1686000000          // Unix Timestamp lúc tạo
}
```

### 2. Access Token (Backend trả về)
Là một JWT thông thường, nhưng có thêm mục `cnf`.
```json
// Access Token Payload
{
  "sub": "user_id_123",
  "email": "test@test.com",
  "role": "manager",
  "cnf": {
    "jkt": "0ZcOCORZNYy-DWpqq30jZyJGHTN0d2HglBV3uiguA4I" // Đại diện cho Public Key
  },
  "exp": 1686003600
}
```

---

## 5. Quản Lý Session Và Thiết Bị
Thay vì chỉ lưu một chuỗi `refresh_token` trên Database, hệ thống hiện tại sử dụng bảng `auth_sessions`.
Mỗi lần người dùng Login, hệ thống tạo một phiên bản (session) bao gồm:
- `user_id`: Người đăng nhập.
- `refresh_token`: Token dùng để xin cấp lại access_token mới.
- `jkt`: JWK Thumbprint gắn chặt với thiết bị lúc đăng nhập.
- `ip_address`: Địa chỉ IP của thiết bị.
- `user_agent`: Thông tin trình duyệt (Chrome, Safari, iOS, Android...).
- `expires_at`: Thời hạn của phiên làm việc.
- `revoked`: Trạng thái bị thu hồi hay chưa.

**Lợi ích:**
- Cho phép người dùng theo dõi các thiết bị đang đăng nhập của mình.
- Cho phép Admin hoặc người dùng tự "Đăng xuất từ xa" (Revoke session) một thiết bị đáng ngờ.
- Nếu hacker đánh cắp `refresh_token`, khi gọi API `/refresh` mà hacker không có **Private Key** tạo ra đúng `jkt`, Backend sẽ lập tức từ chối và có thể đánh dấu phiên này là nguy hiểm.

---

## 6. Tổng Kết

Nhờ DPoP, bảo mật của dự án được nâng lên một tầm cao mới:
1. **Chống trộm Token (Token Theft):** Token (`access_token` hay `refresh_token`) bị lộ cũng vô dụng. Private Key được khóa chặt trong thiết bị dưới dạng `non-extractable`, hacker dù khai thác được XSS cũng không thể lấy cắp (export) Private Key mang sang thiết bị khác sử dụng.
2. **Chống Replay Attack:** Hacker chép lại cả request hợp lệ cũng thất bại, vì mỗi DPoP proof chứa một ID (`jti`) chỉ xài được 1 lần, và có thời gian sống (`iat`) ngắn ngủi.
3. **Quản lý thiết bị minh bạch:** Quản trị được thông tin trình duyệt, IP, mang lại cảm giác an toàn và giống các hệ thống lớn (Google, Facebook).
