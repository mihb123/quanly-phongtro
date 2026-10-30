# Plugin

Mỗi plugin là **một thư mục** trong `plugins/<tên>/`, gồm phần backend Go và (tuỳ chọn) giao diện React. Thả thư mục vào, chạy `make plugins`, build lại là dùng được — không phải sửa code lõi.

```
plugins/
  plugins.go            # sinh tự động bởi `make plugins` (blank import mọi plugin)
  <tên>/
    plugin.go           # đăng ký plugin (Go)
    ...                 # code nghiệp vụ của plugin
    web/index.tsx       # (tuỳ chọn) giao diện, tự được frontend nhận
```

Người dùng bật/tắt plugin và mở giao diện của nó ở tab **Tiện ích**. Trạng thái bật/tắt lưu theo từng chủ trọ (bảng `plugin_settings`).

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
- Route được mount tại `/api/v1/plugins/<ID>/...`, đã qua xác thực (JWT + DPoP, vai trò MANAGER) và chỉ chạy khi chủ trọ đang bật plugin (tắt → 403). Đường dẫn `/enabled` ở gốc plugin là của hệ thống, không dùng.
- `Init` lỗi → plugin bị bỏ qua (ghi log), app vẫn chạy.

### Host API

| API | Mô tả |
|---|---|
| `host.Invoices().ListOutstanding(ctx, managerID)` | Hóa đơn `UNPAID` / `PENDING_VERIFICATION` của chủ trọ, kèm tên phòng, nhà, khách đang thuê |
| `host.Invoices().PaymentRecorded(ctx, source, accountRef, txRef)` | Giao dịch đã được ghi nhận chưa (chống xử lý lặp) |
| `host.Invoices().MarkPaid(ctx, managerID, model.PluginPayment{...})` | Đánh dấu đã thanh toán + ghi `payment_events` trong 1 transaction, idempotent theo `(source, accountRef, txRef)` |
| `host.Storage().Load/Save/Clear(ctx, managerID, ...)` | Lưu cấu hình riêng của plugin theo chủ trọ (JSON, mã hoá AES-256-GCM) |

Plugin chỉ đi qua Host API, không import repository/service nội bộ, để lõi đổi bên trong mà plugin không vỡ.

## Frontend

`plugins/<tên>/web/index.tsx` export mặc định một `definePlugin(...)`; frontend tự quét bằng `import.meta.glob` và tách thành chunk riêng (chỉ tải khi mở tab Tiện ích).

```tsx
import { Puzzle } from '@/components/icons'
import { createPluginApi, definePlugin, toast, type PluginViewProps } from '@/plugins/sdk'

const api = createPluginApi('hello')

function HelloView({ plugin }: PluginViewProps) {
  return <button onClick={() => api.get<{ manager_id: string }>('/ping').then(() => toast.success(plugin.name))}>Ping</button>
}

export default definePlugin({ id: 'hello', icon: Puzzle, View: HelloView })
```

- `id` phải trùng `Manifest().ID`.
- Chỉ import `react` và các module `@/...` của app (UI shadcn `@/components/ui/*`, icon `@/components/icons`, `@/plugins/sdk`). Thư viện khác chưa được dedupe sẽ không resolve được từ ngoài `frontend/`.
- `createPluginApi(id)` gọi `/api/v1/plugins/<id>/...` qua `apiClient` (có DPoP), tự bóc `data`; `FormData` được gửi multipart. `refreshAppData()` làm mới dữ liệu đang hiển thị sau khi plugin sửa hóa đơn.

## Thêm / gỡ plugin

1. Thêm hoặc xoá thư mục `plugins/<tên>/`.
2. `make plugins` (sinh lại `plugins/plugins.go`).
3. Cần bảng riêng thì thêm migration vào `migrations/` như bình thường.
4. Build/deploy như thường lệ.

## Plugin có sẵn

- `bankstatement` (`bank-statement`): tải sao kê ngân hàng (.zip có/không mật khẩu hoặc .pdf), đọc các khoản ghi có, khớp với hóa đơn chưa thanh toán theo đúng số tiền (ưu tiên số phòng / tên khách / tháng trong nội dung chuyển khoản); lệch dưới 1.000 ₫ (khách làm tròn) chỉ được nhận khi nội dung ghi tên khách hoặc số phòng và chỉ đúng 1 hóa đơn khớp, đánh dấu đã thanh toán (`payment_method = BANK_STATEMENT`). Có chế độ xem trước và ghi nhớ mật khẩu. Hiện hỗ trợ TPBank (phiếu báo có, fallback sổ phụ tài khoản); thêm ngân hàng khác bằng cách cài `StatementParser` trong `statement.go`.
