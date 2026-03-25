# Quanly-Phongtro - Frontend 🏠

Ứng dụng quản lý phòng trọ được xây dựng giao diện với **React**, **TypeScript**, và **Vite**.

## 🛠 Prerequisites

Trước khi bắt đầu, hãy đảm bảo bạn đã cài đặt:
- [Node.js](https://nodejs.org/) (Khuyến nghị phiên bản 20 trở lên)
- [pnpm](https://pnpm.io/) (Dùng để quản lý package, cài đặt với `npm install -g pnpm`)

## 🚀 Setup & Installation

Thực hiện các bước sau để khởi chạy dự án tại local:

### 1. Cấu hình Environment
Phần frontend sử dụng cấu hình chung tại file `.env` ở **thư mục gốc** (parent directory) của dự án. Đảm bảo bạn đã sao chép và cấu hình đúng cổng của Backend:

```bash
# Tại thư mục gốc của toàn bộ dự án
cp .env.example .env
```

Vite sẽ tự động đọc `APP_PORT` từ `.env` này để kết nối Proxy tới Backend.

### 2. Cài đặt Dependencies
Di chuyển vào thư mục `frontend` và cài đặt các thư viện cần thiết:

```bash
cd frontend
pnpm install
```

### 3. Chạy Development Server
Khởi chạy frontend ở chế độ phát triển:

```bash
pnpm dev
```

Mặc định ứng dụng sẽ chạy tại địa chỉ [http://localhost:5173](http://localhost:5173).

## 📁 Project Structure

- `src/api`: Chứa các hàm gọi API (Auth, User, etc.)
- `src/components`: Các UI components dùng chung (shadcn/ui)
- `src/hooks`: Custom React hooks
- `src/pages`: Các trang giao diện chính (Login, Register, Dashboard)
- `src/router`: Cấu hình định tuyến với React Router 7

## 🏗 Build for Production

Để tạo bản build tối ưu cho môi trường production:

```bash
pnpm build
```

Kết quả sẽ nằm trong thư mục `dist/`.

---
> [!NOTE]
> Để backend hoạt động đầy đủ, hãy đảm bảo bạn đã khởi chạy cả Go API phục vụ tại cổng đã cấu hình trong `.env`.
