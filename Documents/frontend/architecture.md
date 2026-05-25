# Kiến trúc Frontend (Architecture)

Tài liệu này mô tả cách ứng dụng ReactJS được thiết kế ở mức tổng thể, tập trung vào việc tổ chức logic, quản lý vòng đời dữ liệu và duy trì trạng thái của người dùng.

## 1. Tách biệt Component/Logic (Separation of Concerns)

Nhằm đảm bảo code dễ đọc và dễ maintain, phần UI và phần Logic được tách biệt theo mô hình sau:
- **Pages** (`src/pages/*`): Tổ chức bộ khung ngoài cùng (Layout). Thay vì nhồi nhét mọi thứ vào page, Page chỉ đóng vai trò container tổng (Ví dụ: `Home.tsx` gọi Sidebar và các View tương ứng).
- **Views & Components** (`src/components/home/*`): Các giao diện tính năng cụ thể được xé nhỏ.
  - `DashboardView`: Thể hiện màn hình tổng quan.
  - `HouseRoomsView`: Thể hiện giao diện danh sách phòng trọ cho từng tầng.
  - `TenantsView`: Hiển thị danh sách khách thuê.
  - **Modals** (`CreateHouseModal`, `EditRoomModal`, v.v.): Tách biệt các form nhập dữ liệu thành các tệp tin độc lập.
- **Custom Hooks** (`src/hooks/*`): Chịu trách nhiệm bọc logic về State và API calls. 
  - *Ví dụ:* `useHomeData` xử lý việc fetching dữ liệu `houses`, `rooms`, quản lý phân trang và chọn nhà trọ hiện tại.

## 2. Quản lý trạng thái (State Management)

Hệ thống tận dụng sự kết hợp giữa các mechanism của React và trình duyệt để mang lại trải nghiệm tối ưu (không bị mất phiên làm việc khi Refresh):

- **Local State (`useState`, `useEffect`)**: Thao tác giao diện tức thời như Mở/Đóng Modal, thu gọn Sidebar.
- **Context API (`useAuth`)**: Phân phối trạng thái Đăng nhập, thông tin User Token ra toàn bộ các components.
- **Persistent State (`localStorage`)**: 
  - Các cấu hình hiển thị UI như Tab hiện tại (`home_active_tab`) và Nhà trọ đang được active (`home_selected_house_id`) được lưu lại phía trình duyệt. Khi người dùng tải lại trang (F5), ứng dụng lấy lại đúng trạng thái đã có thay vì nhảy về trang Dashboard mặc định.

## 3. Kiến trúc API và Phân trang (Pagination)

- **API Integration**: Các file trong `src/api/*` sử dụng custom fetch hoặc axios interceptor (nếu có) để gọi tới Backend. Dữ liệu trả về được xử lý thành các TypeScript Interfaces (`House`, `Room`).
- **Phân trang (Pagination)**: 
  - Do số lượng phòng có thể rất lớn, Frontend phối hợp với Backend để fetch theo trang (tham số `page` và `limit = 25`).
  - Giao diện tự động ẩn cụm phân trang nếu dữ liệu có sẵn đổ lại chỉ bao phủ trên 1 trang duy nhất, tối ưu diện tích hiển thị.
