# Cách dự án lấy vị trí tọa độ (GPS) của User

Dự án hiện tại đang lấy thông tin tọa độ GPS của người dùng tại 2 luồng chính: **Đăng nhập** (`frontend/src/pages/Login.tsx`) và **Đăng ký** (`frontend/src/pages/Register.tsx`). 

Mục tiêu chung của chức năng này là lấy được tọa độ (Latitude và Longitude) trước khi thực hiện gọi API đăng nhập/đăng ký, nhằm lưu lại vị trí của người dùng lúc thực hiện hành động.

Dưới đây là chi tiết luồng xử lý:

## 1. Kiểm tra điều kiện và quyền (Permissions)
Trước khi yêu cầu GPS, hệ thống sẽ kiểm tra xem có cần thiết/có thể lấy được vị trí hay không bằng cách xét biến cờ `shouldFetchLocation`:

- Kiểm tra thiết bị có hỗ trợ tính năng định vị không: `'geolocation' in navigator`.
- Hệ thống sử dụng một cờ trong `localStorage` tên là `has_asked_location` để hạn chế việc liên tục popup hỏi quyền người dùng:
  - Nếu chưa từng hỏi (`!localStorage.getItem('has_asked_location')`), hệ thống sẽ quyết định lấy vị trí (và trình duyệt sẽ tự động hiện popup hỏi quyền).
  - Nếu đã từng hỏi trước đó, hệ thống sẽ kiểm tra xem quyền định vị (geolocation) đang ở trạng thái nào thông qua `navigator.permissions.query({ name: 'geolocation' })`. Chỉ khi người dùng đã cấp quyền (`state === 'granted'`), hệ thống mới tiếp tục gọi hàm lấy vị trí.

## 2. Gọi hàm lấy tọa độ (getCurrentPosition)
Khi xác định cần lấy vị trí, mã nguồn sử dụng Web API `navigator.geolocation.getCurrentPosition` nhưng được bọc trong các `Promise` kèm theo cơ chế Timeout để **không làm treo (block) quá lâu** luồng đăng nhập/đăng ký của người dùng.

- **Tại trang Đăng nhập (`Login.tsx`):**
  Sử dụng một hàm custom tên là `getBestEffortPosition()`.
  Hàm này thiết lập một khoảng thời gian chờ tối đa (Timeout) là **4000ms (4 giây)**. 
  - Nếu trong 4 giây lấy được tọa độ (hoặc bị từ chối), `Promise` sẽ resolve.
  - Nếu quá 4 giây mà trình duyệt vẫn đang loay hoay chưa lấy được vị trí, hệ thống sẽ tự động ngắt và trả về `null` (Bỏ qua việc lấy vị trí để tiếp tục luồng đăng nhập). 
  - Cấu hình bắt GPS cũng cho phép dùng vị trí cũ (cache) trong vòng 10 phút (`maximumAge: 600000`) để tăng tốc độ.

- **Tại trang Đăng ký (`Register.tsx`):**
  Trực tiếp bọc `getCurrentPosition` trong một `Promise` với timeout là **10000ms (10 giây)**. 

## 3. Xử lý sau khi có kết quả (hoặc thất bại)
- Nếu lấy **thành công**, hệ thống sẽ bóc tách `latitude = position.coords.latitude` và `longitude = position.coords.longitude`. Đồng thời, lưu cờ `localStorage.setItem('has_asked_location', 'true')` để ghi nhớ đã hỏi.
- Nếu **thất bại** (người dùng từ chối, hết timeout, hoặc lỗi thiết bị):
  - Hệ thống vẫn lưu cờ vào `localStorage` là đã hỏi.
  - **Môi trường Development:** Nếu đang chạy ở môi trường phát triển (biến `import.meta.env.DEV` là true), hệ thống sẽ tự động gán tọa độ giả (mock) là vị trí ở Hà Nội:
    `latitude = 21.028511`
    `longitude = 105.804817`
  - **Môi trường Production:** Tọa độ sẽ bị bỏ trống (`undefined`/`null`).

## 4. Gửi dữ liệu lên API
Cuối cùng, các giá trị `Latitude` và `Longitude` thu thập được sẽ được đính kèm vào payload (cùng với email và password) và gửi qua hàm API `loginAccount` (khi đăng nhập) hoặc `registerAccount` (khi đăng ký) để lưu lên hệ thống cơ sở dữ liệu.
