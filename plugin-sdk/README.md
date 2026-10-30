# Bộ tiêu chuẩn plugin — Quản lý phòng trọ

Tài liệu này là **chuẩn bắt buộc** để developer bên ngoài viết plugin cho Quản lý phòng trọ. Plugin được đóng thành **một tệp `.zip` hoặc `.tar.gz`**; quản trị viên tải tệp lên ở tab **Tiện ích → Cài từ tệp** là plugin được cài ngay, không cần build lại hay khởi động lại app (giống EC-CUBE / Obsidian).

- **Backend** viết bằng Go, build sang WebAssembly và chạy trong sandbox: không đọc được file, biến môi trường hay mạng của máy chủ; chỉ làm việc với dữ liệu qua Host API, theo đúng quyền khai báo trong `manifest.json`.
- **Frontend** là một bundle JS dùng chung React, bộ UI (shadcn) và icon của app, nên plugin trông và hoạt động như một phần của app.
- Plugin mẫu đầy đủ cả hai phía: [`examples/debt-reminder`](examples/debt-reminder) — **lấy đây làm khuôn khi viết plugin mới.**

> Plugin có sẵn của hệ thống (biên dịch cùng app, nằm ở `plugins/`) theo tài liệu riêng [`plugins/README.md`](../plugins/README.md). Tài liệu này chỉ nói về plugin cài từ gói.

## 1. Bắt đầu nhanh

Yêu cầu: Go ≥ 1.24; Node ≥ 20 (chỉ khi plugin có giao diện).

```bash
cp -r plugin-sdk ~/qlpt-plugins && cd ~/qlpt-plugins
npm install                                   # chỉ cần khi có giao diện
cp -r examples/debt-reminder plugins/my-plugin
# đổi id ở 3 chỗ: manifest.json, web/index.tsx (definePlugin), createPluginApi(...)
go run ./cmd/pack plugins/my-plugin           # → dist/my-plugin-1.0.0.zip
```

Mọi lệnh chạy ở thư mục gốc của kit (nơi có `go.mod`, `package.json`). `cmd/pack` build backend (`GOOS=wasip1 GOARCH=wasm`), build giao diện (Vite), rồi nén đúng cấu trúc gói ở mục 2. Kiểm tra kiểu TypeScript: `npm run typecheck`.

## 2. Cấu trúc

### Mã nguồn (trong kit)

```
plugins/<id>/
  manifest.json          # bắt buộc
  backend/main.go        # tuỳ chọn: package main, dùng SDK qlpt
  web/index.tsx          # tuỳ chọn: export default definePlugin(...)
  web/*.tsx              # các component khác
  web/style.css          # tuỳ chọn: Tailwind của plugin
  README.md, CHANGELOG.md, LICENSE   # tuỳ chọn, được đưa vào gói
```

### Gói cài đặt (`dist/<id>-<version>.zip`)

```
manifest.json
backend/plugin.wasm      # nếu có backend
web/index.js             # nếu có giao diện
web/index.css            # nếu có style.css
README.md, CHANGELOG.md, LICENSE, LICENSE.md, LICENSE.txt   # tuỳ chọn
```

- Chấp nhận `.zip`, `.tar.gz`/`.tgz`, `.tar`; tối đa **32 MB**, 64 tệp, 64 MB sau giải nén.
- Gói được phép bọc trong **một** thư mục gốc (`my-plugin/manifest.json`...), host tự bóc.
- Tệp ngoài danh sách trên → **bị từ chối** (trừ rác hệ điều hành `__MACOSX/`, `.DS_Store`). Không đưa mã nguồn, `node_modules`, symlink vào gói.
- Phải có ít nhất `backend/plugin.wasm` hoặc `web/index.js`.

## 3. `manifest.json`

```json
{
  "api": 1,
  "id": "debt-reminder",
  "name": "Nhắc công nợ",
  "version": "1.0.0",
  "description": "Liệt kê hóa đơn chưa thanh toán ...",
  "author": "Tên bạn hoặc công ty",
  "homepage": "https://example.com/debt-reminder",
  "default_enabled": false,
  "permissions": ["invoices:read"],
  "settings": [
    { "key": "min_amount", "type": "number", "group": "Hiển thị", "label": "Chỉ hiện hóa đơn từ (₫)", "default": 0, "min": 0 }
  ]
}
```

