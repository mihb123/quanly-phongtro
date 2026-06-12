# Doanh thu (Revenue & Operating Costs) Feature

Thêm tab "Doanh thu" để manager quản lý chi phí vận hành hàng tháng, theo dõi tổng thu/tổng chi/lợi nhuận ròng cho từng nhà trọ theo kỳ.

## Tổng hợp quyết định đã thống nhất

| Quyết định | Kết quả |
|---|---|
| Database chi phí | **1 bảng `house_costs`** — cột cố định cho 6 loại chính + `extra_costs JSONB` cho tùy chỉnh |
| Database summary | **Bảng `house_revenue_summaries`** — lưu sẵn tổng thu/chi/lợi nhuận per house+period |
| Cập nhật summary | **Goroutine + channel** — async recalculate khi invoice hoặc cost thay đổi |
| Tab name | **"Doanh thu"** (`revenue`) — thay vì "Chi phí vận hành" |
| Layout tab | **1 trang gộp**: summary cards (trên) + bảng chi phí vận hành (dưới) |
| House filter | **Multi-select** — chọn nhiều nhà, cards tổng hợp cross-house |
| Tạo tháng mới | Manager bấm nút, lấy dữ liệu từ **tháng gần nhất** |
| Tạo nhà mới | **Tự động seed** bản ghi chi phí mặc định (6 loại, amount = 0) |
| Edit UX | **Inline editing** trong bảng |
| Dashboard | Thêm card **lợi nhuận ròng** |

---

## Proposed Changes

### Database — 2 bảng mới

#### [NEW] [000009_create_house_costs_table.up.sql](file:///media/minhchu1336/Data/quanly-phongtro/migrations/000009_create_house_costs_table.up.sql)

**Bảng `house_costs`** — chi phí vận hành hàng tháng per house:

```sql
CREATE TABLE IF NOT EXISTS house_costs (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    house_id        UUID            NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
    period          VARCHAR(7)      NOT NULL,  -- yyyy-mm

    -- 6 loại chi phí chính (cột cố định)
    rent            DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền thuê nguyên căn
    electricity     DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền điện (biến đổi)
    water           DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền nước (biến đổi)
    wifi            DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền wifi
    cleaning        DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền vệ sinh/rác
    maintenance     DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền bảo trì (biến đổi)

    -- Chi phí tùy chỉnh (user tự thêm)
    extra_costs     JSONB           NOT NULL DEFAULT '[]',
    -- Format: [{"name": "Tiền thấm", "amount": 500000}, ...]

    note            TEXT            DEFAULT '',
    total_cost      DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tổng cộng (auto-calculated)
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_house_costs_house_period UNIQUE (house_id, period)
);

CREATE INDEX IF NOT EXISTS idx_house_costs_house_id ON house_costs(house_id);
CREATE INDEX IF NOT EXISTS idx_house_costs_period ON house_costs(house_id, period);
```

#### [NEW] [000010_create_house_revenue_summaries_table.up.sql](file:///media/minhchu1336/Data/quanly-phongtro/migrations/000010_create_house_revenue_summaries_table.up.sql)

**Bảng `house_revenue_summaries`** — tổng hợp lợi nhuận per house+period:

```sql
CREATE TABLE IF NOT EXISTS house_revenue_summaries (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    house_id        UUID            NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
    period          VARCHAR(7)      NOT NULL,

    total_revenue   DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- SUM(invoices.total_amount) WHERE PAID
    total_cost      DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- FROM house_costs.total_cost
    profit          DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- revenue - cost
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_revenue_summary_house_period UNIQUE (house_id, period)
);

CREATE INDEX IF NOT EXISTS idx_revenue_summary_house_period ON house_revenue_summaries(house_id, period);
```

#### [NEW] Down migrations cho cả 2 bảng

---

### Backend — Model

#### [NEW] [operating_cost.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/model/operating_cost.go)

