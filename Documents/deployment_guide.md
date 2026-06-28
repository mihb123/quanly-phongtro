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

Ở Production, frontend được build và **nhúng thẳng vào binary Go**. Toàn bộ ứng dụng chạy bằng **một file thực thi duy nhất** vừa phục vụ API vừa phục vụ giao diện trên cùng một origin — **không cần Nginx** hay web server tĩnh riêng.

### 2.1 Chuẩn bị Server và Database
1. Khởi tạo một Server/VPS (Ubuntu/Debian...).
2. Cài đặt PostgreSQL (hoặc dùng dịch vụ Managed Database).
3. Tạo cơ sở dữ liệu và user với quyền hạn phù hợp (Tuyệt đối không dùng account `postgres` ở production).
4. Thực hiện apply DB Migrations (Tương tự như bước ở Dev).

### 2.2 Build một binary duy nhất (frontend nhúng trong backend)
1. Tải source code lên server. Cần có sẵn Go và pnpm (hoặc npm).
2. Build một phát bằng Makefile — lệnh này build frontend, copy `frontend/dist` vào `internal/web/dist`, rồi `go build`:
   ```bash
   make build
   ```
   Kết quả là file `quanly-phongtro-api` đã nhúng cả React app lẫn GeoIP database.

   *(Tương đương thủ công nếu không dùng make):*
   ```bash
   cd frontend && pnpm install --frozen-lockfile && pnpm build && cd ..
   rm -rf internal/web/dist && cp -r frontend/dist internal/web/dist
   go build -o quanly-phongtro-api ./cmd/api
   ```
3. Cấu hình file `.env` trên server production với các biến bảo mật (DB Credentials mật khẩu mạnh, `JWT_SECRET` sinh ngẫu nhiên mạnh, `APP_PORT`,...).

### 2.3 Chạy bằng Systemd
Tạo file `/etc/systemd/system/quanly-phongtro-api.service`:
```ini
[Unit]
Description=QuanLyPhongTro API Backend
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=deploy
Group=deploy
WorkingDirectory=/path/to/project
EnvironmentFile=/path/to/project/.env
ExecStart=/path/to/project/quanly-phongtro-api
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```
Kích hoạt và chạy service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable quanly-phongtro-api
sudo systemctl start quanly-phongtro-api
```

> **Cổng & TLS:** Binary nghe trực tiếp ở `APP_PORT`. Vì không còn Nginx, hãy cho phép truy cập tới cổng đó.
> - Nếu chạy cổng <1024 (vd 80/443), thêm `AmbientCapabilities=CAP_NET_BIND_SERVICE` vào block `[Service]`.
> - Nếu cần HTTPS, đặt một reverse proxy/load balancer terminate TLS ở tầng mạng phía trước, hoặc cấu hình TLS trực tiếp cho server Go.

### 2.4 Cập nhật dự án (Update)
Khi có phiên bản code mới:
1. `git pull` code mới.
2. Chạy `migrate up` nếu có thay đổi DB Schema.
3. Build lại binary (`make build`) — frontend được build và nhúng lại trong cùng bước.
4. Restart service: `sudo systemctl restart quanly-phongtro-api`.

> Lưu ý: pipeline CI/CD (`.github/workflows/deploy.yml`) đã tự động hoá toàn bộ các bước trên cho nhánh `develop`.

---

> **Lưu ý mở rộng (Docker):**
> Dự án cũng định hướng hỗ trợ Docker/Docker Compose để đóng gói (Containerization). Vì frontend đã nhúng trong binary, image chỉ cần build một binary duy nhất; quy trình triển khai Production khi đó đơn giản là `docker-compose up -d --build`.
