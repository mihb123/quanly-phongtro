# Quản Lý Zalo Bot (Zalo Bot Integration)

Tính năng tích hợp Zalo Bot để tự động hóa liên lạc với người thuê (tenant). Hỗ trợ cấu hình bot cá nhân cho từng quản lý (manager), tự động liên kết phòng với group Zalo thông qua webhook, và gửi thông báo/hóa đơn tự động vào group chat.

## 1. Màn hình và cấu hình

Route màn hình cấu hình: Nằm trong phần Settings (`/settings`).

Quyền và luồng cấu hình:
- Tích hợp tại mục cài đặt cá nhân của Manager. Mỗi manager nhập `Zalo Bot Token` và `Zalo Webhook Secret` lấy từ Zalo Bot Platform.
- Frontend sẽ mã hóa Token/Secret bằng Public Key do Server cung cấp (RSA) trước khi gửi đi.
- Backend tiếp tục mã hóa dữ liệu (AES-256-GCM) trước khi lưu vào cơ sở dữ liệu để đảm bảo bảo mật tối đa.
- Màn hình sẽ hiển thị Webhook URL tự động sinh theo ID của manager, dùng để cấu hình trong Zalo Platform.
- Việc gửi hóa đơn qua Zalo được thực hiện trực tiếp từ màn hình chi tiết hóa đơn (InvoiceDetailModal).

## 2. Frontend

File chính nằm trong `frontend/src/components/home/`:
- `SettingsView.tsx`: Màn hình chứa form cài đặt Zalo Bot.
- `InvoiceView.tsx` / `modals/InvoiceDetailModal.tsx`: Giao diện chi tiết hóa đơn tích hợp tính năng gửi hóa đơn qua Zalo.
- `modals/EditRoomModal.tsx`: Hỗ trợ trường thông tin `group_chat_id` của phòng để sửa thủ công hoặc kiểm tra tính năng auto-linking.

API layer nằm trong `frontend/src/api/zalo.tsx`. Chứa các hàm giao tiếp API:
- Lấy Public Key từ backend (`/api/v1/zalo/public-key`)
- Lấy trạng thái cài đặt (`/api/v1/zalo/config`)
- Lưu cấu hình bot (`/api/v1/zalo/config`)

## 3. Backend

Handlers/Controllers (`internal/handler/`):
- `zalo_handler.go`: Xử lý HTTP request liên quan đến cấu hình Zalo, nhận Webhook, test tin nhắn và gửi hóa đơn.
- Các route được đăng ký tại `internal/router/router.go`.

API chính:
- `GET /api/v1/zalo/public-key`: Cấp phát RSA Public Key cho Frontend mã hóa dữ liệu.
- `GET /api/v1/zalo/config`: Trả về trạng thái cấu hình (có hay chưa) và URL Webhook. Yêu cầu Auth.
- `POST /api/v1/zalo/config`: Lưu thiết lập bot. Yêu cầu Auth.
- `POST /api/v1/zalo/webhook/{managerID}`: Endpoint public nhận sự kiện từ Zalo Platform (ví dụ sự kiện bot được thêm vào nhóm).
- `POST /api/v1/zalo/send-message`: Test gửi tin nhắn văn bản thông thường.
- `POST /api/v1/zalo/invoices/{id}/send`: Gửi hóa đơn (text và file ảnh) vào group Zalo của phòng tương ứng.

Service layer (`internal/service/`):
- `zalo_client.go`: Custom HTTP Client sử dụng `net/http` giao tiếp trực tiếp với Zalo API, không dùng SDK ngoài.
- `zalo_service.go`: Xử lý business logic, giải mã token, xác thực webhook qua header `X-Bot-Api-Secret-Token`, auto-linking phòng.
- `zalo_service_image.go`: Xử lý render HTML template thành ảnh (thông qua `wkhtmltoimage`), upload lên Zalo và đính kèm vào tin nhắn hóa đơn.

## 4. Data Model

Nguồn dữ liệu có sẵn:
- `users`: Thông tin người quản lý.
- `rooms`: Quản lý các phòng và cấu hình tương ứng.

Entity của feature:
- Bản ghi `users`: Bổ sung thêm các cột `zalo_bot_token` (TEXT) và `zalo_webhook_secret` (TEXT). Cả hai trường đều được lưu trữ dạng đã mã hóa.
- Bản ghi `rooms`: Bổ sung thêm `group_chat_id` (VARCHAR) lưu Zalo Chat ID của nhóm tương ứng.

Migration liên quan:
- `migrations/000012_add_zalo_fields.up.sql`
- `migrations/000012_add_zalo_fields.down.sql`

## 5. Rule nghiệp vụ cần giữ