| Trường | Bắt buộc | Quy tắc |
|---|---|---|
| `api` | ✔ | Phiên bản API plugin. Hiện là `1`; gói cần API cao hơn host hỗ trợ sẽ bị từ chối |
| `id` | ✔ | `^[a-z0-9][a-z0-9-]{1,62}$`, duy nhất, **không đổi** giữa các phiên bản; không trùng plugin có sẵn |
| `name` | ✔ | ≤ 80 ký tự, tiếng Việt có dấu |
| `version` | ✔ | Semver `MAJOR.MINOR.PATCH[-pre]`; tăng mỗi lần phát hành |
| `description` | | ≤ 500 ký tự, một câu nói plugin làm gì |
| `author` | | ≤ 120 ký tự |
| `homepage` | | URL `https://` |
| `default_enabled` | | Mặc định `false`: chủ trọ tự bật |
| `permissions` | | Quyền Host API cần dùng (mục 4); chỉ khai báo quyền thật sự dùng |
| `settings` | | Trang Cài đặt do host tự dựng (mục 3.1) |

Trường lạ (kể cả gõ sai như `permisions`) làm gói **bị từ chối**.

### 3.1 Cài đặt (`settings`)

Host tự dựng tab **Cài đặt**, kiểm tra kiểu dữ liệu và lưu **mã hoá AES-256-GCM theo từng chủ trọ**. Backend đọc bằng `c.Settings()`, frontend bằng `usePluginConfig(id)`.

| `type` | Giá trị | Ghi chú |
|---|---|---|
| `toggle` | `bool` | công tắc |
| `text` / `textarea` | `string` | ≤ 4000 ký tự; chuỗi rỗng = về `default` |
| `number` | `number` | `min` / `max` kiểm cả 2 phía |
| `select` | `string` | bắt buộc `options: [{ "value", "label" }]` |
| `password` | `string` | không bao giờ trả về trình duyệt; không được có `default` |

`key`: `^[a-z][a-z0-9_]{0,63}$`, không trùng. `label` bắt buộc; `group` gom các dòng liên tiếp dưới một tiêu đề; `description`, `placeholder` tuỳ chọn. Tối đa 50 cài đặt.

## 4. Quyền (`permissions`)

| Quyền | Cho phép |
|---|---|
| *(mặc định)* | Đọc request, trả response, đọc cài đặt, đọc/ghi Storage riêng của plugin |
| `invoices:read` | Đọc hóa đơn chưa thanh toán / đã thanh toán của chủ trọ đang đăng nhập; kiểm tra giao dịch đã ghi nhận |
| `invoices:write` | Đánh dấu hóa đơn đã thanh toán (ghi `payment_events`, idempotent theo mã giao dịch) |

Mọi dữ liệu luôn bị giới hạn trong **chủ trọ đang gọi** — plugin không thể đọc dữ liệu của chủ trọ khác. Gọi API thiếu quyền → lỗi 403. Quyền được hiển thị cho người dùng ở trang plugin.

## 5. Backend (Go → WebAssembly)

### 5.1 Mô hình chạy

- Request `/api/v1/plugins/<id>/<path>` (đã qua đăng nhập + plugin đang bật) → host khởi tạo **một sandbox mới** từ `plugin.wasm`, chạy `main()`, nhận response, huỷ sandbox (~5–10 ms).
- Hệ quả: **không có trạng thái giữa các request** (biến toàn cục bị reset). Dữ liệu cần giữ → `c.Storage()`.
- Không có mạng, không có file system, không có biến môi trường. Giờ hệ thống là UTC.
- `stdout`/`stderr` (`fmt.Println`, `log.Println`) được ghi vào log máy chủ (tối đa 8 KB/request) — dùng để debug, **không** ghi dữ liệu cá nhân.

Giới hạn: 30 giây/request, 256 MB RAM/sandbox, body request ≤ 25 MB, 4 request song song mỗi plugin, mỗi giá trị Storage ≤ 256 KB.

Đường dẫn dành riêng cho hệ thống (plugin không dùng): `/enabled`, `/config`, `/assets/*`.

### 5.2 SDK `qlpt`

```go
package main

import (
	"regexp"

	"github.com/mihb123/quanly-phongtro/plugin-sdk/qlpt"
)

var periodPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func main() {
	app := qlpt.New()
	app.Get("/summary", getSummary)
	app.Post("/reminders/{invoice_id}", markReminded)
	app.Run()
}

func getSummary(c *qlpt.Context) error {
	period := c.Query("period")
	if period != "" && !periodPattern.MatchString(period) {
		return qlpt.BadRequest("Kỳ phải có dạng YYYY-MM")
	}
	invoices, err := c.Invoices().ListOutstanding(period)
	if err != nil {
		return err
	}
	return c.JSON(200, invoices)
}
```

