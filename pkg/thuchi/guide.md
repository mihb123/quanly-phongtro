# pkg/thuchi — hướng dẫn module

Lõi thu chi cá nhân local-first: đồng bộ client ↔ server, nhận giao dịch ngân hàng (SePay, thông báo Android, email), gộp trùng, ghép chuyển khoản nội bộ, hoàn tiền, phân loại tự học. Go chỉ dùng standard library (+ `pkg/sepay` cũng stdlib); web chỉ import `react`, `dexie`, `dexie-react-hooks`, `recharts`, `sonner` và `@/...`.

## Cấu trúc

```
pkg/thuchi/
├── model.go       # Account, Category, Transaction, Rule, Budget, RawEvent, IngestToken, Incoming
├── store.go       # interface Store / Tx (mọi ghi chạy trong RunOwner: advisory lock + seq theo owner)
├── pgstore.go     # Store trên database/sql (PostgreSQL ≥ 18)
├── sync.go        # Pull / Push, whitelist trường client được ghi, LWW theo trường (field_ts)
├── actions.go     # mutation nguyên tử: set_category(+learn), confirm/reject/mark/unmark_transfer, reparse/dismiss_raw
├── pipeline.go    # raw event → parse → dedupe/gộp → loại (hoàn tiền/trả thẻ/nội bộ) → phân loại → ghép nội bộ (+ tách phí)
├── sources.go     # ParseSePay, DecodeNotify/ParseNotify (registry NotifyParsers), ParseEmail/ParseEmailEvents (registry EmailParsers, file đính kèm → sao kê)
├── bank_vcb.go    # parser email biên lai Vietcombank (đọc ô <td> nhãn → giá trị)
├── statement.go   # sao kê nhiều giao dịch (registry StatementParsers): Timo .xls
├── xls.go         # đọc .xls (OLE2 + BIFF8) thuần Go
├── defaults.go    # danh mục mặc định, từ điển merchant, từ khoá hoàn tiền/trả thẻ, mã ngân hàng
├── text.go        # bỏ dấu, chuẩn hoá, hash số TK, UUIDv7, tách tên cửa hàng từ nội dung
├── quick.go       # đọc tin nhắn tự do ("cafe 15k", "+2tr lương", "hôm qua grab 32k") + phân loại theo luật học/từ điển
├── telegram.go    # bot Telegram (long polling getUpdates): ghi giao dịch, nút đổi danh mục, sửa/xoá, /tong
├── handler.go     # HTTP (net/http ServeMux): API đã đăng nhập + endpoint ingest công khai
├── service.go     # Service + worker xử lý raw event
├── sql/schema.sql # bảng thuchi_* (giống migrations/000002)
└── web/           # data layer (Dexie + outbox + SyncEngine) và UI
```

App nối module qua đúng 1 file `internal/service/thuchi/thuchi_adapter.go` (owner từ JWT claims, mã hoá secret bằng AES-GCM của app, worker + dọn tombstone).

## API (sau auth, prefix `/api/v1/plugins/thu-chi`)

| Method | Path | Ghi chú |
|---|---|---|
| GET | `/sync/pull?since=<seq>&limit=` | Trả `{seq, has_more, reset?, changes{accounts,categories,transactions,rules,budgets,raw_events}}`. `reset=true` khi `since` cũ hơn mốc đã dọn tombstone → client xoá local, tải lại từ 0 |
| POST | `/sync/push` | `{mutations:[{mid, entity, op: upsert\|patch\|delete\|action, id, changes, ts, action, args}]}` → `{results:[{mid, ok, error}], seq}` |
| GET | `/ingest/info` | URL ingest theo nguồn, token (không có secret), cấu hình SePay |
| POST | `/ingest/tokens` | `{kind: notify\|email, name}` → secret `"<id>.<secret>"` hiện 1 lần |
| DELETE | `/ingest/tokens/{id}` | Thu hồi |
| PUT | `/ingest/sepay` | `{auth_method: hmac\|apikey, secret}` |
| GET | `/telegram/info` | `{enabled, bot_username, links[]}` |
| POST | `/telegram/link` | Tạo mã kết nối 8 ký tự, 15 phút, dùng 1 lần → `{code, command: "/link CODE", url: t.me/<bot>?start=CODE}` |
| PUT | `/telegram/links/{id}` | `{default_account_id}` (null = tài khoản Tiền mặt) |
| DELETE | `/telegram/links/{id}` | Ngắt kết nối |

