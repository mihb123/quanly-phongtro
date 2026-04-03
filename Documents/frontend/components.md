# Danh sách Frontend Components

Tài liệu này liệt kê chi tiết các thành tố UI nội bộ, đặc biệt trong quy trình quản lý Dashboard (Home) và các form hành vi.

## 1. Tổ chức giao diện chính (Home Views)

Giao diện ứng dụng được lắp ráp từ các components sau trong thư mục `src/components/home/`:

- **`SidebarItem.tsx`**: Thành phần menu điều hướng. Hỗ trợ hiển thị dạng Collapse (chi icon) hoặc đầy đủ, xử lý linh hoạt trạng thái `active`.
- **`DashboardView.tsx`**: Trang tóm tắt. Nơi hiển thị báo cáo tổng. Tận dụng  **`StatCard.tsx`** để tái sử dụng hiển thị các thông số (Tổng doanh thu, Phòng đã thuê).
- **`HouseRoomsView.tsx`**: Thành phần hiển thị danh sách tất cả các phòng thuộc 1 khu trọ. 
  - Tại đây có thanh điều hướng phân trang.
  - Quản lý logic khi click vào icon "Edit" của một phòng trọ cụ thể.
- **`TenantsView.tsx`**: Danh sách người thuê, được nhóm trực quan (Group) theo từng Nhà Trọ.

## 2. Các Modals xử lý Dữ liệu

Form quản lý dữ liệu được thiết kế dạng Modal chìm (Overlay), sử dụng Glassmorphism UI để trông hiện đại hơn.

- **`CreateHouseModal.tsx`**:
  - **Auto-generator**: Chứa thuật toán hỗ trợ người dùng điền tổng số tầng toà nhà và khởi tạo động số phòng ở từng tầng. Hệ thống sẽ tự loop để sinh ra danh sách phòng trọ cho backend theo định dạng (Tầng 1 -> P101, P102).
  - Phân tách rõ ràng phí dịch vụ trung tâm (Giá điện, Giá nước, Wifi) cho toàn toà nhà.
- **`CreateRoomModal.tsx`**: Form căn bản tạo 1 phòng thủ công.
- **`EditRoomModal.tsx`**: Giao diện cập nhật cấu hình cho từng phòng. Đặc điểm nổi bậc: Hỗ trợ người quản lý *override* (ghi đè) giá phí dịch vụ (tiền điện, nước) mặc định của tổ hợp nhà nếu có nhu cầu tuỳ biến riêng lẻ.

## 3. UI Guidelines & Bug Fixes 

Trong quá trình lắp ráp components, một số nguyên tắc UI bắt buộc phải tuân theo:

- **Chống vỡ Layout (Long text handling)**: Danh sách tên nhà, địa chỉ bắt buộc phải áp dụng chuỗi classes `whitespace-nowrap overflow-hidden text-ellipsis` để luôn có dấu `...` khi text quá dài.
- **Hiệu ứng Click (Cursor Feedback)**: Mọi Element có thể tương tác (Buttons, Menu Items, List Items) đều phải có `cursor-pointer`.
- **Empty States**: Khi một Nhà trọ không có phòng, hiển thị State trống (Empty Text) một cách tinh tế thay vì để khung giao diện trắng.