| API | Mô tả |
|---|---|
| `qlpt.New()`, `app.Get/Post/Put/Patch/Delete(pattern, h)`, `app.Run()` | Router. Pattern `"/notes/{id}"`, mỗi `{tên}` khớp 1 đoạn. Sai method → 405, không khớp → 404. `Run()` gọi đúng một lần cuối `main()` |
| `c.Param(name)`, `c.Query(name)`, `c.Header(name)`, `c.ManagerID()` | Đọc tham số. Header chỉ có `Content-Type`, `Accept`, `Accept-Language` |
| `c.Bind(&v)` | Giải mã body JSON; lỗi → trả sẵn `qlpt.BadRequest` |
| `c.Request.Body` | Body thô (`[]byte`), dùng cho upload multipart |
| `c.JSON(status, data)` | Trả `{status, data}` |
| `c.Message(status, message, data)` | Trả kèm thông báo cho người dùng |
| `c.File(contentType, filename, body)` | Trả tệp tải về (luôn `attachment`) |
| `return qlpt.BadRequest(msg)` / `NotFound` / `Conflict` / `qlpt.NewError(status, msg)` | Lỗi có chủ đích → `{status, message}` |
| `return err` (lỗi khác) | Ghi log, người dùng nhận 500 "Tiện ích gặp lỗi" |
| `c.Settings()` → `Values` | Cài đặt đã điền `default`; `.String/.Bool/.Number(key)` |
| `c.Storage().Get(key, &v)` / `Set(key, v)` / `Delete(key)` | Kho JSON riêng của plugin theo từng chủ trọ (mã hoá). Dùng cho dữ liệu nhỏ: trạng thái, đánh dấu — không dùng như database |
| `c.Invoices().ListOutstanding(periods...)` | `invoices:read` — hóa đơn `UNPAID`/`PENDING_VERIFICATION`, kỳ `YYYY-MM` (rỗng = mọi kỳ), kèm phòng, nhà, khách |
| `c.Invoices().ListPaid(periods...)` | `invoices:read` — hóa đơn `PAID` |
| `c.Invoices().RecordedPayment(accountRef, txRef)` | `invoices:read` — giao dịch này plugin đã ghi nhận chưa |
| `c.Invoices().MarkPaid(qlpt.Payment{...})` | `invoices:write` — đánh dấu đã thanh toán; trùng giao dịch / hóa đơn không còn chưa thanh toán → lỗi 409. `payment_events.provider` = id plugin, `payment_method` mặc định `PLUGIN` |

Lỗi từ Host API là `*qlpt.Error` (có `Status`), `return err` sẽ giữ nguyên mã lỗi đó.

Response luôn theo envelope chung của app: `{"status": 200, "data": ..., "message": "..."}`. Frontend đọc bằng `createPluginApi` (tự bóc `data`).

## 6. Frontend

### 6.1 Module được import

| Module | Nội dung |
|---|---|
| `react`, `react/jsx-runtime`, `react-dom` | React của app (React 19) |
| `@qlpt/plugin` | `definePlugin`, `createPluginApi`, `usePluginConfig`, `toast`, `refreshAppData`, `cn`, `apiVersion` |
| `@qlpt/plugin/ui` | `Button`, `Badge`, `Card*`, `Input`, `Textarea`, `Label`, `Checkbox`, `Switch`, `Select*`, `Table*`, `Tabs*`, `Dialog*`, `Tooltip*`, `Skeleton`, `Separator`, `EmptyState`, `ConfirmDialog`, `SearchInput`, `SectionCard`, `StatCard` |
| `@qlpt/plugin/icons` | Bộ icon của app (`Receipt`, `Check`, `Settings`, ... — xem `web/qlpt-plugin.d.ts`) |

Thư viện npm khác được bundle **vào** `index.js` (tăng dung lượng) — chỉ dùng khi thật cần. **Không** bundle React riêng.

### 6.2 Điểm vào `web/index.tsx`

```tsx
import { definePlugin } from '@qlpt/plugin'
import { Receipt } from '@qlpt/plugin/icons'
import { DebtReminderView } from './DebtReminderView'
import './style.css'

export default definePlugin({ id: 'debt-reminder', icon: Receipt, View: DebtReminderView })
```