Quy tắc ghi của client (`sync.go: writable`): chỉ các trường người dùng (`category_id, note, lat, lng, place_name, excluded, kind, pair_*, needs_review`…). Trường ngân hàng (`amount, direction, occurred_at, raw_content, counterparty_*, balance_after, sources`) chỉ sửa được với giao dịch nhập tay. `account_no` gửi lên được băm (`account_no_hash`), không lưu số thật. `ts` lớn hơn giờ server quá 1 phút bị ép về giờ server.

## Ingest công khai

`POST /api/v1/plugins/thu-chi/public/{ownerID}/ingest/{sepay|notify|email}` → xác thực → ghi `thuchi_raw_events` (pending) → 200 → worker xử lý.

| Nguồn | Xác thực | Body |
|---|---|---|
| sepay | Header của SePay: `X-SePay-Timestamp` + `X-SePay-Signature` (HMAC) hoặc `Authorization: Apikey …` | JSON webhook SePay (`transferType` in/out, `accumulated` = số dư) |
| notify | `Authorization: Bearer <id>.<secret>` hoặc `X-ThuChi-Token: <id>` + `X-ThuChi-Timestamp` + `X-ThuChi-Signature: sha256=hex(HMAC(secret, ts + "." + body))` (±5 phút) | JSON `{package, title, text, posted_at, lat?, lng?}` (nhận cả key kiểu SmsForwarder: `from`, `content`, `receive_time`) |
| email | như notify (Worker dùng HMAC) | MIME thô (`message/rfc822`) |

Dedupe: `UNIQUE(owner, source, source_ref)`; SePay `source_ref = <số TK>:<referenceCode>`. Khác nguồn: cùng tài khoản + chiều + số tiền, lệch ≤ 10 phút, số dư sau GD (nếu có) phải khớp → gộp vào bản ghi cũ, `reconciled = true`, nguồn ưu tiên SePay > notify/email.

## Bot Telegram

Bật khi có `TELEGRAM_BOT_TOKEN`; `TELEGRAM_CHAT_ID` (nhiều id cách dấu phẩy) giới hạn nhóm, chat riêng luôn được. Bot cần tắt privacy mode (BotFather `/setprivacy` → Disable) để đọc tin thường trong nhóm, và **token phải dành riêng cho app này**: chỉ một tiến trình được `getUpdates`.

- Kết nối: Cài đặt → Telegram → mã 1 lần; người dùng bấm link `t.me/<bot>?start=CODE` hoặc gửi `/link CODE` trong nhóm. `thuchi_telegram_links` map `telegram_user_id` → owner (mỗi Telegram 1 owner, 1 owner nhiều Telegram).
- Tin nhắn → `ParseQuick` (mỗi dòng / mỗi đoạn `, ` một khoản; `15k`, `1tr2`, `1,5tr`, `15.000đ`, `50 nghìn`, số < 1000 ở cuối dòng = nghìn; `+`/`-` là chiều; `hôm qua`, `hôm kia`, `dd/mm`). Chat thường không có số tiền thì bỏ qua.
- Giao dịch tạo là giao dịch tay (`sources = ["manual"]`, cập nhật số dư tài khoản tay), `merchant` = mô tả, `raw_content` = dòng gốc. Phân loại `classifyQuick`: luật `learned` khớp đúng → luật `learned` nằm trong mô tả (dài nhất) → luật tay → `Keywords` của preset + `MerchantDictionary` (cụm dài nhất, chỉ tính danh mục đúng chiều) → không rõ thì `needs_review` và mở sẵn bảng chọn danh mục. Danh mục thu → chiều `in`.
- Sửa: nút ✏️ (callback `p/P/s/d/u/b:<tx id 32 hex>[:<10 hex cuối id danh mục>]`), reply tin bot bằng tên danh mục / số tiền / `xoá`, hoặc sửa tin gốc (`edited_message`). Chọn danh mục chạy `set_category` với `learn: true` → luật `merchant` theo mô tả, áp cho giao dịch tương tự; app và bot dùng chung luật.
- `thuchi_telegram_messages (chat_id, message_id)` → owner, id tin trả lời của bot, danh sách giao dịch; vừa chống ghi trùng khi nhận lại update, vừa tìm giao dịch khi reply/sửa/bấm nút. Chỉ chủ giao dịch bấm được nút.

