# Hướng dẫn Triển khai (Deployment Guide)

Tài liệu này cung cấp hướng dẫn chi tiết các bước để thiết lập và khởi chạy dự án Quản lý phòng trọ trên cả môi trường Phát triển (Development) và môi trường Sản xuất (Production).

## Yêu cầu hệ thống (Prerequisites)
Trước khi bắt đầu, đảm bảo máy tính/server của bạn đã cài đặt:
- **Go** (version 1.20+)
- **Node.js** (version 18+) & **npm** (hoặc yarn/pnpm)
- **PostgreSQL** (version 14+)
- **golang-migrate**: Công cụ quản lý migration database cho Golang.
- *(Tùy chọn)* **Docker** & **Docker Compose** (Dành cho triển khai production bằng container).

---

## 1. Môi trường Phát triển (Development Environment)

Ở môi trường Dev, chúng ta sẽ chạy frontend và backend trực tiếp trên máy host và sử dụng tính năng live-reload (Air cho Backend và Vite cho Frontend).

### 1.1 Thiết lập Database
1. Cài đặt và khởi chạy PostgreSQL.
2. Tạo database cho dự án, ví dụ: `quanly_phongtro_dev`.
   ```sql
   CREATE DATABASE quanly_phongtro_dev;
   ```

### 1.2 Cấu hình Backend
1. Di chuyển vào thư mục gốc của dự án.
2. Copy file môi trường:
   ```bash
   cp .env.example .env
   ```
3. Cập nhật các thông số trong `.env` cho phù hợp với môi trường dev, đặc biệt là `POSTGRES_DSN`.
4. Chạy DB Migrations để tạo các bảng:
   ```bash
   set -a
   source .env
   set +a
   migrate -path migrations -database "$POSTGRES_DSN" up
   ```
5. Chạy Backend với **Air** (để tự động reload khi sửa code):
   ```bash
   # Nếu chưa cài air thì chạy: go install github.com/air-verse/air@latest
   air
   ```
   *Hoặc chạy trực tiếp không qua Air:*
   ```bash
   go run ./cmd/api
   ```
   API Server sẽ lắng nghe ở cổng mặc định (VD: 8080).

### 1.3 Cấu hình Frontend
1. Mở một terminal mới, di chuyển vào thư mục frontend:
   ```bash
   cd frontend
   ```
2. Cài đặt các gói phụ thuộc (dependencies):
   ```bash
   npm install
   ```
3. Copy và cấu hình file môi trường frontend (nếu có, ví dụ `.env.local`). Đảm bảo base URL trỏ về API Backend (VD: `http://localhost:8080`).
4. Khởi chạy dev server:
   ```bash
   npm run dev
   ```
   Frontend sẽ có thể truy cập tại `http://localhost:5173` (mặc định của Vite).

---

## 2. Môi trường Sản xuất (Production Environment)

Đối với môi trường Production, mã nguồn cần được tối ưu (build) và quản lý bằng các công cụ system (như systemd, PM2) hoặc Docker container hóa, kết hợp Nginx làm Reverse Proxy.

### 2.1 Chuẩn bị Server và Database
1. Khởi tạo một Server/VPS (Ubuntu/Debian...).
2. Cài đặt PostgreSQL (hoặc dùng dịch vụ Managed Database).
3. Tạo cơ sở dữ liệu và user với quyền hạn phù hợp (Tuyệt đối không dùng account `postgres` ở production).
4. Thực hiện apply DB Migrations (Tương tự như bước ở Dev).

### 2.2 Triển khai Backend (Native Build)
1. Tải source code lên server và build file thực thi binary:
   ```bash
   go build -o quanly-phongtro-api ./cmd/api
   ```
2. Cấu hình file `.env` trên server production với các biến bảo mật (DB Credentials mật khẩu mạnh, `JWT_SECRET` sinh ngẫu nhiên mạnh, port,...).
3. Quản lý process bằng **Systemd**:
   Tạo file `/etc/systemd/system/quanly-phongtro-api.service`:
   ```ini
   [Unit]
   Description=QuanLyPhongTro API Backend
   After=network.target postgresql.service

   [Service]
   Type=simple
   User=deploy
   WorkingDirectory=/path/to/project
   ExecStart=/path/to/project/quanly-phongtro-api
   Restart=on-failure

   [Install]
   WantedBy=multi-user.target
   ```
4. Kích hoạt và chạy service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable quanly-phongtro-api
   sudo systemctl start quanly-phongtro-api
   ```

### 2.3 Triển khai Frontend (Static Hosting)
1. Trong thư mục `frontend`, build mã nguồn sản xuất:
   ```bash
   npm install
   npm run build
   ```
2. Quá trình build sẽ tạo ra thư mục `dist/` (nếu dùng Vite).
3. Cấu hình **Nginx** để serve thư mục static này và proxy các request bắt đầu bằng `/api` vào Backend:
   Tạo file cấu hình nginx `/etc/nginx/sites-available/quanly-phongtro`:
   ```nginx
   server {
       listen 80;
       server_name yourdomain.com;

       root /path/to/project/frontend/dist;
       index index.html;

       # Serve static files for React
       location / {
           try_files $uri /index.html;
       }

       # Proxy request tới Backend API
       location /api/ {
           proxy_pass http://localhost:8080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       }
   }
   ```
4. Kích hoạt site Nginx và khởi động lại:
   ```bash
   sudo ln -s /etc/nginx/sites-available/quanly-phongtro /etc/nginx/sites-enabled/
   sudo nginx -t
   sudo systemctl restart nginx
   ```

### 2.4 Cập nhật dự án (Update)
Khi có phiên bản code mới, quy trình sẽ là:
1. `git pull` code mới.
2. Chạy `migrate up` nếu có thay đổi DB Schema.
3. Build lại Backend (`go build...`) và restart service (`systemctl restart quanly-phongtro-api`).
4. Build lại Frontend (`npm run build`) (Nginx tự động serve các file mới nhất).

---

> **Lưu ý mở rộng (Docker):**
> Dự án cũng định hướng hỗ trợ Docker/Docker Compose để đóng gói (Containerization). Khi các file `Dockerfile` và `docker-compose.yml` được viết sẵn, quy trình triển khai Production chỉ đơn giản là `docker-compose up -d --build`, giúp chuẩn hóa môi trường tối đa và không phụ thuộc hệ điều hành máy host.
