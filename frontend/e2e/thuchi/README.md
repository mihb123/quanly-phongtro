# E2E tiện ích Thu chi

Puppeteer + `node:test`, kiểm DB bằng `psql`. Mỗi bước đi đủ 3 lớp: thao tác UI → bản ghi Postgres → hiển thị (kể cả IndexedDB). Bộ test copy từ `~/thu-chi/frontend/e2e`, phần khác là cách vào app: tạo chủ trọ trực tiếp trong DB (đã kích hoạt), đăng nhập qlpt, bật tiện ích ở tab **Tiện ích**, mọi selector `nav/header/main` được giới hạn trong `[data-thuchi]` (gốc của `ThuChiApp`).

## Chạy

```bash
make e2e-thuchi            # build + DB qlpt_thuchi_e2e + server :9192 + Bot API giả :9193 + toàn bộ file
make e2e-thuchi T=07       # chỉ file có tên chứa "07"
cd frontend && E2E_SKIP_BUILD=1 node e2e/thuchi/run.mjs 01 12
```

Cần Postgres 18 dev ở `127.0.0.1:5440` (cluster của `~/thu-chi`, user `thuchi` trust) và Chrome của Puppeteer trong `~/.cache/puppeteer`. Runner **không đọc `.env`**: server chạy trong thư mục tạm với secret ngẫu nhiên, SMTP giả, `POSTGRES_DSN` trỏ DB test.

| Biến | Mặc định | Ý nghĩa |
|---|---|---|
| `E2E_SKIP_BUILD` | — | Bỏ `make build` nếu binary đã có |
| `E2E_CONCURRENCY` | `3` | Số file chạy song song (mỗi file một chủ trọ riêng) |
| `E2E_HEADFUL`, `E2E_SLOWMO` | — | Mở cửa sổ Chrome, làm chậm thao tác (ms) |
| `E2E_PORT` | `9192` | Cổng server; Bot API giả ở `E2E_PORT + 1` |
| `E2E_PG_ADMIN_URL` | `postgres://thuchi@127.0.0.1:5440/postgres` | Cluster để tạo DB test |
| `E2E_DB_NAME` | `qlpt_thuchi_e2e` | DB bị **xoá và tạo lại** mỗi lần chạy (tên bắt buộc chứa `e2e`) |
| `E2E_TMP` | `$TMPDIR/qlpt-thuchi-e2e` | Binary, log server, ảnh chụp lỗi, file tải về |

Bot Telegram chạy với token giả và `THUCHI_TELEGRAM_API_BASE` trỏ vào `lib/telegram-fake.mjs` (getUpdates/sendMessage/editMessageText…; test đẩy update qua `POST /_e2e/update`, đọc tin bot gửi qua `GET /_e2e/sent?chat=`), nên không đụng bot thật.

## Phạm vi

| File | Tính năng |
|---|---|
| `01-plugin` | Tắt mặc định (API 403, ingest công khai 404), bật trong Tiện ích → seed 20 danh mục + Tiền mặt, giao diện nhúng không đè thanh điều hướng qlpt, tắt/bật lại giữ dữ liệu, tải lại trang, đăng xuất qlpt xoá IndexedDB, đăng nhập lại không seed trùng |
| `02`–`09` | Như bộ của thu-chi: giao dịch tay, tài khoản/chuyển tiền, danh mục/luật/hạn mức, vay nợ, định kỳ, SePay, thông báo/email/sao kê, thống kê |
| `10-sync-offline` | Outbox khi mất mạng (không tải lại trang: qlpt chưa có service worker), 2 máy LWW theo trường, mutation bị từ chối, reset |
| `11-security-validation` | Xác thực API/ingest, DPoP, tách dữ liệu giữa chủ trọ, validator + savepoint, ép giờ client, route công khai của plugin |
| `12-telegram` | Kết nối bằng mã 1 lần, ghi "cafe sáng 15k", trả lời "20k" để sửa, tắt tiện ích thì bot báo đang tắt và không ghi, người lạ được nhắc kết nối |

Bước `todo` là lỗi app đang mở (không làm đỏ suite). Sửa `pkg/thuchi` ở `~/thu-chi` trước rồi `make sync-thuchi`, thêm bước test ở cả hai repo.
