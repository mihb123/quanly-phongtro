# Hướng dẫn chạy test frontend

Tài liệu này mô tả kiểm tra tĩnh cho toàn bộ frontend và bài E2E SePay chạy bằng Puppeteer trên SePay Test Mode. Không lưu tài khoản, mật khẩu, API Token hoặc Webhook Secret vào source code hay commit Git.

## 1. Chuẩn bị

- Node.js 20 trở lên và pnpm.
- Go theo phiên bản trong `go.mod` để chạy API local.
- PostgreSQL và file `.env` ở thư mục gốc đã được cấu hình theo `.env.example`.
- Một tài khoản quản lý của ứng dụng local.
- Một tài khoản SePay có quyền dùng Test Mode.

Cài dependency frontend một lần:

```bash
cd frontend
pnpm install
```

Puppeteer tải Chrome tương thích trong quá trình cài dependency. Nếu đã cài với tùy chọn bỏ qua script, chạy lại `pnpm install` mà không đặt `PUPPETEER_SKIP_DOWNLOAD`.

## 2. Kiểm tra tĩnh

Từ thư mục `frontend`, chạy ESLint và TypeScript:

```bash
pnpm test:frontend
```

Có thể chạy riêng từng bước:

```bash
pnpm lint
pnpm typecheck
```

Lệnh `typecheck` kiểm tra cả `tsconfig.app.json` và `tsconfig.node.json`, không tạo file đầu ra.

## 3. Khởi động môi trường E2E

E2E SePay cần backend và frontend cùng hoạt động. Từ thư mục gốc, mở terminal thứ nhất:

```bash
go run ./cmd/api
```

Mở terminal thứ hai:

```bash
cd frontend
pnpm dev --host 127.0.0.1 --port 5173
```

Vite đọc `APP_PORT` từ file `.env` ở thư mục gốc và proxy `/api` tới backend. Nếu frontend không đăng nhập được, kiểm tra cổng backend trong `.env` khớp với tiến trình đang chạy.

## 4. Chạy E2E SePay Test Mode

Đặt credential trong biến môi trường của terminal; không ghi chúng vào file test:

```bash
cd frontend

export SEPAY_APP_URL='http://127.0.0.1:5173'
export SEPAY_APP_EMAIL='<email-quan-ly-local>'
export SEPAY_APP_PASSWORD='<mat-khau-quan-ly-local>'
export SEPAY_LOGIN_EMAIL='<email-sepay-test>'
export SEPAY_LOGIN_PASSWORD='<mat-khau-sepay-test>'

pnpm test:e2e:sepay
```

Nếu thiếu một trong bốn biến credential, Node test runner sẽ báo bài test ở trạng thái `SKIP`; đây không phải kết quả pass.

Bài test thực hiện tuần tự:

1. Đăng nhập `my.sepay.vn` và bật Test Mode.
2. Đọc tài khoản ngân hàng sandbox, xóa các API key cũ có tiền tố riêng của E2E và tạo một key mới.
3. Kiểm tra token với SePay Sandbox API.
4. Đăng nhập ứng dụng local, chọn ngân hàng từ danh sách SePay hỗ trợ và lưu cấu hình sandbox.
5. Xác nhận backend đối chiếu số tài khoản và tên chủ tài khoản từ danh sách tài khoản đã liên kết của SePay.
6. Kiểm tra webhook sai tài khoản bị từ chối, chữ ký HMAC quá hạn bị từ chối và webhook hợp lệ được chấp nhận.
7. Chạy đối soát giao dịch và kiểm tra giao diện desktop/mobile không tràn ngang.

Ảnh bằng chứng được ghi tại:

- `e2e/artifacts/sepay-settings-desktop.png`
- `e2e/artifacts/sepay-settings-mobile.png`

Báo cáo HTML tổng hợp nằm tại `../Documents/reports/sepay_test_report.html`.

## 5. Xử lý lỗi thường gặp

- `net::ERR_CONNECTION_REFUSED`: backend hoặc Vite chưa chạy, hoặc `APP_PORT` không khớp.
- Test bị `SKIP`: thiếu biến môi trường credential.
- Không vào được Test Mode: tài khoản SePay chưa có quyền Test Mode hoặc giao diện SePay đã thay đổi selector.
- HTTP `401` từ SePay Sandbox: token Test Mode không hợp lệ hoặc đã bị thu hồi.
- Không xác thực được tài khoản: số tài khoản/ngân hàng không nằm trong danh sách tài khoản đã liên kết với đúng công ty và đúng môi trường của API Token.
- Không tìm thấy Chrome: chạy lại `pnpm install` để Puppeteer tải browser tương thích.

## 6. Phạm vi xác thực tên chủ tài khoản

SePay API v2 chỉ trả thông tin các tài khoản đã được liên kết với công ty đang sở hữu API Token. Hệ thống dùng endpoint này để xác nhận cặp ngân hàng + số tài khoản và thay tên nhập tay bằng `account_holder_name` chính thức khi có token. SePay không công bố endpoint tra cứu tên chủ tài khoản bất kỳ chỉ từ mã ngân hàng và số tài khoản; vì vậy cấu hình không có API Token sẽ được lưu nhưng không được gắn trạng thái đã xác thực qua SePay.
