# Demo View `/h/example` — Implementation Plan (v2)

Tạo một view public tại `/h/example` cho phép mọi người trải nghiệm app mà không cần tài khoản. Dùng SQLite DB riêng để cách ly dữ liệu, có command reset data.

## Changes from v1

- ✅ Thêm rate limiting 60 req/min/IP cho `/api/v1/example/*`
- ✅ Example routes tách ra file riêng (`example_router.go`) thay vì sửa `router.go`
- ✅ Frontend redesign: **minimal changes** — chỉ sửa `apiClient` interceptor, không sửa bất kỳ API function hay Zustand store nào

## User Review Required

> [!IMPORTANT]
> **SQLite compatibility**: Repository dùng một số Postgres-specific SQL:
> - `ILIKE` → SQLite dùng `LIKE` (case-insensitive by default cho ASCII)
> - `NOW()` → SQLite dùng `datetime('now')`
> - `JSONB` → SQLite lưu dưới dạng `TEXT`
>
> **Approach**: Tạo một lớp **repository wrappers** mỏng cho example, chỉ override các methods có raw Postgres SQL. Các methods dùng Bun ORM thuần sẽ hoạt động bình thường vì Bun abstract dialect.

> [!WARNING]
> **Concurrent SQLite writes**: Enable WAL mode (`PRAGMA journal_mode=WAL`) để tránh lock khi nhiều demo users thao tác cùng lúc.

## Proposed Changes

---

### 1. Database Layer — SQLite Connection

#### [NEW] [sqlite.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/db/sqlite.go)
- Function `NewSQLite(path string) (*bun.DB, error)`:
  - Open SQLite file, enable WAL mode + foreign keys
  - Dùng `bun/dialect/sqlitedialect`
- Function `CreateExampleSchema(db *bun.DB) error`:
  - Tạo tables bằng `CREATE TABLE IF NOT EXISTS` (SQLite syntax)
  - Tables: `users`, `houses`, `rooms`, `tenants`, `invoices`, `house_costs`, `house_revenue_summaries`
  - Không tạo tables cho auth sessions, email verification, otp checks, invoice payments, pending invoice updates (không cần cho demo)
- Dependency mới: `github.com/uptrace/bun/dialect/sqlitedialect`, `github.com/mattn/go-sqlite3`

---

### 2. Backend — Example Router (file riêng)

#### [NEW] [example_router.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/router/example_router.go)
Tách toàn bộ logic example routes vào file riêng, gồm:

**Middleware trong file này:**
- `exampleContext()`: Inject fixed claims vào context:
  - `UserID`: `"00000000-0000-0000-0000-000000000001"`
  - `Role`: `"MANAGER"`
- `blockFileUpload()`: Reject requests có `Content-Type: multipart/form-data` → 403 "File upload is disabled in demo mode"
- `exampleRateLimiter()`: Rate limit **60 requests/phút/IP** cho toàn bộ group. Dùng `golang.org/x/time/rate` (đã có trong go.mod).

**Function chính:**
```go
// MountExampleRoutes gắn example API routes vào router hiện có.
// Tạo bộ repositories + services + handlers riêng dùng exampleDB.
func MountExampleRoutes(r chi.Router, exampleDB *bun.DB)
```

Bên trong function này:
1. Tạo repositories dùng `exampleDB`
2. Tạo services dùng repositories trên (không inject email sender, zalo client, payment provider — chỉ core CRUD)
3. Tạo handlers
4. Mount routes:

```
r.Route("/api/v1/example", func(r chi.Router) {
    r.Use(exampleRateLimiter)     // 60 req/min/IP
    r.Use(exampleContext)          // inject fake manager claims
    r.Use(blockFileUpload)         // chặn multipart upload

    // House routes
    r.Post("/house/create", exampleHouseHandler.CreateHouse)
    r.Get("/house/{id}", exampleHouseHandler.GetHouseByID)
    r.Get("/house/", exampleHouseHandler.ListHouseByManagerID)
    r.Post("/house/{id}", exampleHouseHandler.UpdateHouse)
    r.Delete("/house/{id}", exampleHouseHandler.DeleteHouse)

    // Room routes
    r.Post("/room/", exampleRoomHandler.CreateRoom)
    r.Get("/room/", exampleRoomHandler.ListRooms)
    ... (tương tự cho room, tenant, invoice, house-cost, revenue-summary)

    // KHÔNG mount: auth, zalo, payments, uploads, settings
})
```

