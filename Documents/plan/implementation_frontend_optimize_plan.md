# Refactor Frontend: Zustand Data Stores + React Hook Form

Chuyển từ mô hình prop drilling sang Zustand data stores, chuyển form state sang React Hook Form + Zod (reuse pattern đã có), và chuyển modal ownership về đúng component sử dụng.

## Phân tích hiện trạng

### Vấn đề 1: Prop Drilling
- **`Home.tsx`** (314 dòng) — Chứa ~10 state modal + orchestrate 16 giá trị từ `useHomeData`, truyền xuống child components.
- **`HouseRoomsView`** — Nhận **11 props** (4 data + 6 callback + 1 config).

### Vấn đề 2: Quá nhiều local state — form state thủ công

| Component | Số `useState` | Trong đó form fields |
|---|---|---|
| `TenantRoomModal` | **15** | 9 form fields |
| `CreateHouseModal` | **10** | 8 form fields |
| `EditRoomModal` | **9** | 8 form fields |
| `Home.tsx` | **10** | 0 (toàn modal state) |

Dự án đã cài **react-hook-form@7.72 + @hookform/resolvers@5.2 + zod@3.24** và dùng cho Login/Register. Nhưng các form trong Home page vẫn dùng `useState` thủ công → không nhất quán, re-render mỗi keystroke.

---

## Proposed Changes

### Phase 1: Zustand Data Stores — folder `src/data/`

Tách thành **3 stores** theo entity, mỗi store quản lý CRUD riêng:

#### [NEW] [houseData.ts](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/data/houseData.ts)

```typescript
interface HouseDataState {
  houses: House[]
  fetchHouses: () => Promise<void>
  deleteHouse: (houseId: string) => Promise<boolean>
}
```

#### [NEW] [roomData.ts](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/data/roomData.ts)

```typescript
interface RoomDataState {
  rooms: Room[]
  roomPage: number
  setRoomPage: (page: number) => void
  fetchRooms: (houseId: string, page: number) => Promise<void>
  refreshCurrentRooms: () => Promise<void>
  deleteRoom: (roomId: string, houseId: string) => Promise<boolean>
}
// Export ROOMS_LIMIT = 25
```

`refreshCurrentRooms` đọc `selectedData.getState().selectedHouse` + `roomPage` hiện tại để refetch. Dùng `getState()` cross-store thay vì subscribe.

#### [NEW] [selectedData.ts](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/data/selectedData.ts)

```typescript
interface SelectedDataState {
  selectedHouse: House | null
  activeTab: 'dashboard' | 'house_rooms' | 'tenants'
  isHouseListOpen: boolean
  isSidebarCollapsed: boolean
  selectHouse: (house: House) => void
  setActiveTab: (tab: ...) => void
  setIsHouseListOpen: (open: boolean) => void
  setIsSidebarCollapsed: (collapsed: boolean) => void
}
// localStorage sync cho activeTab + selectedHouseId
```

---

### Phase 2: Modal Ownership — Chuyển modals về đúng component

#### Nguyên tắc
1. Modal render tại component mà **chỉ component đó** trigger nó.
2. Mỗi modal nhận tối thiểu props: `onClose` (bắt buộc). Payload (`room`) chỉ khi không lấy được từ store.
3. **`onSuccess` loại bỏ hoàn toàn** — modal tự gọi store actions + `onClose()`.

#### Phân bổ modal ownership

**`HouseRoomsView`** — tự quản lý 5 modals:

| Modal | Local state | Props modal nhận |
|---|---|---|
| `CreateRoomModal` | `showCreateRoom: boolean` | `onClose` (1) |
| `EditRoomModal` | `editRoom: Room \| null` | `room` + `onClose` (2) |
| `TenantRoomModal` | `tenantRoom: Room \| null` | `room` + `onClose` (2) |
| `QuickSetRoomPriceModal` | `showQuickSetPrice: boolean` | `onClose` (1) |
| `ConfirmModal` (xóa phòng) | `roomToDelete` + `isDeleting` | Generic props |

**`DashboardView`** — tự quản lý CreateHouseModal:

| Modal | Local state | Props modal nhận |
|---|---|---|
| `CreateHouseModal` | `showCreateHouse: boolean` | `onClose` (1) |

**`Home.tsx`** — chỉ giữ modals trigger từ **Sidebar**:

| Modal | Local state |
|---|---|
| `CreateHouseModal` | `showCreateHouse: boolean` |
| `ConfirmModal` (xóa nhà) | `houseToDelete` + `isDeleting` |
| Context menu | `contextMenu` |

---

### Phase 3: React Hook Form + Zod — Thay thế form useState

Reuse **pattern đã có** trong Login/Register:
- Zod schema inline trong component file
- `useForm({ resolver: zodResolver(schema) })`
- `{...register('fieldName')}` trên Input
- `formState: { errors }` cho validation messages
- `reset(values)` để load data khi edit
- `isLoading` giữ `useState` riêng (RHF không quản lý async submit loading)

#### [MODIFY] `TenantRoomModal.tsx` — 15 useState → ~5 useState

**Trước:** 9 form field useState + 6 non-form useState

**Sau:**
- **RHF quản lý** (0 useState): `fullName`, `phone`, `email`, `identityCard`, `startDate`
- **File upload giữ useState** (4): `cccdFiles`, `contractFiles`, `existingCccdPaths`, `existingContractPaths` — RHF không quản lý File objects tốt
- **View state giữ useState** (1): `selectedImageUrl`
- **Tách hook** `useTenantList`: `tenants`, `isLoading`, `isFetching`, `showAddForm`, `editingTenantId`

```typescript
const tenantSchema = z.object({
  fullName: z.string().min(1, 'Bắt buộc'),
  phone: z.string().min(1, 'Bắt buộc'),
  email: z.string().email().optional().or(z.literal('')),
  identityCard: z.string().min(1, 'Bắt buộc'),
  startDate: z.string(),
})

const { register, handleSubmit, reset, formState: { errors } } = useForm({
  resolver: zodResolver(tenantSchema),
  defaultValues: { fullName: '', phone: '', email: '', identityCard: '', startDate: today }
})
// Edit: reset({ fullName: tenant.full_name, phone: tenant.phone, ... })
```

#### [MODIFY] `CreateHouseModal.tsx` — 10 useState → 3 useState

**Trước:** 7 text field useState + floorCount + roomsPerFloor + isLoading

**Sau:**
- **RHF quản lý** (0 useState): `name`, `address`, `electricity`, `water`, `wifi`, `parking`, `service`
- **Giữ useState** (3): `floorCount` + `roomsPerFloor` (dynamic floor config, RHF không phù hợp với dynamic Record) + `isLoading`

```typescript
const houseSchema = z.object({
  name: z.string().min(1, 'Bắt buộc'),
  address: z.string().min(1, 'Bắt buộc'),
  electricity: z.string(),
  water: z.string(),
  wifi: z.string(),
  parking: z.string(),
  service: z.string(),
})
```

#### [MODIFY] `EditRoomModal.tsx` — 9 useState → 1 useState

**Trước:** 3 room info + 5 pricing useState + isLoading

**Sau:**
- **RHF quản lý** (0 useState): `name`, `price`, `maxTenants`, `electricity`, `water`, `wifi`, `parking`, `service`
- **Giữ useState** (1): `isLoading`

```typescript
const roomSchema = z.object({
  name: z.string().min(1, 'Bắt buộc'),
  price: z.string(),
  maxTenants: z.string(),
  electricity: z.string(),
  water: z.string(),
  wifi: z.string(),
  parking: z.string(),
  service: z.string(),
})
// Init: reset({ name: room.name, price: room.price?.toString(), ... })
```

#### [MODIFY] `CreateRoomModal.tsx` — 4 useState → 1 useState

**Sau:** RHF quản lý `name`, `price`, `maxTenants`. Giữ `isLoading` useState.

#### [NEW] [useTenantList.ts](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/hooks/useTenantList.ts)

Gom logic fetch/delete tenants + view state (non-form). Hook này tái sử dụng được nếu tương lai có component khác cần hiển thị danh sách tenant:

```typescript
interface UseTenantListReturn {
  tenants: Tenant[]
  isLoading: boolean
  isFetching: boolean
  showAddForm: boolean
  editingTenantId: string | null
  fetchTenants: (roomId: string) => Promise<void>
  handleDeleteTenant: (tenantId: string, roomId: string) => Promise<void>
  setShowAddForm: (show: boolean) => void
  setEditingTenantId: (id: string | null) => void
}
```

---

### Phase 4: Utilities

#### [NEW] [file.ts](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/utils/file.ts)

Chuyển 2 pure functions từ `TenantRoomModal`:
- `getFileName(path)` — trích tên file từ path
- `isImagePath(path)` — kiểm tra extension image

---

### Tổng hợp files

#### New files (5)

| File | Mô tả |
|---|---|
| `src/data/houseData.ts` | Zustand store — houses CRUD |
| `src/data/roomData.ts` | Zustand store — rooms CRUD + pagination |
| `src/data/selectedData.ts` | Zustand store — selection + navigation |
| `src/hooks/useTenantList.ts` | Fetch/delete + view state cho tenant list |
| `src/utils/file.ts` | Pure helpers: `getFileName`, `isImagePath` |

#### Modified files (10)

| File | Thay đổi chính |
|---|---|
| `Home.tsx` | Xóa `useHomeData` + 7 modal state. Giữ 3 state. Subscribe stores |
| `HouseRoomsView.tsx` | **11 props → 0 props.** Tự quản lý 5 modals. Data từ stores |
| `DashboardView.tsx` | **1 prop → 0 props.** Tự quản lý `CreateHouseModal` |
| `TenantsView.tsx` | **1 prop → 0 props.** `houses` từ store |
| `CreateHouseModal.tsx` | 2 props → **1 prop** (`onClose`). RHF + Zod cho 7 text fields |
| `CreateRoomModal.tsx` | 3 props → **1 prop** (`onClose`). RHF + Zod |
| `EditRoomModal.tsx` | 4 props → **2 props** (`room`, `onClose`). RHF + Zod cho 8 fields |
| `QuickSetRoomPriceModal.tsx` | 3 props → **1 prop** (`onClose`). `rooms` từ store |
| `TenantRoomModal.tsx` | 3 props → **2 props** (`room`, `onClose`). RHF cho 5 fields + `useTenantList` hook |
| `ConfirmModal.tsx` | Giữ nguyên (generic component) |

#### Deleted files (1)

| File | Lý do |
|---|---|
| `src/hooks/useHomeData.ts` | Logic chuyển sang 3 data stores |

---

### Kết quả tổng hợp

| Metric | Trước | Sau | Ghi chú |
|---|---|---|---|
| `Home.tsx` useState | 10 | 3 | Chỉ sidebar modal/context |
| `HouseRoomsView` props | 11 | **0** | Data từ store, modals local |
| `TenantRoomModal` useState | 15 | ~5 | RHF: 5 fields, file: 4 useState, view: 1 |
| `CreateHouseModal` useState | 10 | 3 | RHF: 7 fields, floor config: 2, loading: 1 |
| `EditRoomModal` useState | 9 | 1 | RHF: 8 fields, loading: 1 |
| `CreateRoomModal` useState | 4 | 1 | RHF: 3 fields, loading: 1 |
| Pattern consistency | RHF ở Login/Register, useState ở Home | **RHF everywhere** | Reuse existing pattern |
| Re-render on keystroke | Mọi form | **Không** (RHF uncontrolled) | Performance gain thực sự |

---

## Verification Plan

### Automated Tests
- `pnpm build` — TypeScript compile thành công.

### Manual Verification
1. Navigation: Click house → rooms. Switch tab → content đổi.
2. Modals từ HouseRoomsView: Tạo phòng, sửa phòng, set giá nhanh, tenant CRUD, xóa phòng.
3. Modals từ Sidebar: Tạo nhà, xóa nhà.
4. Modals từ Dashboard: Tạo nhà.
5. Form validation: Bỏ trống required fields → hiện lỗi Zod.
6. Form edit: Click sửa tenant/room → form load đúng data. File upload/remove.
7. Pagination, LocalStorage persist, Sidebar collapse.
