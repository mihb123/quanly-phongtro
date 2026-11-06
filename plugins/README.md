# Plugin

> Plugin của bên thứ ba **cài bằng gói .zip/.tar.gz** (không build lại app) theo chuẩn riêng: [`plugin-sdk/README.md`](../plugin-sdk/README.md). Tài liệu dưới đây dành cho plugin có sẵn biên dịch cùng app.

Mỗi plugin là **một thư mục** trong `plugins/<tên>/`, gồm phần backend Go và (tuỳ chọn) giao diện React. Thả thư mục vào, chạy `make plugins`, build lại là dùng được — không phải sửa code lõi.

```
plugins/
  plugins.go            # sinh tự động bởi `make plugins` (blank import mọi plugin)
  <tên>/
    plugin.go           # đăng ký plugin (Go)
    ...                 # code nghiệp vụ của plugin
    web/index.tsx       # (tuỳ chọn) giao diện, tự được frontend nhận
```

Người dùng quản lý plugin ở tab **Tiện ích** (bố cục giống Settings → Community plugins của Obsidian): trang "Quản lý tiện ích" liệt kê mọi plugin với công tắc bật/tắt và nút cài đặt; cột trái liệt kê plugin đang bật; mỗi plugin có tab "Sử dụng" (giao diện riêng) và "Cài đặt" (host tự dựng từ khai báo `Settings`, lưu ngay khi đổi). Trạng thái bật/tắt và cài đặt lưu theo từng chủ trọ (bảng `plugin_settings`); tắt plugin không xoá cài đặt.

## Backend

Plugin cài interface `plugin.Plugin` (`internal/plugin`) và tự đăng ký trong `init()`:

```go
package hello

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/plugin"
)

func init() { plugin.Register(&Plugin{}) }

type Plugin struct{ host plugin.Host }

func (p *Plugin) Manifest() plugin.Manifest {
	return plugin.Manifest{
		ID:             "hello",
		Name:           "Xin chào",
		Version:        "1.0.0",
		Description:    "Plugin mẫu",
		Author:         "Tên bạn",
		DefaultEnabled: false,
		Settings: []plugin.SettingField{
			{Key: "greeting", Type: plugin.SettingText, Group: "Chung", Label: "Lời chào", Default: "Xin chào"},
			{Key: "loud", Type: plugin.SettingToggle, Label: "Viết hoa", Description: "In lời chào bằng chữ in hoa"},
		},
	}
}

func (p *Plugin) Init(host plugin.Host) error { p.host = host; return nil }

func (p *Plugin) Routes(r chi.Router) {
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		managerID, ok := plugin.ManagerID(w, r)
		if !ok {
			return
		}
		plugin.WriteJSON(w, http.StatusOK, map[string]string{"manager_id": managerID}, "")
	})
}
```

- `ID`: chữ thường, số, gạch ngang (`^[a-z0-9][a-z0-9-]{1,62}$`), duy nhất. Trùng ID thì app panic lúc khởi động.
- Route được mount tại `/api/v1/plugins/<ID>/...`, đã qua xác thực (JWT + DPoP, vai trò MANAGER) và chỉ chạy khi chủ trọ đang bật plugin (tắt → 403). Đường dẫn `/enabled` và `/config` ở gốc plugin là của hệ thống, không dùng.
- `Init` lỗi → plugin bị bỏ qua (ghi log), app vẫn chạy.

### Host API