- `id` trùng `manifest.json`. `View` nhận `{ plugin, openSettings }` (`openSettings()` chuyển sang tab Cài đặt).
- `createPluginApi('<id>')` → `get/post/put/patch/delete<T>(path, body?)` gọi backend của plugin (có xác thực DPoP sẵn), trả về `data`; `FormData` gửi multipart; `api.errorMessage(err, fallback)` lấy thông báo lỗi tiếng Việt từ backend.
- `usePluginConfig(id)` → `{ config, error, save, reload }`, dùng chung cache với tab Cài đặt (đổi cài đặt là view cập nhật ngay).
- `refreshAppData()` sau khi plugin sửa hóa đơn để các tab khác tải lại.

### 6.3 Style (Tailwind)

`web/style.css` chuẩn (giữ nguyên, chỉ thêm CSS riêng nếu cần):

```css
@import "tailwindcss/theme" layer(theme);
@import "tailwindcss/utilities" layer(utilities) source(none);
@import "../../../web/theme.css";

@source "./";
```

Chỉ sinh utility class plugin dùng, theo đúng token màu/cỡ chữ/bo góc của app (`web/theme.css`), hỗ trợ sáng/tối. Không import preflight, không viết CSS toàn cục (`body`, `*`, thẻ HTML).

## 7. Quy tắc code chuẩn

**Chung**
- Một plugin làm **một việc**. Tên, mô tả, thông báo lỗi bằng tiếng Việt có dấu.
- `id` thống nhất ở `manifest.json`, `definePlugin`, `createPluginApi`. Không đổi `id` sau khi phát hành.
- Không nhúng khoá bí mật trong mã; khoá/mật khẩu của người dùng khai báo bằng setting `password`.

**Backend**
- Kiểm tra mọi input (query, param, body) **trước** khi gọi Host API; sai → `qlpt.BadRequest("...")` với thông báo người dùng hiểu được.
- Mỗi handler: đọc input → kiểm tra → gọi Host API → `c.JSON` / `c.Message`. Logic dùng chung tách hàm thường, không tạo lớp trừu tượng thừa.
- Không giữ trạng thái trong biến toàn cục; dữ liệu cần giữ → `Storage`, dọn bản ghi cũ khi ghi (xem `markReminded` trong mẫu).
- Thao tác ghi (`MarkPaid`) phải idempotent: luôn truyền `TransactionRef` ổn định để gửi lại không bị ghi trùng.
- Không `panic`; lỗi bất ngờ `return err` để host ghi log.
- Chỉ dùng thư viện chuẩn Go (và SDK `qlpt`) để `plugin.wasm` nhỏ và build được sang `wasip1`.

**Frontend**
- Chỉ dùng component trong `@qlpt/plugin/ui` và icon trong `@qlpt/plugin/icons`; màu dùng token (`text-muted-foreground`, `bg-muted`, `text-destructive`...), không hardcode mã màu.
- Mobile-first: bảng ẩn bớt cột trên màn hẹp (`hidden sm:table-cell`), không gây cuộn ngang trang.
- Đủ 3 trạng thái: đang tải (`Skeleton`), lỗi (`EmptyState` + nút Thử lại), rỗng (`EmptyState`). Thao tác ghi: khoá nút khi đang gửi, báo kết quả bằng `toast`.
- Gọi backend qua `createPluginApi`, không tự gọi `fetch`/`axios`. Không lưu dữ liệu nghiệp vụ ở `localStorage`.

## 8. Cài đặt, cập nhật, gỡ (quản trị viên)

- Bật quyền cài: khai `PLUGIN_ADMIN_EMAILS=email1,email2` trong `.env` của máy chủ (trống = không ai cài được; plugin đã cài vẫn chạy). Thư mục lưu plugin: `PLUGIN_DIR` (mặc định `uploads/plugins`, được giữ qua các lần deploy và backup).
- **Cài**: tab Tiện ích → **Cài từ tệp** → chọn gói. Host kiểm tra gói, manifest, biên dịch `plugin.wasm` rồi mới thay thế; gói lỗi không ảnh hưởng plugin đang chạy. Plugin mới cài ở trạng thái tắt (trừ khi `default_enabled`), mỗi chủ trọ tự bật.
- **Cập nhật**: tải gói cùng `id` với `version` mới. Cài đặt và Storage của chủ trọ được giữ nguyên.
- **Gỡ**: nút thùng rác trên dòng plugin. Tệp plugin bị xoá; cài đặt/Storage của chủ trọ được giữ để cài lại.

