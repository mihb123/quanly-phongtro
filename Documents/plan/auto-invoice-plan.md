# Tự động tạo hóa đơn qua Zalo Chat

## Tóm tắt

Thêm tính năng tự động tạo/cập nhật hóa đơn (invoice) thông qua tin nhắn Zalo Bot, hỗ trợ cả group chat và private chat cho cả manager và tenant.

---

## Tổng quan quyết định thiết kế

| Quyết định | Kết quả |
|---|---|
| Period | Tự động = tháng tiếp theo của invoice cuối. Không có invoice trước → hỏi user chọn tháng |
| Dữ liệu thiếu | Tạo invoice thật với chỉ số thiếu = chỉ số cũ (usage = 0), nhắn user bổ sung |
| Old index | Tự động lấy từ invoice trước (logic có sẵn trong `CreateInvoice`) |
| Match tên phòng | Case-insensitive, bỏ dấu, viết tắt (`P101` → `Phòng 101`) |
| Match tên nhà | Dùng `house_code` (field mới) thay vì tên nhà |
| Ghi đè | Hỏi xác nhận trước khi ghi đè, lưu pending vào DB |
| Pending state | Bảng `pending_invoice_updates` trong PostgreSQL, không hết hạn |
| Vehicle/Other/Discount | Lấy từ invoice trước hoặc mặc định = 0 |
| Gửi ảnh invoice | Ưu tiên group → chỉ gửi group. Không group → gửi riêng cho **tất cả tenant linked** + manager |
| Quyền trong group | Ai cũng có thể gửi cú pháp (không cần xác thực thêm) |
| Tổ chức code | 2 file mới: `zalo_command_parser.go` + `zalo_invoice_command_service.go` |

---

## Cú pháp tin nhắn

### 1. Group chat / Tenant private chat

```
#dien <chỉ số mới>
#nuoc <chỉ số mới>
```
Lưu ý, nếu dùng điện/nước là fixed thì không cần nhập chỉ số mới. ví dụ tiền nước là 100k thì không cần nhập gì cả, bot sẽ tự hiểu là tiền nước là 100k, chỉ cần nhập tiền điện. Cả điện/nước là fixed thì cứ tự động tạo hóa đơn vào ngày 1 hàng tháng và gửi cho tenant, manager.

Cần handle cả trường hợp user nhắn tiếng việt, ví dụ `#Điện #điện #nước #Nước`

Ví dụ:
- `#dien 750` → số điện mới là 750
- `#nuoc 150` → số nước mới là 150

### 2. Manager private chat (batch nhiều phòng)

```
#dien <house_code>
<room_name> <chỉ số mới>
<room_name> <chỉ số mới>
...
```

Ví dụ:
```
#dien 679qt
P101 750
P201 900
```

Tương tự cho nước:
```
#nuoc 679qt
P101 150
P201 200
```

### 3. Xác nhận / Hủy (khi bot hỏi)

- `ok` → xác nhận ghi đè
- `huy` → hủy thao tác
- `5` hoặc `6` → chọn tháng khi bot hỏi period

---

## Database Schema Changes

### Migration 1: Thêm `house_code` vào bảng `houses`

```sql
ALTER TABLE houses ADD COLUMN house_code VARCHAR(12);
CREATE UNIQUE INDEX uq_houses_manager_code ON houses (manager_id, house_code);
```

**Ràng buộc:**
- Unique per manager (2 manager khác nhau có thể trùng code)
- Chỉ chứa `[a-zA-Z0-9_-]`, không khoảng trắng, không ký tự đặc biệt
- Max 12 ký tự
- Frontend tự generate suggestion từ tên nhà (bỏ dấu, lowercase, viết tắt), user có thể sửa
- Validate cả frontend và backend

### Migration 2: Bảng `pending_invoice_updates`

```sql
CREATE TABLE pending_invoice_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manager_id UUID NOT NULL REFERENCES users(id),
    chat_id VARCHAR(255) NOT NULL,
    is_group_chat BOOLEAN NOT NULL DEFAULT false,
    room_id UUID REFERENCES rooms(id),
    action_type VARCHAR(50) NOT NULL,
    pending_data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_pending_updates_chat ON pending_invoice_updates (manager_id, chat_id);
```

