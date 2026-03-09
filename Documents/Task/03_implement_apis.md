# Task 3: Viết API theo yêu cầu

## Mục tiêu
- Triển khai các API backend đã được định nghĩa trong `api_specifications.md`.

## Công việc chi tiết
1. **Auth APIs**:
   - `POST /register`: Đăng ký Manager.
   - `POST /login`: Đăng nhập lấy JWT.
   - `POST /verify-zalo`: API kích hoạt tài khoản.
2. **House Management APIs**:
   - `GET`, `POST`, `PUT`, `DELETE` cho House.
   - Bổ sung logic filtering và sorting cho `GET`.
3. **Room Management APIs**:
   - CRUD cho Room.
   - Logic filtering và sorting cho `GET`.
4. **Tenant Management APIs**:
   - Thêm người thuê, tự động sinh account và gửi qua Zalo API.
   - Thay đổi trạng thái người thuê.
5. **Invoice Management APIs**:
   - Tính toán và lưu hóa đơn dựa trên chỉ số điện nước.
   - Cập nhật trạng thái thanh toán.
6. **Tenant APIs**:
   - Các API `GET` dành riêng cho người thuê xem hóa đơn và thông tin phòng của mình.
