# Zalo Bot Service — Tích hợp Zalo Bot (Go Monolith & Custom Client)

## Bối cảnh & Các quyết định thiết kế

Dự án **Quản lý phòng trọ** cần tích hợp Zalo Bot để tự động hóa liên lạc với người thuê (tenant). Dựa trên yêu cầu:

1. **Kiến trúc Native (Không dùng SDK ngoài)**: Tích hợp trực tiếp vào Go backend hiện tại. **KHÔNG** sử dụng SDK bên thứ 3 (`go-zalo-bot`) vì lý do bảo mật. Chúng ta sẽ tự viết một HTTP Client nhỏ (Custom Client) giao tiếp trực tiếp với [Zalo Bot API](https://bot.zaloplatforms.com/docs/) qua `net/http`.
2. **Mô hình Manager**: Mỗi quản lý (manager) sẽ có **Bot riêng**. Manager nhập `Bot Token` và `Webhook Secret` trên Frontend. Thông tin này được **mã hóa (AES-256-GCM)** và lưu vào DB.
3. **Mô hình Chat**: Bot sẽ được manager add vào **Group chat** (gồm Manager + Tenant + Bot).
4. **Phase 1 Scope**: 
   - Hỗ trợ gửi tin nhắn (hóa đơn) từ hệ thống vào Group Zalo.
   - Hỗ trợ **Webhook tối thiểu** để tự động bắt event `group.bot.add` (khi bot được add vào group) nhằm lấy `group_chat_id`.
5. **Auto-linking (Tự động liên kết)**: Đặt tên group theo cú pháp `<room_name> <house_name>` (VD: `P101 679QT`). Khi webhook nhận event, backend tự động parse tên group và gán `group_chat_id` vào đúng phòng trong DB.

---

## Cấu hình Bảo mật (Đã hoàn thành)

- Đã tạo key AES-256 (32 bytes hex) và đưa vào `.env` dưới tên `ZALO_BOT_ENCRYPTION_KEY`.
- Cả `Bot Token` và `Webhook Secret` của user sẽ được mã hóa bằng key này.

---

## Proposed Changes

### Component 1: Database Migration

#### [NEW] migrations/000012_add_zalo_fields.up.sql

```sql
-- Thêm thông tin Zalo Bot (encrypted) cho users (manager)
ALTER TABLE users ADD COLUMN zalo_bot_token TEXT NULL;
ALTER TABLE users ADD COLUMN zalo_webhook_secret TEXT NULL;

-- Thêm cột group_chat_id cho rooms
ALTER TABLE rooms ADD COLUMN group_chat_id VARCHAR(255) NULL;
```

#### [NEW] migrations/000012_add_zalo_fields.down.sql
```sql
ALTER TABLE users DROP COLUMN IF EXISTS zalo_bot_token;
ALTER TABLE users DROP COLUMN IF EXISTS zalo_webhook_secret;
ALTER TABLE rooms DROP COLUMN IF EXISTS group_chat_id;
```

---

### Component 2: Go Backend — Model & Repository

#### [MODIFY] `internal/model/user.go`
- Thêm fields `ZaloBotToken` và `ZaloWebhookSecret` vào struct `User`.
- Cập nhật `UserRepository` để có các hàm update/get các token này.

#### [MODIFY] `internal/model/room.go`
- Thêm field `GroupChatID` vào `Room` và `UpdateRoomParams`.
- Cập nhật các query (SELECT, UPDATE) trong `room_repository.go`.

---

### Component 3: Go Backend — Crypto Layer

#### [NEW] `internal/security/crypto.go`
- `Encrypt(plaintext string, key []byte) (string, error)`: Mã hóa bằng AES-256-GCM, trả về base64.
- `Decrypt(ciphertext string, key []byte) (string, error)`.

---

### Component 4: Go Backend — Zalo Custom Client & Service

#### [NEW] `internal/service/zalo_client.go`
Thay vì dùng SDK, tạo Zalo Client gọn nhẹ bằng `net/http`:

```go
// Tương tác trực tiếp API: https://bot.zaloplatforms.com/docs/apis/sendMessage/
type ZaloClient interface {
    GetMe(ctx context.Context, botToken string) (*ZaloAppInfo, error)
    SendMessage(ctx context.Context, botToken, chatID, text string) error
}
```
- `SendMessage`: Gửi POST request tới `https://bot-api.zaloplatforms.com/bot${botToken}/sendMessage` với body json `{"chat_id": chatID, "text": text}`.

#### [NEW] `internal/service/zalo_service.go`
Trách nhiệm:
1. **Quản lý Credentials**: Lưu và giải mã Token + Secret (kết hợp `crypto.go`).
2. **Gửi tin nhắn**: Hàm `SendTextMessage(ctx, managerID, chatID, text)`:
   - Lấy & giải mã `Bot Token` của manager.
   - Gọi `ZaloClient.SendMessage`.
3. **Webhook Handler**: Hàm `HandleWebhook(ctx, managerID, body []byte, secretTokenHeader string)`:
   - Lấy & giải mã `Webhook Secret` của manager.
   - **Xác thực**: Zalo Bot API gửi header `X-Bot-Api-Secret-Token` hoặc `X-Zalo-Signature`. Ta kiểm tra xem header có khớp với `Webhook Secret` đã lưu hay không. (Theo doc mới nhất, webhook từ Bot Platform sử dụng Header `X-Bot-Api-Secret-Token` so khớp trực tiếp).
   - Parse event: Bắt event `group.bot.add` hoặc tin nhắn từ group.
   - Đọc `group_name`. Lấy list Rooms của Manager, tìm match `<RoomName> <HouseName>` và update `group_chat_id` cho DB.

---

### Component 5: Go Backend — API Handler

#### [NEW] `internal/handler/zalo_handler.go`

Các Endpoints:
- `POST /api/v1/zalo/config` (Auth: MANAGER): Nhận `{ bot_token, webhook_secret }` từ FE. Gọi `ZaloClient.GetMe` để test thử Token hợp lệ không, sau đó mã hóa & lưu.
- `GET /api/v1/zalo/config` (Auth: MANAGER): Trả về `{ has_config: true, webhook_url: "https://.../api/v1/zalo/webhook/<manager_id>" }`.
- `POST /api/v1/zalo/webhook/{managerID}` (Public): Nhận Webhook từ Zalo. Endpoint này đọc header `X-Bot-Api-Secret-Token` và body truyền cho `zalo_service`.
- `POST /api/v1/zalo/send-message` (Auth: MANAGER): Endpoint hỗ trợ test gửi tin hoặc được gọi bởi module Invoice.

#### [MODIFY] `internal/router/router.go` & `cmd/api/main.go`
- Đăng ký `zalo_handler` vào `chi.Router`.

---

### Component 6: Frontend — UI & API Clients

#### [NEW] `frontend/src/api/zalo.tsx`
- Các API definitions: `saveZaloConfig`, `getZaloConfigStatus`.

#### [MODIFY] Zalo Settings UI (Tích hợp vào Settings/Profile của Manager)
- Form nhập `Bot Token` và `Webhook Secret`.
- Hướng dẫn cấu hình Zalo: 
  1. Tạo Bot tại `bot.zaloplatforms.com`
  2. Dán Token và Webhook Secret (tự định nghĩa) vào form.
  3. Copy URL do hệ thống cấp (`https://.../api/v1/zalo/webhook/XYZ`) dán vào cấu hình Webhook của Zalo.
  4. Tạo Group chat Zalo, đặt tên `<Tên Phòng> <Tên Nhà>` và add Bot vào.

#### [MODIFY] `frontend/src/api/room.tsx` & Các components Room
- Thêm field `group_chat_id` (Text Input) vào chi tiết phòng.
- Mặc dù hệ thống tự động liên kết (Auto-linking), vẫn cần form này để hiển thị hoặc cho phép Manager sửa thủ công nếu Auto-linking có vấn đề (VD: đặt tên group sai).

---

## Lộ trình triển khai

**Phase 1: Backend Core (Ước tính: 4-5 tiếng)**
- Code Migration & Update Model, Repository.
- Code `crypto.go` (Mã hóa).
- Code `zalo_client.go` (HTTP client tương tác Zalo).
- Code `zalo_service.go` (Business logic, Auto-linking webhook).
- Code API Handlers & Router wiring.

**Phase 2: Frontend & Testing (Ước tính: 2-3 tiếng)**
- Xây dựng Zalo Settings UI trên trang quản lý.
- Hiển thị & cấu hình Webhook URL.
- Thêm thuộc tính `group_chat_id` vào Module Phòng (Room).
- Postman testing và giả lập Zalo Webhook payload để verify luồng Auto-linking.