**`action_type` values:**
- `CONFIRM_OVERWRITE` – chờ user xác nhận ghi đè chỉ số
- `AWAIT_PERIOD` – chờ user chọn tháng
- `AWAIT_UTILITY` – chờ user bổ sung chỉ số còn thiếu (informational, không block)

**`pending_data` schema (JSONB):**
```json
{
  "utility_type": "dien",
  "new_index": 750,
  "period": "2026-06",
  "room_id": "uuid...",
  "house_code": "679qt",
  "entries": [
    {"room_name": "P101", "new_index": 750},
    {"room_name": "P201", "new_index": 900}
  ]
}
```

---

## Backend Changes

### Component 1: Model Layer

#### [MODIFY] `internal/model/house.go`

```go
type House struct {
    // ... existing fields ...
    HouseCode string `json:"house_code"`  // NEW
}

type UpdateHouseParams struct {
    // ... existing fields ...
    HouseCode string  // NEW
}
```

#### [NEW] `internal/model/pending_invoice_update.go`

```go
type PendingInvoiceUpdate struct {
    ID          string                 `json:"id"`
    ManagerID   string                 `json:"manager_id"`
    ChatID      string                 `json:"chat_id"`
    IsGroupChat bool                   `json:"is_group_chat"`
    RoomID      *string                `json:"room_id,omitempty"`
    ActionType  string                 `json:"action_type"`
    PendingData map[string]interface{} `json:"pending_data"`
    CreatedAt   time.Time              `json:"created_at"`
}

type PendingInvoiceUpdateRepository interface {
    Create(ctx context.Context, pending *PendingInvoiceUpdate) error
    GetByChatID(ctx context.Context, managerID, chatID string) (*PendingInvoiceUpdate, error)
    DeleteByChatID(ctx context.Context, managerID, chatID string) error
    DeleteByID(ctx context.Context, id string) error
}
```

#### [MODIFY] `internal/model/house.go` – HouseRepository interface

```go
type HouseRepository interface {
    // ... existing methods ...
    GetHouseByCode(ctx context.Context, managerID, houseCode string) (*House, error) // NEW
}
```

---

### Component 2: Repository Layer

#### [MODIFY] `internal/repository/house_repository.go`

- Thêm `house_code` vào tất cả query: `CreateHouse`, `UpdateHouse`, `GetByID`, `ListHouseByManagerID`
- Thêm method:

```go
// GetHouseByCode finds a house by its short code within a manager's houses.
func (r *HouseRepository) GetHouseByCode(ctx context.Context, managerID, houseCode string) (*House, error)
```

#### [NEW] `internal/repository/pending_invoice_update_repository.go`

```go
type PendingInvoiceUpdateRepository struct { db *bun.DB }

func (r *PendingInvoiceUpdateRepository) Create(ctx, pending) error
func (r *PendingInvoiceUpdateRepository) GetByChatID(ctx, managerID, chatID) (*PendingInvoiceUpdate, error)
func (r *PendingInvoiceUpdateRepository) DeleteByChatID(ctx, managerID, chatID) error
func (r *PendingInvoiceUpdateRepository) DeleteByID(ctx, id) error
```

---

### Component 3: Zalo Command Parser

#### [NEW] `internal/service/zalo_command_parser.go`

Parse cú pháp tin nhắn thành structured command.

```go
type CommandType int

const (
    CommandUnknown       CommandType = iota
    CommandUtilitySingle             // #dien 750 hoặc #nuoc 150
    CommandUtilityBatch              // #dien 679qt\nP101 750\nP201 900
    CommandConfirm                   // ok
    CommandCancel                    // huy
    CommandPeriodSelect              // 5, 6 (chọn tháng)
)

type ParsedCommand struct {
    Type        CommandType
    UtilityType string            // "dien" hoặc "nuoc"
    HouseCode   string            // chỉ có khi batch
    Entries     []RoomUtilityEntry
    PeriodMonth int
}

type RoomUtilityEntry struct {
    RoomName string
    NewIndex int
}

// ParseCommand parses a raw chat message into a structured invoice command.
func ParseCommand(text string) *ParsedCommand

// NormalizeRoomName strips Vietnamese diacritics, "phòng"/"phong"/"p" prefix,
// and returns the numeric portion for fuzzy matching.
func NormalizeRoomName(name string) string

// MatchRoom finds the best-matching room from a list by normalized name comparison.
func MatchRoom(input string, rooms []model.Room) (*model.Room, error)
```