## 9. Plugin mẫu `debt-reminder` (Nhắc công nợ)

| Tệp | Minh hoạ |
|---|---|
| `manifest.json` | Khai báo quyền `invoices:read`, 2 cài đặt `number` có `group`, `min`/`max`, `default` |
| `backend/main.go` | Router 3 route, validate query (`period`), đọc cài đặt, đọc hóa đơn, lưu/xoá đánh dấu trong Storage và dọn bản ghi cũ, trả lỗi 400/404 đúng chuẩn |
| `web/index.tsx` | Điểm vào `definePlugin` + icon |
| `web/DebtReminderView.tsx` | Bộ lọc kỳ (Tabs), thẻ tổng quan (`StatCard`), bảng có dòng Tổng, đủ trạng thái tải/lỗi/rỗng, nút thao tác có khoá khi đang gửi + `toast`, tự tải lại khi đổi cài đặt (`usePluginConfig`) |
| `web/style.css` | Style chuẩn mục 6.3 |

API của mẫu: `GET /summary?period=YYYY-MM`, `POST /reminders/{invoice_id}`, `DELETE /reminders/{invoice_id}`.

## 10. Checklist trước khi phát hành

- [ ] `go vet ./...` và `GOOS=wasip1 GOARCH=wasm go vet ./...` sạch; `npm run typecheck` sạch.
- [ ] `version` đã tăng; `CHANGELOG.md` ghi thay đổi.
- [ ] Chỉ khai báo quyền thật sự dùng.
- [ ] Mọi input được kiểm tra; mọi lỗi trả thông báo tiếng Việt.
- [ ] Giao diện ổn ở màn hình điện thoại và desktop, chế độ sáng và tối.
- [ ] Cài thử gói trong `dist/` lên môi trường dev: cài → bật → dùng → đổi cài đặt → cập nhật bản mới → gỡ.

## 11. Tham chiếu ABI (API 1)

Dành cho người viết SDK bằng ngôn ngữ khác (Rust, TinyGo, AssemblyScript...). SDK Go đã cài sẵn phần này.

- Module là **WASI preview 1 command** (có export `_start`). Chỉ được import từ `wasi_snapshot_preview1` và `qlpt`.
- `qlpt.call(ptr: i32, len: i32) -> i32`: gửi lệnh JSON `{"op": "...", "params": {...}}`, trả độ dài kết quả.
- `qlpt.result(ptr: i32)`: chép kết quả của lần `call` gần nhất vào bộ nhớ tại `ptr`.
- Kết quả: `{"ok": true, "data": ...}` hoặc `{"ok": false, "status": 403, "error": "..."}`.

| `op` | Quyền | `params` | `data` |
|---|---|---|---|
| `request` | | | `{method, path, query, header, body (base64), manager_id}` |
| `respond` | | `{status, data, message}` hoặc `{status, content_type, filename, body (base64)}` | |
| `config.get` | | | Giá trị cài đặt (đã điền `default`) |
| `storage.get` | | `{key}` | Giá trị JSON hoặc `null` |
| `storage.set` | | `{key, value}` (`value: null` = xoá) | |
| `invoices.list_outstanding` | `invoices:read` | `{periods: ["YYYY-MM"]}` | `Invoice[]` |
| `invoices.list_paid` | `invoices:read` | `{periods}` | `Invoice[]` |
| `invoices.recorded_payment` | `invoices:read` | `{account_ref, transaction_ref}` | `{invoice_id, recorded}` |
| `invoices.mark_paid` | `invoices:write` | `{invoice_id, amount, account_ref, transaction_ref, matching_method, payment_method, raw_payload}` | |

Chương trình phải gọi `request` rồi `respond` đúng một lần; thoát mà chưa `respond` → 500.

Bundle giao diện: định dạng IIFE gán vào biến `__qlpt_plugin__` (`exports.default = definePlugin(...)`), các module dùng chung lấy từ biến `__qlpt_react__`, `__qlpt_jsx__`, `__qlpt_react_dom__`, `__qlpt_sdk__`, `__qlpt_ui__`, `__qlpt_icons__` — `web/vite.config.ts` đã cấu hình sẵn.