| API | Mô tả |
|---|---|
| `host.Invoices().ListOutstanding(ctx, managerID, periods)` | Hóa đơn `UNPAID` / `PENDING_VERIFICATION` của chủ trọ thuộc các kỳ `periods` (`YYYY-MM`, rỗng = mọi kỳ), kèm tên phòng, nhà, khách đang thuê |
| `host.Invoices().ListPaid(ctx, managerID, periods)` | Hóa đơn `PAID` thuộc các kỳ `periods`, cùng cấu trúc như trên |
| `host.Invoices().RecordedPayment(ctx, source, accountRef, txRef)` | Giao dịch đã được ghi nhận chưa (chống xử lý lặp) và hóa đơn nó đã thanh toán |
| `host.Invoices().MarkPaid(ctx, managerID, model.PluginPayment{...})` | Đánh dấu đã thanh toán + ghi `payment_events` trong 1 transaction, idempotent theo `(source, accountRef, txRef)` |
| `host.Config().Get(ctx, managerID)` | Cài đặt của chủ trọ (`plugin.Values`, đã điền `Default`), đọc bằng `.String/.Bool/.Number(key)` |
| `host.Config().Set(ctx, managerID, map[string]any{...})` | Gộp giá trị vào cài đặt (`nil` = xoá key). Key ngoài `Settings` vẫn lưu được — dùng làm dữ liệu riêng, API không trả ra ngoài |
| `host.Enabled(ctx, managerID)` | Chủ trọ đang bật plugin chưa (đã tính `DefaultEnabled`) — dùng cho việc chạy nền ngoài request (bot, worker) |

### Route công khai (webhook)

Plugin cài thêm `plugin.PublicRouter` (`PublicRoutes(r chi.Router)`) thì host mount `/api/v1/plugins/<ID>/public/{managerID}/*` **không qua đăng nhập**, giới hạn 5 req/s (burst 60) theo IP. Host chỉ gọi plugin khi `managerID` là UUID và chủ trọ đó đang bật plugin, ngược lại trả 404; `plugin.PublicManagerID(r)` lấy `managerID`. Plugin tự xác thực request (HMAC, token thiết bị…).

### Plugin built-in cần hạ tầng của app

Plugin không import `internal/...`. Khi cần DB/mã hoá/worker (như `thuchi`), adapter `internal/service/<tên>/<tên>_adapter.go` dựng module rồi `cmd/api/main.go` gọi hàm `Use(...)` của plugin **trước** `plugin.NewManager`; `Init` báo lỗi nếu chưa được nối.

### Cài đặt (`Manifest.Settings`)

Khai báo cài đặt trong manifest, host tự dựng trang **Cài đặt** và API `GET/PUT /api/v1/plugins/<ID>/config` (kiểm tra kiểu, `Min`/`Max`, `Options`; key lạ → 400). Toàn bộ cài đặt mã hoá AES-256-GCM trong `plugin_settings.encrypted_data`.

| `Type` | Điều khiển | Ghi chú |
|---|---|---|
| `SettingToggle` | công tắc | `bool` |
| `SettingText` / `SettingTextarea` | ô nhập, tự lưu sau 0,8s | chuỗi rỗng = xoá (về `Default`) |
| `SettingNumber` | ô số | `Min`/`Max` kiểm cả 2 phía |
| `SettingSelect` | danh sách chọn | giá trị phải nằm trong `Options` |
| `SettingPassword` | Đặt / Đổi / Xoá | không bao giờ trả giá trị về trình duyệt, chỉ trả đã đặt hay chưa |

`Group` gom các dòng liên tiếp dưới một tiêu đề; `Description`, `Placeholder` hiện kèm ô nhập.

Plugin chỉ đi qua Host API, không import repository/service nội bộ, để lõi đổi bên trong mà plugin không vỡ.

## Frontend

`plugins/<tên>/web/index.tsx` export mặc định một `definePlugin(...)`; frontend tự quét bằng `import.meta.glob` và tách thành chunk riêng (chỉ tải khi mở tab Tiện ích).

```tsx
import { Puzzle } from '@/components/icons'
import { createPluginApi, definePlugin, toast, usePluginConfig, type PluginViewProps } from '@/plugins/sdk'

const api = createPluginApi('hello')

function HelloView({ plugin, openSettings }: PluginViewProps) {
  const { config } = usePluginConfig(plugin.id)
  return (
    <>
      <button onClick={() => api.get<{ manager_id: string }>('/ping').then(() => toast.success(String(config?.values.greeting ?? plugin.name)))}>Ping</button>
      <button onClick={openSettings}>Cài đặt</button>
    </>
  )
}

export default definePlugin({ id: 'hello', icon: Puzzle, View: HelloView })
```