**Logic normalize tên phòng:**

| Input | Normalized | Match target |
|---|---|---|
| `P101` | `101` | `Phòng 101` → `101` ✓ |
| `Phòng 101` | `101` | `P101` → `101` ✓ |
| `101` | `101` | `Phòng 101` → `101` ✓ |
| `p201` | `201` | `P201` → `201` ✓ |

---

### Component 4: Zalo Invoice Command Service

#### [NEW] `internal/service/zalo_invoice_command_service.go`

```go
type ZaloInvoiceCommandService interface {
    // HandleInvoiceCommand processes a parsed invoice command from a Zalo webhook message.
    HandleInvoiceCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext) error
}

type zaloInvoiceCommandServiceImpl struct {
    invoiceService InvoiceService
    roomRepo       model.RoomRepository
    houseRepo      model.HouseRepository
    tenantRepo     model.TenantRepository
    userRepo       model.UserRepository
    pendingRepo    model.PendingInvoiceUpdateRepository
    zaloClient     ZaloClient
    imageService   ImageService
    encryptionKey  []byte
    publicBaseURL  string
}
```

**Flow xử lý chính:**

```
Tin nhắn webhook
  │
  ├─ Parse command → Không phải command → Bỏ qua
  │
  ├─ Check pending state cho chat_id
  │   ├─ Có pending CONFIRM_OVERWRITE
  │   │   ├─ User nhắn "ok"  → Thực hiện ghi đè → Xóa pending
  │   │   └─ User nhắn "huy" → Xóa pending → Thông báo hủy
  │   │
  │   └─ Có pending AWAIT_PERIOD
  │       ├─ User nhắn số tháng → Tạo invoice với period đã chọn → Xóa pending
  │       └─ Sai format → Nhắn lại cú pháp
  │
  ├─ #dien / #nuoc (SINGLE - group hoặc tenant private)
  │   ├─ Group → Tìm room qua GetRoomByGroupChatID(chatID)
  │   └─ Private tenant → Tìm user qua GetByZaloUserID(senderID) → GetFirstTenantByUserID → room
  │   │
  │   ├─ Xác định period
  │   │   ├─ Có invoice trước → period = next month
  │   │   └─ Không có → Tạo pending AWAIT_PERIOD, hỏi user
  │   │
  │   ├─ Check ghi đè
  │   │   ├─ Invoice kỳ này đã có chỉ số điện/nước mới (khác old) → Tạo pending CONFIRM_OVERWRITE
  │   │   └─ Chưa có → Tiếp tục
  │   │
  │   ├─ Gọi CreateInvoice (upsert)
  │   │
  │   └─ Check đủ thông tin
  │       ├─ Đủ (hoặc billing type = FIXED) → Gửi ảnh invoice
  │       └─ Thiếu → Nhắn user bổ sung
  │
  └─ #dien / #nuoc (BATCH - manager private)
      ├─ Parse house_code → GetHouseByCode
      ├─ List rooms cho house
      ├─ Loop từng entry:
      │   ├─ MatchRoom(entry.RoomName, rooms)
      │   ├─ Xác định period + check ghi đè (tương tự single)
      │   └─ CreateInvoice
      └─ Gửi summary kết quả + ảnh cho từng phòng đủ thông tin
```

---

### Component 5: Logic gửi ảnh invoice (SendInvoiceToZalo)

#### [MODIFY] `internal/service/zalo_service.go` – `SendInvoiceToZalo`

**Logic mới (thay thế logic cũ):**

```
SendInvoiceToZalo(managerID, invoiceID)
  │
  ├─ Lấy invoice, room, generate ảnh
  │
  ├─ Room có group_chat_id?
  │   ├─ CÓ → Gửi ảnh vào group → RETURN (không gửi private)
  │   │
  │   └─ KHÔNG → Gửi tin nhắn riêng:
  │       ├─ Gửi cho manager (nếu manager đã link Zalo)
  │       └─ Gửi cho TẤT CẢ tenant đã link Zalo trong phòng
  │           (VD: 3 tenant linked → gửi 3 tin + 1 tin manager)
  │
  └─ Nếu không có group VÀ không có ai linked → return error
```

