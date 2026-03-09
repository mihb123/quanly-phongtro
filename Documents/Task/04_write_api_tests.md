# Task 4: Viết các file test để kiểm thử API

## Mục tiêu
- Đảm bảo tính chính xác và ổn định của các API đã viết thông qua Unit Test và/hoặc Integration Test.

## Công việc chi tiết
1. **Setup môi trường Test**: Tạo test database riêng để chạy test tự động.
2. **Viết test cho Controller/Handler**:
   - Test các endpoint Auth (đăng ký, đăng nhập).
   - Test các API bảo mật cần Token (kiểm tra JWT Middleware).
   - Test logic CRUD (House, Room).
   - Test logic tạo tài khoản đồng thời khi thêm Tenant.
   - Test logic tính tiền Hóa đơn dựa trên điện nước cũ mới.
   - Test các filter bằng query params (`?sort=...&filter=...`).
3. **Kiểm tra Edge Cases**: Những dòng dữ liệu lỗi (sai mã nhà, thiết lập giá không hợp lệ, đăng ký email trùng...).
