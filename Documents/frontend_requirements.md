# Yêu cầu Frontend (ReactJS)

Tài liệu này xác định các yêu cầu giao diện (UI) và trải nghiệm người dùng (UX) đã và đang được phát triển cho dự án.

## 1. Kiến trúc App
- Viết bằng ReactJS (Single Page Application).
- Modular Components: Quản lý Code chặt chẽ (Tách riêng UI components chung, Views, Hooks xử lý API, Context cho global states).
- **Manager Dashboard**: View quản trị với Sidebar điều hướng (Khách thuê, Nhà trọ, Thống kê), hỗ trợ thu gọn mở rộng mượt mà.
- **Authentication Layout**: Giao diện chung cho đăng nhập, đăng ký.

## 2. Các trang chính & Tính năng (Manager)
- **Trang Đăng nhập / Đăng ký**: Hoạt động với AuthContext và local token parsing.
- **Trang Tổng quan (Dashboard)**: Thống kê số phòng trống, số tiền thu trong tháng, số hóa đơn chưa thanh toán. Trình bày lưới Grid Cards (StatCards) rõ ràng.
- **Tính năng Quản lý Nhà trọ & Tổ chức Phòng**: 
  - Khóa (Modal) tạo nhà trọ mới, cho phép tự động map khởi tạo các phòng (Auto Generate) với hậu tố đúng chuẩn P101, P102... khi nhập số tầng và số lượng.
  - Set Default Pricing (Thiết lập giá tiện ích trung tâm): Điện, Nước, Wifi, Gửi xe.
  - Phân trang (Pagination) thông minh tối đa 25 phòng / trang cho mỗi khu nhà.
  - Điều chỉnh phụ phí nâng cao (tùy chỉnh riêng cho một phòng độc lập qua EditRoom Modal, nếu trống giá sẽ lấy mặc định của nhà).
- **Trang Quản lý Khách thuê**: Tổ chức hiển thị dữ liệu Group theo Nhà Trọ (Theo cấu trúc Data Structure Table).

## 3. Core UI Guidelines
- Hướng tới **Soft Light Aesthetic**: Nền nhạt, văn bản tương phản nhẹ (slate-800) không gây nhức mắt khi trực ban lâu.
- **Micro-interactions**: Scale Effects khi Hover, Nổi bóng Box Shadow. 
- Giữ trạng thái của Web (Persisted State): Ghi nhớ nơi đang thiết lập / đang xem dở vào Browsers `localStorage` (Ví dụ Active Tab & Id Nhà đang khảo sát).
