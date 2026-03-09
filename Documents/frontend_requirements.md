# Yêu cầu Frontend (ReactJS)

Tài liệu này sẽ được bổ sung chi tiết sau theo luồng thiết kế UI/UX cụ thể. 

## Yêu cầu cơ bản hiện tại:
1. **Kiến trúc App**
   - ReactJS SPA (Single Page Application).
   - Chia làm 2 giao diện chính (hoặc 2 layout):
      - **Manager Dashboard**: View quản trị với Sidebar điều hướng (Tổng quan, Nhà trọ, Khách thuê, Hóa đơn).
      - **Tenant Portal**: View cho người thuê xem số liệu, hóa đơn hàng tháng.
      - **Authentication Layout**: Giao diện chung cho đăng nhập, đăng ký.

2. **Các trang dự kiến cần có (Manager)**
   - Trang Đăng nhập / Đăng ký.
   - Trang Tổng quan (Dashboard): Thống kê số phòng trống, số tiền thu trong tháng, số hóa đơn chưa thanh toán.
   - Trang Danh sách Nhà trọ: Thêm/Sửa/Xóa nhà, cấu hình giá điện/nước mặc định.
   - Trang Chi tiết Nhà trọ -> Danh sách Phòng.
   - Trang Quản lý Khách thuê: Tìm kiếm khách, thêm người mới, cấp tài khoản.
   - Trang Quản lý Hóa đơn: Giao diện kiểu bảng (Table) hoặc lưới để tiện chốt số điện nước và xuất hóa đơn hàng tháng.

*(Cấu trúc Component và luồng chạy chi tiết sẽ được thảo luận ở bước lên yêu cầu UI)*
