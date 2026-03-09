# Yêu cầu Backend (Golang)

## Chức năng hệ thống chính
1. **Quản lý quyền (Authorization & Authentication)**
   - Sử dụng JWT để mã hóa phiên đăng nhập.
   - Viết Middleware để check quyền:
      - `RoleManager`: Chỉ cho phép người dùng có `role = MANAGER` gọi các API CRUD nhà, phòng, người thuê, hóa đơn.
      - `RoleTenant`: Cho phép `role = TENANT` gọi các API xem hóa đơn của chính phòng họ.
      - Check Ownership: Manager A KHÔNG ĐƯỢC phép truy xuất hay chỉnh sửa Nhà / Phòng / Hóa đơn của Manager B. Bắt buộc kiểm tra quan hệ `house.manager_id == current_user.id`.

2. **Cronjob / Tự động hóa (Tùy chọn nâng cao)**
   - Hệ thống có khả năng tự động gen nháp (Draft) hóa đơn vào ngày cuối tháng hoặc mùng 1 đầu tháng để Manager chỉ việc điền số điện nước mới và xác nhận.

3. **Transaction an toàn**
   - Đảm bảo tính nhất quán dữ liệu ở các thao tác phức tạp, ví dụ: Khi thêm 1 Tenant, hệ thống tạo User + tạo hồ sơ Tenant ==> phải dùng DB Transaction rollback nếu 1 trong 2 lỗi.

4. **Xử lý tính toán công thức hóa đơn**
   - Đảm bảo độ chính xác (Precision) khi nhân chia giá tiền. Trong Go nên lưu ở định dạng số thực khắt khe, hoặc quy về số nguyên (vi phân tiền / VND thì có thể xài int/int64 để tránh sai số floating point).

5. **Security & Validation**
   - Mật khẩu phải mã hóa bcrypt.
   - Validate API input (body, header, query) đầy đủ bằng thư viện validator như `go-playground/validator`.