### Bảo mật Token và Thông tin
- Frontend sử dụng RSA Public Key để mã hóa `Bot Token` và `Webhook Secret` trước khi gọi POST. Backend sử dụng Private Key giải mã, rồi lại dùng AES-256-GCM (với `ZALO_BOT_ENCRYPTION_KEY` trong `.env`) mã hóa trước khi lưu Database. 
- Get Config API chỉ trả về `has_config` (true/false) và `webhook_url`, không trả về plain-text Token/Secret.

### Webhook & Tự động liên kết (Auto-linking)
- Zalo gửi event `group.bot.add` khi bot vào nhóm. Tên nhóm bắt buộc tuân theo định dạng: `<Tên Phòng> <Tên Nhà>` (VD: `P101 679QT`).
- Webhook Payload sẽ được verify bằng cách so sánh header `X-Bot-Api-Secret-Token` với Webhook Secret (đã giải mã) lưu trong DB.
- Backend parse `group_name` từ payload, đối chiếu với danh sách phòng của `managerID`, tự động điền `group_chat_id` cho phòng hợp lệ.

### Gửi hóa đơn
- Không sử dụng SDK bên thứ 3 (chẳng hạn `go-zalo-bot`) do yêu cầu custom và bảo mật riêng.
- Khi người dùng gửi hóa đơn, hệ thống sử dụng `wkhtmltoimage` để tạo ảnh hóa đơn. Tấm ảnh này được upload qua Zalo Client để lấy file token trước khi gửi dưới dạng tin nhắn đính kèm.

## 6. Khi maintain

- Không chia sẻ key `ZALO_BOT_ENCRYPTION_KEY` cũng như RSA Private Key. Việc rotate key sẽ ảnh hưởng đến mọi token hiện hành.
- Nếu thêm/sửa Endpoint Zalo, khai báo thủ công trong `zalo_client.go` thay vì chèn thư viện SDK.
- Payload Webhook từ nền tảng Zalo Bot có thể thay đổi, lúc này cần điều chỉnh struct dùng trong `zalo_service.go` tương ứng với tài liệu API mới của Zalo.
- Backend yêu cầu server cài đặt sẵn `wkhtmltopdf` (và `wkhtmltoimage`) cho tính năng tạo ảnh hóa đơn. Nếu tính năng tạo và gửi ảnh bị lỗi do không tìm thấy lệnh, cần kiểm tra lại cài đặt trên hệ điều hành (ví dụ: `apt-get install wkhtmltopdf`).

Official Documentation Zalo-bot:

### 1. Bắt đầu (Getting Started)
- https://bot.zapps.me/docs/ (Giới thiệu)
- https://bot.zapps.me/docs/create-bot/ (Tạo Bot)
- https://bot.zapps.me/docs/authorize/ (Xác thực)
- https://bot.zapps.me/docs/call-api/ (Sử dụng API)

### 2. API Reference (Danh sách API)
- https://bot.zapps.me/docs/apis/getMe/ (API getMe)
- https://bot.zapps.me/docs/apis/getUpdates/ (API getUpdates)
- https://bot.zapps.me/docs/apis/sendMessage/ (API sendMessage)
- https://bot.zapps.me/docs/apis/sendPhoto/ (API sendPhoto)
- https://bot.zapps.me/docs/apis/sendSticker/ (API sendSticker)
- https://bot.zapps.me/docs/apis/sendChatAction/ (API sendChatAction)

### 3. Tích hợp Webhook (Webhook Integration)
- https://bot.zapps.me/docs/webhook/ (Webhook)
- https://bot.zapps.me/docs/apis/setWebhook/ (API setWebhook)
- https://bot.zapps.me/docs/apis/deleteWebhook/ (API deleteWebhook)
- https://bot.zapps.me/docs/apis/getWebhookInfo/ (API getWebhookInfo)
- https://bot.zapps.me/docs/build-your-bot-with-webhook/ (Hướng dẫn tạo bot với Webhook)

### 4. Tham khảo thêm (References & Tutorials)
- https://bot.zapps.me/docs/build-personal-assistant-with-open-claw/ (Hướng dẫn xây dựng trợ lý ảo với Open Claw - Best Practices)
- https://bot.zapps.me/docs/build-your-bot/ (Hướng dẫn từng bước xây dựng Bot)
- https://bot.zapps.me/docs/build-bot-interaction-with-group/ (Tương tác với Group Chat - sẽ được ra mắt trong thời gian tới)
- https://bot.zapps.me/docs/error-code/ (Bảng mã lỗi)
- https://bot.zapps.me/docs/terms/ (Điều khoản sử dụng)
- https://hecigo.com/blog/toi-uu-tu-dong-hoa-zalo-bot-voi-n8n-huong-dan-chi-tiet-tu-hecigo (Tối ưu tự động hóa Zalo Bot với n8n)