**So sánh logic cũ vs mới:**

| Trường hợp | Logic cũ | Logic mới |
|---|---|---|
| Có group + 3 tenant linked | Gửi group + 3 private | Chỉ gửi group |
| Có group + 0 tenant linked | Gửi group | Chỉ gửi group |
| Không group + 3 tenant linked | Gửi 3 private | Gửi 3 private + manager |
| Không group + 0 tenant linked | Error | Error |
| Không group + manager linked + 2 tenant | Gửi 2 private | Gửi manager + 2 private |

---

### Component 6: Webhook Integration

#### [MODIFY] `internal/service/zalo_service.go` – `HandleWebhook`

Thêm đoạn sau vào `HandleWebhook`, **trước** logic xử lý `botoi` và image:

```go
// Check if message is an invoice command (#dien, #nuoc, ok, huy, period select)
if s.invoiceCommandService != nil {
    if isInvoiceCommand(webhookCtx.text) || s.hasPendingState(ctx, managerID, webhookCtx) {
        err := s.invoiceCommandService.HandleInvoiceCommand(ctx, managerID, webhookCtx)
        if err != nil {
            fmt.Printf("Invoice command error: %v\n", err)
        }
        return nil  // consumed by invoice command handler
    }
}
```

#### [MODIFY] `internal/service/zalo_service.go` – `ZaloService` interface & struct

```go
type zaloServiceImpl struct {
    // ... existing fields ...
    invoiceCommandService ZaloInvoiceCommandService  // NEW
}
```

#### [MODIFY] `cmd/api/main.go`

Wire up new dependencies:
- `PendingInvoiceUpdateRepository`
- `ZaloInvoiceCommandService`
- Inject into `ZaloService`

---

### Component 7: House Code – Backend

#### [MODIFY] `internal/handler/house_handler.go`

- Thêm `HouseCode string` vào `createHouseRequest` và `updateHouseRequest`
- Validate: `^[a-zA-Z0-9_-]+$`, max 12 chars, không trống

#### [MODIFY] `internal/service/house_service.go`

- Pass `HouseCode` qua create/update params

---

### Component 8: House Code – Frontend

#### [MODIFY] `frontend/src/api/house.tsx`

```typescript
export interface House {
  // ... existing fields ...
  house_code: string;  // NEW
}

export interface CreateHousePayload {
  // ... existing fields ...
  house_code: string;  // NEW
}
```

#### [MODIFY] `frontend/src/components/home/modals/CreateHouseModal.tsx`

- Thêm input field `Mã nhà (House Code)`
- Auto-generate suggestion khi user nhập tên nhà:
  - `"679 Quang Trung"` → suggestion `"679qt"`
  - `"Nhà trọ Cầu Giấy"` → suggestion `"ntcg"`
- Validate realtime: regex `^[a-zA-Z0-9_-]+$`
- Error message nếu trùng code (từ API response)

#### [MODIFY] `frontend/src/components/home/modals/EditHouseModal.tsx`

- Hiển thị và cho phép sửa `house_code`
- Cùng validation logic

---

## Flow chi tiết từng kịch bản

### Kịch bản 1: Tenant gửi `#dien 750` trong group chat

```
1. Webhook nhận tin nhắn từ group
2. Parse: CommandUtilitySingle, UtilityType="dien", NewIndex=750
3. Tìm room qua GetRoomByGroupChatID(chatID) → Room "Phòng 101"
4. Tìm latest invoice cho room → period "2026-05"
5. Period mới = "2026-06"
6. Check invoice "2026-06" đã tồn tại?
   - Chưa → Tạo mới
7. House dùng water billing type = "USAGE"
   → Invoice tạo với new_water_index = old_water_index (usage = 0)
8. Nhắn vào group: "✅ Đã ghi nhận số điện mới: 750 cho Phòng 101 (tháng 06/2026).
   Vui lòng bổ sung số nước bằng cú pháp: #nuoc <số mới>"
```