```go
// ExtraCost represents a custom cost item in the JSONB field
type ExtraCost struct {
    Name   string  `json:"name"`
    Amount float64 `json:"amount"`
}

// HouseCost stores monthly operating costs for a house
type HouseCost struct {
    ID          string      `json:"id"`
    HouseID     string      `json:"house_id"`
    Period      string      `json:"period"`
    Rent        float64     `json:"rent"`
    Electricity float64     `json:"electricity"`
    Water       float64     `json:"water"`
    Wifi        float64     `json:"wifi"`
    Cleaning    float64     `json:"cleaning"`
    Maintenance float64     `json:"maintenance"`
    ExtraCosts  []ExtraCost `json:"extra_costs"`
    Note        string      `json:"note"`
    TotalCost   float64     `json:"total_cost"`
    CreatedAt   time.Time   `json:"created_at"`
    UpdatedAt   time.Time   `json:"updated_at"`
}

// HouseRevenueSummary stores aggregated profit per house per period
type HouseRevenueSummary struct {
    ID           string    `json:"id"`
    HouseID      string    `json:"house_id"`
    Period       string    `json:"period"`
    TotalRevenue float64   `json:"total_revenue"`
    TotalCost    float64   `json:"total_cost"`
    Profit       float64   `json:"profit"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// HouseCostRepository defines DB operations for operating costs
type HouseCostRepository interface {
    Create(ctx, cost *HouseCost) error
    GetByHouseAndPeriod(ctx, houseID, period string) (*HouseCost, error)
    GetLatestByHouseID(ctx, houseID string) (*HouseCost, error)
    ListByHouseIDs(ctx, houseIDs []string, period string) ([]HouseCost, error)
    Update(ctx, cost *HouseCost) error
    Delete(ctx, id, houseID string) error
}

// RevenueSummaryRepository defines DB operations for revenue summaries
type RevenueSummaryRepository interface {
    Upsert(ctx, summary *HouseRevenueSummary) error
    GetByHouseAndPeriod(ctx, houseID, period string) (*HouseRevenueSummary, error)
    ListByHouseIDs(ctx, houseIDs []string, period string) ([]HouseRevenueSummary, error)
    CalculateRevenue(ctx, houseID, period string) (float64, error)  // SUM invoices PAID
}
```

---

### Backend — Repository

#### [NEW] [house_cost_repository.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/repository/house_cost_repository.go)

Bun ORM — pattern giống `invoice_repository.go`:
- `Create` — insert, returning id + created_at
- `GetByHouseAndPeriod` — select with unique constraint
- `GetLatestByHouseID` — `ORDER BY period DESC LIMIT 1`
- `ListByHouseIDs` — `WHERE house_id IN (?) AND period = ?`
- `Update` — update amount fields, ownership check via houses.manager_id
- `Delete` — soft delete hoặc hard delete

#### [NEW] [revenue_summary_repository.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/repository/revenue_summary_repository.go)

- `Upsert` — `ON CONFLICT (house_id, period) DO UPDATE`
- `GetByHouseAndPeriod` — single record
- `ListByHouseIDs` — multi-house, single period
- `CalculateRevenue` — `SELECT SUM(total_amount) FROM invoices JOIN rooms JOIN houses WHERE PAID AND period = ?`

---

### Backend — Service

#### [NEW] [house_cost_service.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/house_cost_service.go)

```go
type HouseCostService interface {
    // Chi phí vận hành
    CreateMonthlyCost(ctx, managerID, houseID, period string) (*model.HouseCost, error)
    GetMonthlyCost(ctx, managerID, houseID, period string) (*model.HouseCost, error)
    UpdateMonthlyCost(ctx, managerID string, input UpdateCostInput) error

    // Revenue summaries
    GetRevenueSummaries(ctx, managerID string, houseIDs []string, period string) ([]model.HouseRevenueSummary, error)
}
```

**`CreateMonthlyCost` logic**:
1. Verify house ownership
2. Check duplicate (house_id + period)
3. Lấy bản ghi tháng gần nhất (`GetLatestByHouseID`)
4. Tạo record mới:
   - Có tháng trước → copy rent, wifi, cleaning từ tháng trước. electricity, water, maintenance = 0
   - Không có tháng trước → tất cả = 0 (bản ghi seed)
5. Tính `total_cost` = sum all fields + sum extra_costs
6. Insert + **gửi event vào channel** để recalculate summary

**`UpdateMonthlyCost` logic**:
1. Verify ownership
2. Update amounts + recalculate total_cost
3. **Gửi event vào channel** → recalculate summary

---

### Backend — Revenue Summary Worker

#### [NEW] [revenue_worker.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/revenue_worker.go)

Goroutine-based async worker:

```go
type RevenueSummaryEvent struct {
    HouseID string
    Period  string
}

