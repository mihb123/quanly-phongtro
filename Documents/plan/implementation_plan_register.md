# Cải thiện luồng DPoP Authentication

## Thiết kế đã thống nhất

Luồng sau khi cải thiện:

```
Đăng ký → Tự động login (tạo session + cookie) → Redirect /verify-email
→ Gửi OTP → Nhập OTP → Verify thành công → Backend trả token mới (is_activated=true)
→ Frontend set token + getMe() → vào Homepage
```

- **User chưa verify email bị chặn hoàn toàn** khỏi Homepage (ProtectedRoute redirect về `/verify-email`)
- **Logout revoke session trong DB** (không chỉ xóa cookie)

---

## Proposed Changes

### Backend: Register tạo session đầy đủ

#### [MODIFY] [auth_service.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/auth_service.go)

- Thay đổi hàm `Register`: ngoài `GenerateAccessToken`, thêm `GenerateRefreshToken` giống luồng Login
- Return type thay từ `*AuthOutput` sang `*LoginOutput` (chứa cả access_token + refresh_token + expires_in)
- Cần thêm logic location lookup (copy từ Login)

#### [MODIFY] [auth_handler.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/handler/auth_handler.go)

- Handler `Register`: gọi `setTokenCookies()` sau khi register thành công (giống Login)
- Handler `Logout`: lấy `refresh_token` từ cookie → parse → lấy userID → gọi `RevokeRefreshToken()` → rồi mới `clearTokenCookies()`
- Handler `VerifyEmail`: sau khi verify thành công, sinh `access_token` mới với `is_activated=true` → set cookie mới → trả về trong response

#### [MODIFY] [auth_service.go](file:///media/minhchu1336/Data/quanly-phongtro/internal/service/auth_service.go) (interface)

- Cập nhật `AuthService` interface: `Register` trả `*LoginOutput` thay vì `*AuthOutput`
- Thêm method `VerifyEmailAndRefreshToken` hoặc cập nhật `VerifyEmail` để trả token mới

---

### Frontend: Trang Verify Email + chặn user chưa activated

#### [MODIFY] [ProtectedRoute.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/components/ProtectedRoute.tsx)

- Thêm check `user.is_activated === false` → redirect đến `/verify-email`
- Giữ nguyên logic loading spinner và check `!user` → redirect `/login`

#### [NEW] [VerifyEmail.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/pages/VerifyEmail.tsx)

Trang đơn giản:
- Hiển thị email user (lấy từ AuthContext)
- Nút "Gửi mã OTP" → gọi `GET /auth/verify-email`
- Ô nhập 6 số OTP
- Nút "Xác nhận" → gọi `POST /auth/verify-email/otp` → nhận token mới → set vào memory + gọi `getMe()` → navigate `/`

#### [MODIFY] [router/index.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/router/index.tsx)

- Thêm route `/verify-email` → `<VerifyEmailPage />`

#### [MODIFY] [api/auth.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/api/auth.tsx)

- Thêm hàm `createOTP()` → `GET /auth/verify-email`
- Thêm hàm `verifyEmailOTP(otp)` → `POST /auth/verify-email/otp`

#### [MODIFY] [pages/Register.tsx](file:///media/minhchu1336/Data/quanly-phongtro/frontend/src/pages/Register.tsx)

- Sau đăng ký thành công: `navigate('/verify-email')` thay vì `navigate('/')`

---

## Verification Plan

### Automated Tests
- `go test ./internal/...` — đảm bảo tất cả tests vẫn pass
- Update unit test cho Register handler (giờ trả LoginOutput + set cookie)
- Thêm unit test cho Logout handler (verify revoke được gọi)

### Manual Verification
- Puppeteer E2E: Register → redirect verify-email → nhập OTP → vào Homepage
- Reload ở mỗi bước để đảm bảo session persist
- Logout → kiểm tra DB session đã revoked