### Kịch bản 2: Tiếp theo, ai đó gửi `#nuoc 150` trong cùng group

```
1. Parse: CommandUtilitySingle, UtilityType="nuoc", NewIndex=150
2. Tìm room qua group → Room "Phòng 101"
3. Invoice "2026-06" đã tồn tại (vừa tạo ở bước trước)
4. Update: new_water_index = 150
5. Bây giờ đủ cả điện + nước → Generate ảnh invoice
6. Gửi ảnh vào group (vì có group chat)
7. Nhắn: "📄 Hóa đơn tháng 06/2026 cho Phòng 101. Tổng tiền: 1,500,000đ"
```

### Kịch bản 3: Manager gửi batch qua private chat

```
Manager nhắn:
  #dien 679qt
  P101 750
  P201 900

1. Parse: CommandUtilityBatch, HouseCode="679qt"
2. GetHouseByCode(managerID, "679qt") → House "679 Quang Trung"
3. ListAllRoomsByHouseID → [Phòng 101, Phòng 201, ...]
4. Entry "P101":
   - MatchRoom("P101", rooms) → Phòng 101
   - Tạo/update invoice cho Phòng 101 với new_elec = 750
5. Entry "P201":
   - MatchRoom("P201", rooms) → Phòng 201
   - Tạo/update invoice cho Phòng 201 với new_elec = 900
6. Gửi summary cho manager:
   "✅ Đã cập nhật số điện cho nhà 679 Quang Trung:
   - Phòng 101: 750 (cũ: 680) → 70 số
   - Phòng 201: 900 (cũ: 820) → 80 số
   Bổ sung số nước bằng: #nuoc 679qt"
```

### Kịch bản 4: Ghi đè chỉ số

```
Tenant gửi: #dien 780  (đã có 750 từ trước cho cùng kỳ)

1. Parse: single, dien, 780
2. Tìm room → Phòng 101
3. Invoice "2026-06" đã có new_electricity_index = 750 (khác old)
4. Tạo pending CONFIRM_OVERWRITE:
   pending_data = {"utility_type": "dien", "new_index": 780, "old_value": 750}
5. Nhắn: "⚠️ Số điện tháng 06/2026 đã được ghi là 750.
   Bạn muốn cập nhật thành 780?
   Nhắn 'ok' để xác nhận, 'huy' để hủy."

Tenant nhắn: ok

6. Check pending → CONFIRM_OVERWRITE
7. Update invoice: new_electricity_index = 780, tính lại fee
8. Xóa pending
9. Nhắn: "✅ Đã cập nhật số điện thành 780."
10. Nếu đủ thông tin → gửi lại ảnh invoice
```

### Kịch bản 5: Không có invoice trước, hỏi period

```
Tenant gửi: #dien 750  (phòng chưa có invoice nào)

1. GetPreviousInvoice → ErrInvoiceNotFound
2. Ngày hiện tại: 2026-06-14
3. Tạo pending AWAIT_PERIOD:
   pending_data = {"utility_type": "dien", "new_index": 750, "options": [5, 6]}
4. Nhắn: "Phòng này chưa có hóa đơn trước đó.
   Bạn muốn tạo hóa đơn cho tháng nào?
   Nhắn '5' cho tháng 05/2026
   Nhắn '6' cho tháng 06/2026"

Tenant nhắn: 6

5. Check pending → AWAIT_PERIOD
6. Period = "2026-06"
7. Tạo invoice → xóa pending → nhắn xác nhận
```

### Kịch bản 6: Tenant gửi qua private chat (không có group)

```
Tenant nhắn riêng cho bot: #dien 750

1. Parse: single, dien, 750
2. GetByZaloUserID(senderID) → User (role=TENANT)
3. GetFirstTenantByUserID(managerID, userID) → Tenant ở Phòng 101
4. Tương tự xử lý như group, nhưng khi gửi ảnh:
   - Phòng 101 không có group_chat_id
   - → Gửi ảnh riêng cho tenant + manager
```

---

## File thay đổi tổng hợp

### Files mới (NEW)