type RevenueWorker struct {
    eventChannel chan RevenueSummaryEvent
    summaryRepo  model.RevenueSummaryRepository
    costRepo     model.HouseCostRepository
}

// Start spawns a background goroutine that listens for events
func (w *RevenueWorker) Start()

// Enqueue sends a recalculation event (non-blocking)
func (w *RevenueWorker) Enqueue(houseID, period string)

// processEvent recalculates and upserts the summary
func (w *RevenueWorker) processEvent(event)
```

Worker logic:
1. Lấy `total_revenue` = SUM invoices PAID cho house+period
2. Lấy `total_cost` từ house_costs cho house+period
3. `profit` = revenue - cost
4. Upsert vào `house_revenue_summaries`

---

### Backend — Integration (trigger events)

#### [MODIFY] [invoice_service.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/invoice_service.go)

Thêm `revenueWorker` dependency. Sau khi:
- `CreateInvoice` → `worker.Enqueue(houseID, period)`
- `PayInvoice` / `UnpayInvoice` → `worker.Enqueue(houseID, period)`
- `DeleteInvoice` → `worker.Enqueue(houseID, period)`

#### [MODIFY] [house_service.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/house_service.go)

Thêm `houseCostRepo` dependency. Trong `CreateHouse`:
- Sau khi tạo house thành công → tạo bản ghi `house_costs` seed cho kỳ hiện tại (6 loại, amount = 0)

---

### Backend — Handler

#### [NEW] [house_cost_handler.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/handler/house_cost_handler.go)

| Method | Endpoint | Handler | Description |
|---|---|---|---|
| `POST` | `/api/v1/house-cost` | CreateMonthlyCost | Tạo chi phí tháng mới |
| `GET` | `/api/v1/house-cost?house_id=&period=` | GetMonthlyCost | Lấy chi phí 1 nhà 1 kỳ |
| `PATCH` | `/api/v1/house-cost/{id}` | UpdateMonthlyCost | Cập nhật chi phí (inline edit) |
| `GET` | `/api/v1/revenue-summary?house_ids=&period=` | GetRevenueSummaries | Tổng thu/chi/lợi nhuận (multi-house) |

---

### Backend — Router & DI

#### [MODIFY] [router.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/router/router.go)

- Thêm `houseCostHandler *handler.HouseCostHandler` vào `New()`
- Thêm route group:
```go
r.Route("/api/v1/house-cost", func(r chi.Router) {
    r.Use(authMiddleware(tokenProvider))
    r.Use(requireRole("MANAGER"))
    r.Post("/", houseCostHandler.CreateMonthlyCost)
    r.Get("/", houseCostHandler.GetMonthlyCost)
    r.Patch("/{id}", houseCostHandler.UpdateMonthlyCost)
})
r.Route("/api/v1/revenue-summary", func(r chi.Router) {
    r.Use(authMiddleware(tokenProvider))
    r.Use(requireRole("MANAGER"))
    r.Get("/", houseCostHandler.GetRevenueSummaries)
})
```

#### [MODIFY] [main.go](file:///media/minhchu1336/Data/quanly-phongtro/cmd/api/main.go)

```go
// New repositories
houseCostRepo := repository.NewHouseCostRepository(sqlDB)
revenueSummaryRepo := repository.NewRevenueSummaryRepository(sqlDB)

