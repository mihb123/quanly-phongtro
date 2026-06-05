# Cải thiện Service Test Coverage từ 81% → 90%+

Thêm error-path test cases cho các hàm có coverage thấp, giữ nguyên gomock pattern, thêm trực tiếp vào file test hiện tại. Bỏ qua `HandleWebhook` (60.3%) vì ROI thấp.

## Proposed Changes

### auth_service — 7 hàm cần cải thiện

#### [MODIFY] [auth_service_test.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/auth_service_test.go)

Thêm test cases mới cho các nhánh chưa cover:

**Register (70% → ~95%)**
- `GetByEmail returns unexpected error` — cover dòng 115-116 (`!errors.Is(err, model.ErrNotFound)`)
- `Hash password fails` — cover dòng 120-121
- `Create returns non-duplicate error` — cover dòng 138
- `GenerateAccessToken fails` — cover dòng 142-143
- `Create returns ErrAlreadyExists` — cover dòng 134-135

**Login (73.7% → ~95%)**
- `GetAuthUserByEmail returns unexpected error` — cover dòng 171
- `Wrong password` — cover dòng 174-175
- `GenerateAccessToken fails` — cover dòng 179-180
- `GenerateRefreshToken fails` — cover dòng 184-185

**RefreshToken (66.7% → ~95%)**
- `Parse fails` — cover dòng 213-214
- `FindByToken fails` — cover dòng 223-224
- `Token is revoked` — cover dòng 227-228
- `GetByUserID fails` — cover dòng 232-233
- `GenerateAccessToken fails after user lookup` — cover dòng 237-238
- `GenerateRefreshToken fails` — cover dòng 242-243

**GetMe (50% → ~95%)**
- `GetByUserID returns ErrNotFound` — cover dòng 256-257
- `GetByUserID returns unexpected error` — cover dòng 259

**CreateOTP (75% → ~95%)**
- `CreateOTP repo fails` — cover dòng 285-286
- `EmailSender fails` — cover dòng 291-292
- `EmailSender is nil` — verify nil sender doesn't panic (đã partial cover)

**VerifyEmail (66.7% → ~95%)**
- `GetOTP returns non-ErrNoRows error` — cover dòng 309-310
- `OTP already used (IsUsed=true)` — cover dòng 312
- `UpdateUsedOTP fails` — cover dòng 317-318
- `ActivateUser fails` — cover dòng 321-322

**IncrementOTPCheck (50% → ~95%)**
- `GetOTPCheck returns ErrNoRows, CreateOTPCheck fails` — cover dòng 332-333
- `GetOTPCheck returns unexpected error` — cover dòng 336-337

**IsBlockOTP (81.8% → ~95%)**
- `GetOTPCheck returns unexpected error` — cover dòng 346-347
- `ResetOTP fails` — cover dòng 352-353

---

### tenant_service — 3 hàm cần cải thiện

#### [MODIFY] [tenant_service_test.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/tenant_service_test.go)

**RegisterTenant (76.2% → ~95%)**
- `Auto-generate email from FullName (no phone, no email)` — cover dòng 131-137
- `Auto-generate email with empty FullName (guest)` — cover dòng 132-133
- `Hash password fails` — cover dòng 162-163
- `UpdateRoomStatus fails` — cover dòng 200-201
- `Unsupported contract file extension` — cover dòng 179-183 cho ContractFiles

**UpdateTenantInfo (63.6% → ~90%)**
- `UpdateTenant repo error` — cover dòng 302-303
- `GetTenantByID (final re-fetch) error` — cover dòng 307-308
- `New CCCD files without kept paths` — cover dòng 277-278
- `New contract files with kept paths` — cover dòng 289-295
- `New contract files without kept paths` — cover dòng 293-294

**DeleteTenant (80% → ~95%)**
- `DeactivateUser fails` — cover dòng 332-333
- `GetCurrentNumTenantInRoom fails` — cover dòng 337-338
- `UpdateRoomStatus fails (remaining=0 branch)` — cover dòng 342-343

---

### invoice_service — CreateInvoice cần cải thiện

#### [MODIFY] [invoice_service_test.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/invoice_service_test.go)

**CreateInvoice (74.7% → ~90%)**
- `GetRoomByIDOnly fails` — cover dòng 53-55
- `IsHouseOwnedBy fails (returns err)` — cover dòng 59-60
- `GetByID (house) fails` — cover dòng 67-68
- `GetPreviousInvoice returns non-ErrInvoiceNotFound` — cover dòng 77-78
- `GetCurrentNumTenantInRoom fails` — cover dòng 97-98
- `Room has custom prices (electricity, water, wifi, parking, service overrides)` — cover dòng 104-126
- `FIXED billing type with index normalization` — cover dòng 141-151
- `Invalid water index (less than old)` — cover dòng 149-150
- `GetInvoiceByRoomAndPeriod returns unexpected error` — cover dòng 213-214
- `Extra person fee and extra vehicle fee calculations` — cover dòng 176-183

---

### zalo_client — 4 hàm cần cải thiện

#### [MODIFY] [zalo_client_test.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/zalo_client_test.go)

**GetMe (86.4% → ~95%)**
- `HTTP status non-200` — cover dòng 48-49

**SendMessage (81.2% → ~95%)**
- `Invalid JSON response body` — verify decode behavior (nhánh không kiểm tra)

**SendPhoto (73.1% → ~95%)**
- `HTTP non-200 status` — cover dòng 152-153
- `Caption empty` — cover dòng 121 (skip WriteField for text)

**SetWebhook (77.3% → ~95%)**
- `OK response but decode error` — cover dòng 190-191
- `API returns ok=false` — cover dòng 194-195

---

### zalo_cron — markInactive cần cải thiện

#### [MODIFY] [zalo_cron_test.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/zalo_cron_test.go)

**markInactive (66.7% → ~95%)**
- `UpdateUser fails in markInactive` — cover dòng 105-106 (log.Printf branch)

---

### zalo_service — các hàm ngoài HandleWebhook

Các hàm cần cải thiện (bỏ qua HandleWebhook 60.3%):
- `SendTextMessage` (75%), `SendInvoiceToZalo` (76.9%), `NewZaloService` (75%)
- Sẽ thêm tests cho các error branches cơ bản

#### [MODIFY] [zalo_service_test.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/zalo_service_test.go)

Thêm error-path tests cho SendTextMessage, SendInvoiceToZalo nếu cần.

## Verification Plan

### Automated Tests
```bash
go test -cover ./internal/service
```
Mục tiêu: coverage ≥ 90%
