# Quy ước Styling (Design System)

UI Frontend được thiết kế dựa trên triết lý **Soft Light** kết hợp với **Glassmorphism**, mang đến trải nghiệm người dùng hiện đại, sắc nét cao cấp nhưng không gây chói/mỏi mắt khi thao tác trong thời gian dài (rất quan trọng do tính chất ứng dụng quản trị số liệu).

## 1. Bảng màu chủ đạo (Palette)
Màu sắc được sử dụng trực tiếp qua các cấu hình hệ màu mặc định của Tailwind CSS nhưng được chọn lọc dựa trên tiêu chí sau:

- **Nền Giao Diện (Background)**:
  - Nền toàn trang sử dụng màu Xám Dịu `bg-slate-50` tạo độ ấm, tránh dùng màu trắng tinh (#FFFFFF) gây chói.
- **Văn bản (Typography)**:
  - Tiêu đề chính, Text nổi bật: `text-slate-800` hoặc `text-slate-900`.
  - Phụ đề, Chú thích: `text-slate-500` hoặc `text-slate-400`.
- **Màu Nhấn (Brand / Interactions)**:
  - Primary Color: Tone màu Tím (`purple-600` cho nút mặc định, `purple-700` khi hover). Tone màu này được trải dài từ Sidebar đến Logo tạo nét cao cấp.
  - Selection: Dùng `selection:bg-purple-200/50` thay vì bôi đen mặc định của Trình duyệt.

## 2. Glassmorphism và Depth

Thay vì dùng các layout cứng nhắc, UI tạo chiều sâu (Depth) thông qua hiệu ứng kính mờ và bóng đổ mịn:

- **Các Cards và Sidebar**: Áp dụng nền kính mờ: `bg-white/60 backdrop-blur-3xl` hoặc `bg-white/80`.
- **Viền khéo léo (Borders)**: Dùng các viền border mảnh mai cực kì tinh tế với độ mờ: `border-slate-200/60` thay vì các vách chia solid sẫm màu.
- **Shadows**: Nâng bóng đổ mềm (`shadow-slate-200/40`, `shadow-md`) trên các Modal hay Card khi chuột đi qua.

## 3. Micro-Interactions (Hoạt ảnh tương tác nhỏ)

Dynamic UI giúp người dùng có cảm giác ứng dụng "Sống" và phản hồi:

- **Hover Card**: Hiệu ứng nổi (`hover:translate-y-[-4px] transition-all duration-300`) đối với StatCard.
- **Scale Elements**: Các khối Icon khi được group trỏ chuột vào sẽ hơi phình to ra nhè nhẹ (`group-hover:scale-110`).
- **Smooth Sidebar Transitions**: Thu phóng (Collapse/Expand) của Sidebar không "giật" mạnh, quy chuẩn bằng class `transition-all duration-300 ease-in-out`.
- **Hoạt ảnh Menu**: Đánh dấu Active bằng chấm tròn (`scale-100` vs `scale-0` có sẵn ở Sidebar item houses list).