// Revenue worker (goroutine + channel)
revenueWorker := service.NewRevenueWorker(revenueSummaryRepo, houseCostRepo)
revenueWorker.Start()
defer revenueWorker.Stop()

// Updated services (inject worker)
houseCostService := service.NewHouseCostService(houseCostRepo, houseRepo, revenueWorker)
invoiceService := service.NewInvoiceService(invoiceRepo, roomRepo, houseRepo, tenantRepo, revenueWorker)
houseService := service.NewHouseServiceImpl(houseRepo, houseCostRepo)  // inject costRepo for seed

// Handler
houseCostHandler := httpHandler.NewHouseCostHandler(houseCostService)

// Router
router := httpRouter.New(..., houseCostHandler)
```

---

### Frontend — API Layer

#### [NEW] [houseCost.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/api/houseCost.tsx)

```typescript
export interface ExtraCost {
  name: string;
  amount: number;
}

export interface HouseCost {
  id: string;
  house_id: string;
  period: string;
  rent: number;
  electricity: number;
  water: number;
  wifi: number;
  cleaning: number;
  maintenance: number;
  extra_costs: ExtraCost[];
  note: string;
  total_cost: number;
  created_at: string;
  updated_at: string;
}

export interface RevenueSummary {
  house_id: string;
  period: string;
  total_revenue: number;
  total_cost: number;
  profit: number;
}