| File | Mô tả |
|---|---|
| `internal/model/pending_invoice_update.go` | Model + Repository interface cho pending state |
| `internal/repository/pending_invoice_update_repository.go` | PostgreSQL implementation |
| `internal/service/zalo_command_parser.go` | Parse cú pháp tin nhắn + normalize tên phòng |
| `internal/service/zalo_invoice_command_service.go` | Business logic xử lý command tạo invoice |

### Files sửa đổi (MODIFY)

| File | Thay đổi |
|---|---|
| `internal/model/house.go` | Thêm `HouseCode` field + `GetHouseByCode` vào interface |
| `internal/repository/house_repository.go` | Thêm `house_code` vào queries + method `GetHouseByCode` |
| `internal/handler/house_handler.go` | Thêm `house_code` vào request structs + validation |
| `internal/service/house_service.go` | Pass `house_code` qua create/update |
| `internal/service/zalo_service.go` | Thêm invoice command delegation vào webhook + sửa `SendInvoiceToZalo` |
| `cmd/api/main.go` | Wire up `PendingInvoiceUpdateRepository` + `ZaloInvoiceCommandService` |
| `frontend/src/api/house.tsx` | Thêm `house_code` vào interfaces |
| `frontend/src/components/home/modals/CreateHouseModal.tsx` | Thêm house code input + auto-generate |
| `frontend/src/components/home/modals/EditHouseModal.tsx` | Thêm house code input |

---

## Verification Plan

### Automated Tests

#### 1. Unit test `zalo_command_parser.go`

```bash
go test ./internal/service/ -run TestParseCommand -v
```

Test cases:
- `#dien 750` → `CommandUtilitySingle`, `UtilityType="dien"`, `Entries=[{NewIndex:750}]`
- `#nuoc 150` → `CommandUtilitySingle`, `UtilityType="nuoc"`, `Entries=[{NewIndex:150}]`
- Multi-line batch → `CommandUtilityBatch` với `HouseCode` + `Entries`
- `ok` → `CommandConfirm`
- `huy` → `CommandCancel`
- `5` → `CommandPeriodSelect`, `PeriodMonth=5`
- `NormalizeRoomName("P101")` → `"101"`
- `NormalizeRoomName("Phòng 101")` → `"101"`
- `NormalizeRoomName("phong101")` → `"101"`
- `MatchRoom("P101", rooms)` tìm đúng phòng
- Ambiguous match → error

#### 2. Unit test `zalo_invoice_command_service.go`

```bash
go test ./internal/service/ -run TestInvoiceCommand -v
```

Test cases:
- Group chat: tạo invoice mới thành công
- Tenant private: tạo invoice thành công
- Manager batch: tạo nhiều invoice thành công
- Ghi đè → pending → confirm → update thành công
- Ghi đè → pending → cancel → hủy thành công
- Thiếu nước → nhắn bổ sung → bổ sung → gửi ảnh
- Hỏi period → user chọn → tạo invoice
- FIXED billing type → chỉ cần 1 loại chỉ số là đủ
- House code không tìm thấy → error message
- Room name không match → error message

#### 3. Unit test `SendInvoiceToZalo` (logic mới)

```bash
go test ./internal/service/ -run TestSendInvoiceToZalo -v
```

Test cases:
- Có group → chỉ gửi group, không gửi private
- Không group, 3 tenant linked + manager linked → gửi 4 tin
- Không group, 0 tenant linked → error

#### 4. Regression tests

```bash
go test ./internal/... -v
```

### Manual Verification

1. **Group chat flow**: Gửi `#dien 750` vào group → kiểm tra invoice tạo → gửi `#nuoc 150` → kiểm tra ảnh gửi qua group
2. **Ghi đè**: Gửi `#dien 780` → bot hỏi confirm → nhắn `ok` → kiểm tra update
3. **Manager batch**: Gửi tin nhắn nhiều dòng qua private → kiểm tra tạo nhiều invoice
4. **Tenant private**: Tenant gửi `#dien 750` qua private → kiểm tra bot nhận biết phòng
5. **Không có invoice trước**: Gửi command cho phòng mới → bot hỏi period → chọn tháng
6. **House code**: Tạo nhà mới từ web → kiểm tra house code auto-generate + validate
