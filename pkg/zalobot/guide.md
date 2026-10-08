# Zalo Bot module — hướng dẫn tích hợp

Module độc lập để làm việc với [Zalo Bot API](https://bot.zapps.me/docs/): gọi API bot (`getMe`, `sendMessage`, `sendPhoto`, `setWebhook`), xác thực và parse webhook, tải ảnh người dùng gửi cho bot an toàn (chống SSRF).

- Go: chỉ dùng **standard library**, không phụ thuộc DB/ORM/framework.
- Không giữ state: lưu bot token, webhook secret, liên kết chat ↔ đối tượng trong app là việc của app (xem `sql/schema.sql`).
- Chỉ có backend. Nội dung tin nhắn, lệnh chat, luồng liên kết tài khoản và giao diện cấu hình thuộc về app.
- Đang chạy thật trong `quanly-phongtro`; phần nối với app nằm trong `internal/service/zalo/zalobot_adapter.go`.

## 1. Cấu trúc thư mục

```
zalobot/
├── guide.md         # tài liệu này
├── client.go        # Client: GetMe, SendMessage, SendPhoto, SetWebhook; BotInfo
├── errors.go        # APIError, IsAuthError, ErrMissingSecretToken, ErrInvalidSecretToken, ErrInvalidImageURL, ErrInvalidImage
├── webhook.go       # ParseUpdate, Update, VerifySecretToken, SecretTokenHeader, MaxWebhookBodyBytes, Event*
├── image.go         # ImageDownloader / DownloadImage: chỉ HTTPS, chỉ IP public, giới hạn kích thước và content-type
├── *_test.go        # go test ./... (không cần mạng)
└── sql/schema.sql   # bảng cấu hình bot / liên kết chat mẫu (PostgreSQL)
```

## 2. Copy sang dự án khác

```bash
cp -r pkg/zalobot <du-an-moi>/pkg/zalobot
cd <du-an-moi> && go test ./pkg/zalobot/...
```

Import theo module path của dự án mới, ví dụ `github.com/acme/shop/pkg/zalobot`. Package không import gì ngoài stdlib.

## 3. Chuẩn bị trên Zalo

1. Tạo bot tại https://bot.zapps.me, lấy **Bot Token** (dạng `<bot_id>:<secret>`).
2. Không cần cấu hình webhook bằng tay: app gọi `SetWebhook` với URL + secret do app sinh.
3. Muốn bot hoạt động trong nhóm: thêm bot vào nhóm Zalo. Bot chỉ nhận tin nhóm khi được **@mention** hoặc **reply**.

Giới hạn của Zalo Bot API cần thiết kế quanh:

- Chỉ có 5 loại event tin nhắn (text / image / sticker / voice / unsupported). **Không có** event "bot được thêm vào nhóm".
- `chat` chỉ có `{id, chat_type}`: không có tên nhóm và không có API lấy thông tin nhóm. Muốn gắn nhóm với một đối tượng (phòng, đơn hàng...) thì cho bot trả lời Group ID để người quản lý dán vào app, hoặc cho người quản lý gõ lệnh liên kết ngay trong nhóm.
- Danh thiếp (contact card) đến dưới dạng `message.unsupported.received`.

## 4. Lưu cấu hình bot

```go
client := zalobot.NewClient(nil) // nil = http.Client timeout 10s

if _, err := client.GetMe(ctx, botToken); err != nil {
    return fmt.Errorf("invalid bot token: %w", err)
}
secret := randomSecret() // tự sinh, ví dụ 12–32 ký tự [A-Za-z0-9-]
webhookURL := "https://app.example.com/api/v1/zalo/webhooks/" + ownerID
if err := client.SetWebhook(ctx, botToken, webhookURL, secret); err != nil {
    return err
}
// mã hoá botToken + secret (AES-256-GCM) rồi lưu, đánh dấu is_active = true
```

Khuyến nghị:

- Mỗi chủ (manager/shop) một bot riêng thì URL webhook nên chứa id chủ để biết lấy secret nào.
- Mã hoá token và secret bằng khoá riêng, không trả plaintext về frontend.
- Lỗi từ `Client` không chứa URL request (URL có bot token), có thể log thẳng.
- Kiểm tra token định kỳ (cron gọi `GetMe`); `zalobot.IsAuthError(err)` → đánh dấu bot inactive.

## 5. Webhook

```go
func (h *Handler) ZaloWebhook(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, zalobot.MaxWebhookBodyBytes))
    if err != nil { http.Error(w, "invalid body", http.StatusBadRequest); return }

    update, err := zalobot.ParseUpdate(body)
    if err != nil { w.WriteHeader(http.StatusOK); return }

    ownerID := chi.URLParam(r, "ownerID")
    secret, err := h.store.WebhookSecret(r.Context(), ownerID)
    if err == nil {
        err = zalobot.VerifySecretToken(r.Header.Get(zalobot.SecretTokenHeader), secret)
    }
    if err != nil { w.WriteHeader(http.StatusOK); return } // ErrMissingSecretToken / ErrInvalidSecretToken

    h.bot.Handle(r.Context(), ownerID, update) // logic riêng của app
    w.WriteHeader(http.StatusOK)
}
```

- Trả 200 kể cả khi lỗi để Zalo không gửi lại liên tục; tự log lỗi.
- `VerifySecretToken` so sánh constant-time và từ chối khi secret đã lưu rỗng. Nếu app cần chấp nhận cấu hình cũ chưa có secret thì tự xử lý trước khi gọi (quanly-phongtro làm vậy trong adapter).

`Update` đã chuẩn hoá cả payload mới (`{"ok":true,"result":{...}}`) lẫn payload cũ:

| Field | Ý nghĩa |
|---|---|
| `EventName` | ví dụ `message.text.received`, `message.unsupported.received` (`EventTextReceived`, `EventUnsupportedReceived`) |
| `SenderID` | Zalo user id người gửi |
| `ChatID` | id nhóm; rỗng khi chat riêng |
| `ReplyChatID` | chat cần trả lời: id nhóm khi ở nhóm, id người gửi khi chat riêng |
| `IsGroup` | tin đến từ nhóm |
| `GroupName` | chỉ có ở payload cũ / giả lập; Zalo Bot API hiện không gửi |
| `Text` | nội dung đã trim |
| `PhotoURL` | ảnh trong `message.photo` hoặc attachment `image` |
| `ContactPhone` | số điện thoại trong attachment khác image (nếu có) |

## 6. Gửi tin

```go
err := client.SendMessage(ctx, botToken, update.ReplyChatID, "Xin chào!")
err = client.SendPhoto(ctx, botToken, chatID, "https://app.example.com/files/invoice.png", "Hoá đơn tháng 10")
```

- `SendPhoto` nhận URL ảnh public (HTTPS) mà Zalo tải được; app tự host ảnh (nên dùng URL ký có hạn).
- Lỗi API trả về `*zalobot.APIError` (`Method`, `StatusCode`, `Code`, `Description`, `Body`).

## 7. Tải ảnh người dùng gửi

```go
data, err := zalobot.DownloadImage(ctx, update.PhotoURL) // mặc định: tối đa 5MB, timeout 10s
```

`ImageDownloader` chỉ chấp nhận HTTPS, chặn host phân giải ra IP nội bộ/loopback/link-local (kiểm tra cả lúc dial để chống DNS rebinding), redirect tối đa 3 lần và cùng host, content-type `image/jpeg|png|gif|webp`. Tuỳ chỉnh bằng struct:

```go
d := &zalobot.ImageDownloader{MaxBytes: 2 << 20, Timeout: 5 * time.Second}
data, err := d.Download(ctx, url)
```

Trong test có thể gán `Transport` và `LookupIP` để không gọi mạng.

## 8. Kiểm thử

- Unit: `go test ./pkg/zalobot/...` (giả lập HTTP bằng `RoundTripper`, không gọi mạng).
- Giả lập webhook: POST body như `{"ok":true,"result":{"event_name":"message.text.received","message":{"from":{"id":"u1"},"chat":{"id":"u1","chat_type":"PRIVATE"},"text":"bot ơi"}}}` kèm header `X-Bot-Api-Secret-Token`.

## 9. Lỗi hay gặp

| Triệu chứng | Nguyên nhân / cách xử lý |
|---|---|
| `getMe` lỗi `-216` / 401 | Token sai hoặc đã bị thu hồi; `IsAuthError` trả `true`. |
| Webhook không đến | Chưa gọi `SetWebhook` thành công, URL không public HTTPS, hoặc trong nhóm bot không được @mention/reply. |
| `ErrInvalidSecretToken` | Secret đã lưu khác secret truyền vào `SetWebhook` (lưu lại cấu hình để gọi `SetWebhook` lần nữa). |
| Không biết tin đến từ nhóm nào | Zalo không gửi tên nhóm; liên kết nhóm thủ công bằng `ChatID`. |
| `ErrInvalidImageURL` khi tải ảnh | URL không phải HTTPS hoặc host trỏ về IP nội bộ. |
