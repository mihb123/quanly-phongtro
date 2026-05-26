# Kiến trúc Frontend (Architecture)

Tài liệu này mô tả cách ứng dụng ReactJS được thiết kế ở mức tổng thể, tập trung vào việc tổ chức logic, quản lý vòng đời dữ liệu và duy trì trạng thái của người dùng.

## 1. Tách biệt Component/Logic (Separation of Concerns)

Nhằm đảm bảo code dễ đọc và dễ maintain, phần UI và phần Logic được tách biệt theo mô hình sau:
- **Pages** (`src/pages/*`): Tổ chức bộ khung ngoài cùng (Layout). Thay vì nhồi nhét mọi thứ vào page, Page chỉ đóng vai trò container tổng (Ví dụ: `Home.tsx` gọi Sidebar và các View tương ứng).
- **Views & Components** (`src/components/home/*`): Các giao diện tính năng cụ thể được xé nhỏ.
  - `DashboardView`: Thể hiện màn hình tổng quan.
  - `HouseRoomsView`: Thể hiện giao diện danh sách phòng trọ cho từng tầng.
  - `TenantsView`: Hiển thị danh sách khách thuê.
  - **Modals** (`CreateHouseModal`, `EditRoomModal`, v.v.): Tách biệt các form nhập dữ liệu thành các tệp tin độc lập. Áp dụng quy tắc "Modal Ownership" chặt chẽ: Modal render tại component mà **chỉ component đó** trigger nó. Modal tự đảm nhiệm việc tương tác với Store/API, không đẩy ngược callback (`onSuccess`) lên parent.
- **Global Data Stores** (`src/data/*`): Tách biệt logic quản lý dữ liệu nghiệp vụ (Domain Data) ra khỏi các UI component bằng **Zustand**. Ví dụ: `houseData.ts`, `roomData.ts`, `selectedData.ts`.
- **Custom Hooks** (`src/hooks/*`): Chịu trách nhiệm bọc các đoạn logic thuần tuý có thể tái sử dụng.
  - *Ví dụ:* `useTenantList` gom nhóm logic fetch, delete và view state của danh sách khách thuê.
- **Utilities** (`src/utils/*`): Chứa các pure functions xử lý dữ liệu chung. Ví dụ: `file.ts` (trích xuất tên, kiểm tra ảnh).

## 2. Quản lý trạng thái (State Management)

Hệ thống kết hợp nhiều cơ chế quản lý trạng thái phù hợp cho từng mục đích:

- **Global Domain State (Zustand)**: Lưu trữ các dữ liệu xuyên suốt ứng dụng (`houses`, `rooms`, `selectedHouse`, `activeTab`). Đọc/ghi dữ liệu từ store, tránh tình trạng "Prop Drilling" (truyền state quá sâu qua nhiều lớp component).
- **Form State (React Hook Form + Zod)**: Toàn bộ form được quản lý bằng React Hook Form để tối ưu performance (không re-render mỗi keystroke) kết hợp cùng Zod để đảm bảo tính đồng nhất về Form Validation (từ Login/Register cho đến các CRUD modals).
- **Local State (`useState`, `useEffect`)**: Chỉ dành cho các thao tác giao diện nội bộ của một component: ví dụ như trạng thái `isLoading`, `showModal`, hoặc quản lý File upload (nơi RHF không hoạt động tối ưu).
- **Context API (`useAuth`)**: Phân phối trạng thái Đăng nhập, thông tin User Token ra toàn bộ các components.
- **Persistent State (`localStorage`)**:
  - Trạng thái Zustand (hoặc tương đương) lưu lại những cấu hình hiển thị UI như Tab hiện tại và Nhà trọ đang được active. Khi người dùng tải lại trang (F5), ứng dụng lấy lại đúng trạng thái đã có thay vì nhảy về trang Dashboard mặc định.

## 3. Kiến trúc API và Phân trang (Pagination)

- **API Integration**: Các file trong `src/api/*` đóng gói HTTP calls tới Backend. Dữ liệu trả về được xử lý thành các TypeScript Interfaces (`House`, `Room`, `Tenant`).
- **Phân trang (Pagination)**:
  - Do số lượng phòng/tenant có thể rất lớn, Frontend phối hợp với Backend để fetch theo trang (tham số `page` và `limit = 25` được cấu hình tại store `roomData.ts`).
  - Giao diện tự động ẩn cụm phân trang nếu dữ liệu có sẵn đổ lại chỉ bao phủ trên 1 trang duy nhất, tối ưu diện tích hiển thị.
