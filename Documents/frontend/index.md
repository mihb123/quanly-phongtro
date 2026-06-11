# Tài liệu Frontend

## Tổng quan
Tài liệu này cung cấp các nguyên tắc tổ chức mã nguồn, cấu trúc thư mục và quy ước dành cho phần Frontend (ReactJS) của dự án Quản lý phòng trọ. 
Mục tiêu là giúp quá trình maintain và scale dự án sau này trở nên rõ ràng và đồng nhất.

## Sơ đồ tổ chức tài liệu

Dưới đây là danh sách các tài liệu (dự kiến/đã có) giải thích chi tiết cho Frontend nằm trong `Documents/frontend`:

1. **[architecture.md](./architecture.md)**: Kiến trúc tổng thể của Frontend, sơ đồ Component và cách quản lý State.
2. **[components.md](./components.md)**: Danh sách các UI components chung (Tailwind, shadcn/ui), quy tắc thiết kế và cách tái sử dụng.
3. **[api_integration.md](./api_integration.md)**: Cách Frontend kết nối với Backend, quy tắc viết API/Axios và cách xử lý lỗi.
4. **[styling.md](./styling.md)**: Quy ước về CSS, TailwindCSS, tổ chức Design Tokens và Palette màu sắc.
5. **[invoice_house_selection.md](./invoice_house_selection.md)**: Logic tự động chọn nhà trọ khi tạo hóa đơn dựa trên trạng thái hóa đơn của các phòng.

## Cấu trúc thư mục mã nguồn (`frontend/src`)

- `api/`: Chứa các hàm giao tiếp với Backend (Ví dụ: `auth.ts` gồm các API calls đăng nhập/đăng ký).
- `assets/`: Chứa các hình ảnh, fonts, icons tĩnh cần thiết.
- `components/`: Chứa các UI Components tái sử dụng (được tổ chức theo `ui`, `layout`, `home` v.v.).
- `contexts/`: React Contexts để chia sẻ trạng thái toàn cục (Ví dụ: `AuthContext` quản lý thông tin đăng nhập tập trung).
- `hooks/`: Các Custom Hooks dùng chung (Ví dụ: `useHomeData.ts` để gói gọn logic xử lý cho trang chủ).
- `pages/`: Các màn hình (Pages) chính của ứng dụng (`Home`, `Login`, `Register`, v.v.).
- `utils/`: Các hàm tiện ích, helpers xử lý (format ngày tháng, tiền tệ,...).

## Chức năng chính hiện có

1. **Xác thực người dùng (Authentication)**: Đăng nhập (`Login.tsx`), Đăng ký (`Register.tsx`) tích hợp React Hook Form, Zod schema mapping với RESTful API.
2. **Dashboard & Nhà trọ**: Thống kê doanh thu cơ bản; Giao diện xem danh sách nhà, giao diện xem và tạo các phòng trọ có hỗ trợ tính năng tự động tạo số phòng nối tiếp theo cấu hình tầng (`CreateHouseModal`).
3. **Tính năng trải nghiệm UX**: Lưu lại trạng thái phiên hoạt động (nhớ vị trí nhà đang xem) ở LocalStorage thay cho URL, có áp dụng Pagination thông minh chỉ xuất hiện khi số lượng phòng từ 25 đổ lên.
4. **Design System & Styling**: Giao diện Soft Light Theme dùng `Glassmorphism`, kết hợp `TailwindCSS` giúp ứng dụng hiện đại, chuyển động mượt và thân thiện với mắt người quản trị.

## Quy tắc tổ chức code (Conventions)

1. **Naming**: 
   - Components & Pages: PascalCase (vd: `HomePage.tsx`).
   - Hooks: camelCase, bắt đầu bằng từ khóa `use` (vd: `useAuth.ts`).
   - Utils/Api/Functions: camelCase (vd: `loginAccount`).
2. **Styling & UI**: 
   - Lấy tông sáng (Soft Light Theme) làm chủ đạo theo yêu cầu thiết kế để thân thiện với mắt.
   - Thống nhất config màu sắc qua `index.css` (:root oklch / hsl variables format) kết hợp với các shadcn utility classes có sẵn.