#### [MODIFY] [router.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/router/router.go)
- Chỉ thay đổi **nhỏ nhất**: thêm optional parameter `exampleDB *bun.DB` vào `New()` function.
- Nếu `exampleDB != nil`, gọi `MountExampleRoutes(r, exampleDB)`.
- Không thay đổi bất kỳ route hiện tại nào.

#### [MODIFY] [main.go](file:///media/minhchu1336/Data/quanly-phongtro/cmd/api/main.go)
- Thêm logic: kiểm tra file `example.db` tồn tại → `db.NewSQLite("example.db")` → truyền vào router.
- Nếu file không tồn tại, `exampleDB = nil` → demo mode bị disable (graceful).

---

### 3. Seed Command — Example Data

#### [NEW] [main.go](file:///media/minhchu1336/Data/quanly-phongtro/cmd/example-seed/main.go)

Command `go run cmd/example-seed/main.go`:
1. Xóa file `example.db` nếu tồn tại
2. Tạo SQLite DB mới, gọi `CreateExampleSchema()`
3. Seed dữ liệu mẫu:

| Entity | Qty | Chi tiết |
|--------|-----|----------|
| User (Manager) | 1 | ID cố định `00000000-...0001`, `demo@example.com` |
| Houses | 3 | "Nhà trọ Bình Thạnh" (5 rooms), "Nhà trọ Thủ Đức" (5 rooms), "Nhà trọ Quận 7" (5 rooms) |
| Rooms | 15 | Mix occupied/available, giá 2.5M–5M, chỉ số điện nước khác nhau |
| Tenant Users | 20 | Tên tiếng Việt thực tế, email/phone giả |
| Tenants | 20 | Mix ACTIVE/INACTIVE, phân bổ đều vào rooms |
| Invoices | 18+ | 3 tháng gần nhất, mix PAID/UNPAID, chỉ số điện nước tăng dần |
| House Costs | 9 | 3 tháng × 3 houses |
| Revenue Summaries | 9 | Tổng hợp doanh thu/chi phí/lợi nhuận |

---

### 4. Frontend — Minimal Changes Approach

**Nguyên tắc thiết kế**: Tận dụng việc tất cả API functions đều dùng singleton `apiClient`. Chỉ cần sửa `apiClient` interceptor để auto-detect example mode → **zero changes** tới API functions và Zustand stores.

#### [NEW] [example.ts](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/utils/example.ts)
```typescript
// Kiểm tra app đang ở example/demo mode dựa trên URL path.
export function isExampleMode(): boolean {
  return window.location.pathname.startsWith('/h/example')
}
```

#### [MODIFY] [client.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/api/client.tsx)
Sửa interceptor để detect example mode:
```typescript
apiClient.interceptors.request.use(async (config) => {
  // Trong example mode: đổi baseURL và bỏ qua DPoP
  if (isExampleMode()) {
    config.baseURL = '/api/v1/example'
    return config // skip DPoP proof
  }

  // Production: giữ nguyên logic DPoP hiện tại
  const method = config.method || 'GET'
  const url = buildDPoPUrl(...)
  // ... (code DPoP hiện tại, không đổi)
  return config
})
```

**Tại sao approach này an toàn:**
- `apiClient` singleton tự động switch baseURL → tất cả API calls tự trỏ đến `/api/v1/example/*`
- Backend example routes dùng SQLite DB riêng → không thể access production data
- Không gửi DPoP proof → không leak crypto keys
- Nếu ai đó cố gắng gọi `/api/v1/*` (production) từ demo page, request sẽ fail vì không có auth token