## Thêm parser khi có mẫu thật

Thông báo/email chưa có parser vẫn được lưu và hiện ở *Cài đặt → Thông báo không đọc được*; viết parser xong bấm **Đọc lại**.

```go
func init() {
    thuchi.NotifyParsers["VCB"] = func(p *thuchi.NotifyPayload, receivedAt time.Time) (*thuchi.Incoming, error) {
        // trả Incoming{SourceRef (duy nhất, vd mã GD hoặc hash title+text+posted_at), AccountNumber, Direction, Amount,
        //            OccurredAt: thuchi.NotifyPostedAt(p, receivedAt), Content, BalanceAfter}
    }
    thuchi.EmailParsers["timo.vn"] = func(m *thuchi.EmailMessage) (*thuchi.Incoming, error) { /* … */ }
}
```

Package Android đã map sẵn trong `NotifyPackages` (VCB `com.VCB`, MoMo `com.mservice.momotransfer`, TPBank, Timo, MB, ACB, TCB, Cake, ZaloPay).

## Email ngân hàng + sao kê

- Một email có thể sinh nhiều giao dịch: `ParseEmailEvents` thử file đính kèm `.xls` trước (`ParseStatementFile` → `StatementParsers`), sau đó mới tới `EmailParsers[domain]`. Raw event lưu `transaction_id` của giao dịch đầu; tất cả trùng thì `duplicate`.
- VCB: `EmailParsers["vietcombank.com.vn"]`, chỉ tiền ra; `source_ref = vcb:<TK nguồn>:<số lệnh>`; phí người chuyển trả được cộng vào số tiền (ghép nội bộ sẽ tách phí).
- Timo: nhận dạng theo dòng tiêu đề (`Ngày GD`, `Mã giao dịch`, `Phát sinh có/nợ`, `Số dư`, `Diễn giải`); `source_ref = stmt:<TK>:<mã TF…>`, có số dư sau GD nên cập nhật số dư tài khoản.
- `.xls` đặt mật khẩu hoặc Excel 95 báo lỗi rõ ở *Thông báo không đọc được*; `.xlsx` chưa hỗ trợ.
- Cron kéo Gmail qua IMAP: `deploy/mail-pull/` (xem `deploy/README.md`).

## Còn để lại (chưa đủ thông tin)

- Parser thông báo VCB (chính chủ + chia sẻ biến động), MoMo — cần mẫu thật.
- Email Routing trên subdomain `in.thu-chi.mihb.site` khi domain gốc đã có MX riêng; dự phòng IMAP chưa làm.
- Đối soát SePay qua User API v2 khi miss webhook (cần API token; dedupe theo `referenceCode` đã sẵn).
- Bản đồ (giai đoạn 3), cảnh báo "24h không nhận thông báo" (giai đoạn 5).

## Dùng trong quanly-phongtro

Module này được copy sang `~/quanly-phongtro/pkg/thuchi` bằng `make sync-thuchi` (bên qlpt), chạy như plugin `thu-chi` (xem `plugins/README.md` bên đó). Điểm nối: `TelegramBot.OwnerActive` (bot bỏ qua owner đã tắt tiện ích), `ThuChiApp` với `embedded` (tab nằm trên, không nav/FAB đè layout host) và `onLogout` tuỳ chọn. Sửa ở `~/thu-chi` trước, giữ code không phụ thuộc host.