- `id` phải trùng `Manifest().ID`.
- Chỉ import `react` và các module `@/...` của app (ngoại lệ: alias của module `pkg/<tên>/web` như `@thuchi/*`, đã khai trong Vite + tsconfig; thư viện riêng của module như `dexie` phải có trong `frontend/package.json` và `dedupe`) (UI shadcn `@/components/ui/*`, icon `@/components/icons`, `@/plugins/sdk`). Thư viện khác chưa được dedupe sẽ không resolve được từ ngoài `frontend/`.
- `View` nhận `plugin` và `openSettings()` (chuyển sang tab Cài đặt). Plugin không có `View` nhưng có `Settings` vẫn mở được (chỉ có trang Cài đặt).
- `usePluginConfig(id)` trả `{ config, error, save, reload }` (`config.values`, `config.secrets`), dùng chung cache với trang Cài đặt nên đổi ở đâu cũng cập nhật ngay.
- Class Tailwind trong `plugins/*/web` được quét nhờ `@source` trong `frontend/src/index.css`.
- `createPluginApi(id)` gọi `/api/v1/plugins/<id>/...` qua `apiClient` (có DPoP), tự bóc `data`; `FormData` được gửi multipart. `refreshAppData()` làm mới dữ liệu đang hiển thị sau khi plugin sửa hóa đơn.

## Thêm / gỡ plugin

1. Thêm hoặc xoá thư mục `plugins/<tên>/`.
2. `make plugins` (sinh lại `plugins/plugins.go`).
3. Cần bảng riêng thì thêm migration vào `migrations/` như bình thường.
4. Build/deploy như thường lệ.

## Plugin có sẵn

- `thuchi` (`thu-chi`): sổ thu chi cá nhân local-first của từng chủ trọ (dữ liệu riêng, owner = managerID), tắt mặc định. Lõi `pkg/thuchi` copy từ repo `~/thu-chi` bằng `make sync-thuchi` (đổi import path); sửa lõi ở `~/thu-chi` trước. Adapter `internal/service/thuchi/thuchi_adapter.go` (PGStore trên `*sql.DB`, AES-GCM bằng `APP_SECRET_ENCRYPTION_KEY`, worker raw event + định kỳ + dọn tombstone). Route đã đăng nhập `/api/v1/plugins/thu-chi/{sync,ingest,telegram}/...`, ingest công khai `/api/v1/plugins/thu-chi/public/{managerID}/ingest/{sepay|notify|email}`, bảng `thuchi_*` (migration `000032` = `pkg/thuchi/sql/schema.sql`). Web: `plugins/thuchi/web` mount `ThuChiApp` (`@thuchi/App`, chế độ `embedded`: tab ở trên, không nav cố định) với transport axios tự refresh token khi 401; đăng xuất xoá IndexedDB `thuchi_*`. Bot Telegram bật khi có `THUCHI_TELEGRAM_BOT_TOKEN` (bot riêng, long polling; `THUCHI_TELEGRAM_CHAT_ID` giới hạn nhóm), bot bỏ qua chủ trọ đã tắt tiện ích. E2E: `frontend/e2e/thuchi/` (`make e2e-thuchi`).

- `bankstatement` (`bank-statement`): tải sao kê ngân hàng (.zip có/không mật khẩu hoặc .pdf), đọc các khoản ghi có, khớp với hóa đơn chưa thanh toán theo đúng số tiền (ưu tiên số phòng / tên khách / tháng trong nội dung chuyển khoản); lệch dưới 1.000 ₫ (khách làm tròn) chỉ được nhận khi nội dung ghi tên khách hoặc số phòng và chỉ đúng 1 hóa đơn khớp, đánh dấu đã thanh toán (`payment_method = BANK_STATEMENT`). Cài đặt: luôn xem trước, mức lệch làm tròn chấp nhận (0 / 1.000 / 5.000 / 10.000 ₫), mật khẩu giải nén. Sau khi xem trước có thể cập nhật từng giao dịch đã chọn hoặc tất cả. Hiện hỗ trợ TPBank (phiếu báo có, fallback sổ phụ tài khoản); thêm ngân hàng khác bằng cách cài `StatementParser` trong `statement.go`.