#### [MODIFY] [AuthContext.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/contexts/AuthContext.tsx)
Trong example mode, skip auth flow và inject fake user:
```typescript
useEffect(() => {
  if (isExampleMode()) {
    setUser({ id: '00000000-...0001', email: 'demo@example.com', 
              role: 'MANAGER', full_name: 'Demo Manager', ... })
    setIsLoading(false)
    return
  }
  // ... logic auth hiện tại không đổi
}, [])
```

#### [MODIFY] [router/index.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/router/index.tsx)
Thêm route duy nhất:
```typescript
{
  path: '/h/example',
  element: <ExampleHomePage />,  // không wrap ProtectedRoute
}
```

#### [NEW] [ExampleHome.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/pages/ExampleHome.tsx)
- Import và render `DemoBanner` + nội dung giống `HomePage`
- Pass prop hoặc dùng `isExampleMode()` để ẩn Settings tab (Zalo/PayOS)

#### [MODIFY] [Home.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/pages/Home.tsx)
- Thay thế render Settings bằng check `isExampleMode()`:
```typescript
{activeTab === 'settings' && !isExampleMode() && <SettingsView />}
{activeTab === 'settings' && isExampleMode() && <DemoSettingsPlaceholder />}
```
- Hoặc đơn giản hơn: hiển thị message "Tính năng này chỉ có trong bản đầy đủ" khi ở demo mode.

#### [NEW] [DemoBanner.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/components/DemoBanner.tsx)
- Banner cố định ở đầu trang:
  - Background gradient tím (match theme)
  - "🎯 Bạn đang xem bản demo — Dữ liệu được reset định kỳ"
  - Button "Đăng ký miễn phí →" link đến `/register`
  - Height ~40px, responsive trên mobile

---

### Tổng kết thay đổi Frontend

| File | Thay đổi |
|------|----------|
| `api/client.tsx` | Thêm 4 dòng trong interceptor (detect + switch baseURL) |
| `contexts/AuthContext.tsx` | Thêm 5 dòng (early return fake user) |
| `router/index.tsx` | Thêm 4 dòng (1 route mới) |
| `pages/Home.tsx` | Thêm 2 dòng (hide Settings in example) |
| **Tổng modified** | **~15 dòng sửa** |
| `utils/example.ts` | **[NEW]** 4 dòng |
| `pages/ExampleHome.tsx` | **[NEW]** ~25 dòng |
| `components/DemoBanner.tsx` | **[NEW]** ~30 dòng |
| **Tổng new** | **~60 dòng mới** |
| **API functions** | **0 thay đổi** |
| **Zustand stores** | **0 thay đổi** |

---

## Security Analysis

| Rủi ro | Biện pháp |
|--------|-----------|
| Cross-contamination (demo ↔ production) | SQLite DB riêng, handler dùng repo riêng |
| Auth bypass lợi dụng example routes | Example routes chỉ access SQLite, không thể query Postgres |
| API spam (vô auth) | Rate limit 60 req/min/IP |
| File upload abuse | `blockFileUpload` middleware reject multipart |
| Config exposure (Zalo/PayOS) | Routes Zalo/PayOS không mount; Settings ẩn trên frontend |
| DPoP key leak | Example mode bỏ qua DPoP hoàn toàn |
| SQLi | Bun ORM parameterized queries (giống production) |
| XSS stored | React auto-escapes (giống production) |

---

## Verification Plan

### Automated Tests
```bash
# Build check
go build ./cmd/api/...
go build ./cmd/example-seed/...

# Existing tests không bị break
go test ./internal/... -count=1

# Seed và verify
go run cmd/example-seed/main.go
ls -la example.db

# Frontend build
cd frontend && pnpm build
```

### Manual Verification
1. `go run cmd/example-seed/main.go` → file `example.db` được tạo
2. Start server → `/h/example` → hiển thị data mẫu, banner demo
3. CRUD demo: tạo room, thêm tenant → thành công trên SQLite
4. Upload file → bị chặn 403
5. Settings tab → ẩn Zalo/PayOS
6. Spam requests → bị rate limit sau 60 req/min
7. Re-run seed → data reset về ban đầu
8. Production routes (`/api/v1/*`) → vẫn hoạt động bình thường