// API functions
export const createMonthlyCost = async (houseId: string, period: string) => ...
export const getMonthlyCost = async (houseId: string, period: string) => ...
export const updateMonthlyCost = async (id: string, payload: Partial<HouseCost>) => ...
export const getRevenueSummaries = async (houseIds: string[], period: string) => ...
```

---

### Frontend — Zustand Store

#### [NEW] [houseCostData.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/data/houseCostData.tsx)

State: `costs` (map by houseId), `summaries`, `isLoading`, `selectedHouseIds`, `period`

Actions: `fetchCost()`, `createMonthlyCost()`, `updateCost()`, `fetchSummaries()`, `setSelectedHouseIds()`, `setPeriod()`

---

### Frontend — Tab Navigation

#### [MODIFY] [selectedData.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/data/selectedData.tsx)

- `TabType` thêm `'revenue'`

#### [MODIFY] [Sidebar.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/components/home/sidebar/Sidebar.tsx)

- Thêm `SidebarItem` icon `Wallet` (lucide), label "Doanh thu", ngay **dưới** tab "Hóa đơn"
- Cập nhật `handleTabClick` union type

#### [MODIFY] [MobileNav.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/components/home/sidebar/MobileNav.tsx)

- Thêm tab `{ id: 'revenue', icon: Wallet, label: 'Doanh thu' }`

#### [MODIFY] [Home.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/pages/Home.tsx)

- Import + render `RevenueView` khi `activeTab === 'revenue'`

---

### Frontend — Main View

#### [NEW] [RevenueView.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/components/home/RevenueView.tsx)

Layout:

```
┌─────────────────────────────────────────────────────────┐
│ Header: "Doanh thu"                   [Tạo chi phí mới] │
├─────────────────────────────────────────────────────────┤
│ Filters: [Multi-select Nhà ▼]  [Kỳ: month picker]      │
├─────────────────────────────────────────────────────────┤
│ ┌──────────┐  ┌──────────┐  ┌──────────┐               │
│ │💰 Tổng thu│  │💸 Tổng chi│  │📊 Lợi nhuận│             │
│ │12,500,000│  │ 8,200,000│  │ 4,300,000│               │
│ └──────────┘  └──────────┘  └──────────┘               │
├─────────────────────────────────────────────────────────┤
│ Bảng chi phí vận hành (hiện khi chọn 1 nhà cụ thể):    │
│ ┌────────────┬────────┬──────────────┬──────────┐       │
│ │ Tên        │ Loại   │ Số tiền      │ Ghi chú  │       │
│ ├────────────┼────────┼──────────────┼──────────┤       │
│ │ Thuê nhà   │ CĐ     │ [5,000,000]  │ [      ] │       │
│ │ Điện       │ BĐ     │ [2,100,000]  │ [      ] │       │
│ │ Nước       │ BĐ     │ [  350,000]  │ [      ] │       │
│ │ Wifi       │ CĐ     │ [  300,000]  │ [      ] │       │
│ │ Vệ sinh    │ CĐ     │ [  200,000]  │ [      ] │       │
│ │ Bảo trì    │ BĐ     │ [        0]  │ [      ] │       │
│ │ ── Thêm ── │        │              │          │       │
│ │ Tiền thấm  │ TC     │ [  500,000]  │ [Tầng 2] │ [🗑]  │
│ ├────────────┴────────┼──────────────┼──────────┤       │
│ │ TỔNG CỘNG           │  8,450,000   │          │       │
│ └─────────────────────┴──────────────┴──────────┘       │
│                                        [💾 Lưu thay đổi] │
└─────────────────────────────────────────────────────────┘
```

Tính năng:
- **Multi-select house** — summary cards tổng hợp nhiều nhà
- **Bảng chi phí** — chỉ hiện khi chọn **đúng 1 nhà** (vì mỗi nhà có bản ghi riêng)
- **Inline editing** — click vào ô số tiền/ghi chú để sửa, nút "Lưu thay đổi" xuất hiện khi có thay đổi
- **Thêm chi phí tùy chỉnh** — nút "Thêm" cuối bảng → thêm row vào extra_costs
- **Nút "Tạo chi phí tháng mới"** — tạo record mới, copy dữ liệu từ tháng trước

---

### Frontend — Dashboard Integration

#### [MODIFY] [DashboardView.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/components/home/DashboardView.tsx)

Thêm 1 card mới trong grid 4-column stats:
- **📊 Lợi nhuận ròng** — tổng hợp tất cả houses trong kỳ hiện tại
- Sub-value: "Thu: X | Chi: Y"
- Click → navigate to `revenue` tab

---

### Documentation

#### [MODIFY] [database_structure.md](file:///media/minhchu1336/Data/quanly-phongtro/Documents/database_structure.md)

Thêm mô tả bảng `house_costs` và `house_revenue_summaries`.

#### [MODIFY] [api_specifications.md](file:///media/minhchu1336/Data/quanly-phongtro/Documents/api_specifications.md)

Thêm section "7. House Cost & Revenue APIs".

---

## Verification Plan

### Automated Tests
- `go test ./internal/repository/ -run TestHouseCost -v`
- `go test ./internal/repository/ -run TestRevenueSummary -v`
- `go test ./internal/service/ -run TestHouseCost -v`
- `go test ./internal/service/ -run TestRevenueWorker -v`
- `go test ./internal/handler/ -run TestHouseCost -v`

### Manual Verification
1. Chạy migration `000009` + `000010`
2. Tạo nhà mới → verify auto-seed `house_costs` record với 6 loại, amount = 0
3. Vào tab Doanh thu → chọn nhà → verify bảng chi phí hiện đúng
4. Inline edit số tiền → Save → verify update thành công
5. Thêm chi phí tùy chỉnh (extra_costs) → verify JSONB lưu đúng
6. Tạo chi phí tháng mới → verify copy từ tháng trước
7. Thanh toán 1 hoá đơn → verify `house_revenue_summaries` được cập nhật async
8. Multi-select nhà → verify cards tổng hợp đúng
9. Kiểm tra Dashboard card lợi nhuận
10. `pnpm build` → no TypeScript errors
