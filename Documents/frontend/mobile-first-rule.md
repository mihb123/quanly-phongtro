# 📱 Tiêu chuẩn Thiết kế Mobile-First (Mobile-First Rules)

Tài liệu này định nghĩa các nguyên tắc cốt lõi và hướng dẫn thực thi khi phát triển giao diện theo hướng **Mobile-First**. Mục tiêu là mang lại trải nghiệm mượt mà, tối ưu không gian, thân thiện với ngón tay người dùng (Touch-friendly) trên thiết bị di động trước khi mở rộng lên màn hình lớn.

---

## 1. Chiến lược Layout & Breakpoints (Mặc định là Mobile)

*   **Mobile làm gốc (Baseline):** Toàn bộ các class CSS/Tailwind (không có prefix) mặc định phải áp dụng cho màn hình di động (dọc).
    *   ✅ *Đúng:* `flex flex-col md:flex-row w-full md:w-1/2`
    *   ❌ *Sai:* `flex-row w-1/2 md:flex-col md:w-full` (Tuyệt đối không lấy Desktop làm chuẩn rồi dùng class để bóp lại cho mobile).
*   **Grid Hữu Cơ (Fluid Grid):** Khuyến khích sử dụng Grid với `auto-fit` hoặc `auto-fill` (vd: `grid-cols-[repeat(auto-fit,minmax(250px,1fr))]`) để các card tự rớt dòng trên mobile mà không cần khai báo quá nhiều breakpoint `sm:`, `md:`, `lg:`.
*   **Container Queries (Khuyến khích):** Khi xây dựng các UI component độc lập (như Card, List Item), cân nhắc sử dụng `@container` (của Tailwind) thay vì media queries (`@media`) để component tự co giãn dựa trên kích thước của thẻ cha chứa nó thay vì kích thước toàn màn hình.
*   **Không Fix cứng kích thước:** Không sử dụng các giá trị tĩnh lớn như `w-[500px]`, `w-96`, `h-[300px]`. Dùng `w-full`, `max-w-md`, `min-h-screen`, `aspect-video` để phần tử tự động co giãn.

## 2. Tương tác và Tiện ích Chạm (Touch-Target & Navigation)

*   **Kích thước vùng chạm (Touch Targets):** Đảm bảo tất cả phần tử tương tác (Button, Link, Icon Button, Checkbox, Switch) có kích thước tối thiểu **44x44px** (theo chuẩn Apple/Google). 
    *   Sử dụng `min-h-[44px] min-w-[44px]` hoặc padding đủ rộng (vd: `p-3`) để mở rộng diện tích chạm.
*   **Vùng An Toàn (Safe Areas):** Giao diện mobile hiện đại (tai thỏ, Dynamic Island, thanh điều hướng dưới) yêu cầu khoảng trống an toàn.
    *   Luôn cộng thêm `pb-[env(safe-area-inset-bottom)]` cho các thanh Bottom Navigation hoặc nút nổi (FAB) ở đáy màn hình.
    *   Cộng thêm `pt-[env(safe-area-inset-top)]` cho các Header cố định sát cạnh trên.
*   **Tái cấu trúc Điều hướng (Navigation):**
    *   Thay thế Navbar ngang phức tạp thành **Bottom Navigation Bar**, Menu Hamburger, hoặc **Drawer / Bottom Sheet**.
    *   Thiết kế theo **Thumb Zone** (Vùng ngón cái): Đặt các nút hành động chính (Primary CTA) ở nửa dưới màn hình để dễ thao tác bằng một tay.
*   **Phản hồi chạm (Touch Feedback):** 
    *   Mobile không có `hover`. Mọi tương tác chạm phải được phản hồi ngay lập tức qua hiệu ứng `active:` (vd: `active:scale-95 active:opacity-80`) hoặc ripple effect.
    *   Thay thế các menu Dropdown dài bằng Modal, Bottom Sheet hoặc giao diện chọn ở màn hình mới.
    *   Thêm `touch-manipulation` để vô hiệu hóa độ trễ double-tap 300ms trên các nút bấm.

## 3. Xử lý Dữ liệu Phức tạp (Bảng biểu, Danh sách, Văn bản)

*   **Hiển thị Bảng dữ liệu (Data Tables):** 
    *   *Giải pháp 1 (Cuộn ngang):* Bọc bảng bằng div có `overflow-x-auto` và `overscroll-x-contain`.
    *   *Giải pháp 2 (Card View):* Chuyển đổi mỗi dòng (row) của bảng thành dạng **Thẻ thông tin (Card view)** xếp dọc trên thiết bị di động (phổ biến và UX tốt hơn).
*   **Thao tác cuộn mượt mà (Scroll Snap):** Với các danh sách ngang (Carousel, Gallery, Tab list), sử dụng kết hợp `flex overflow-x-auto snap-x snap-mandatory` để tạo cảm giác vuốt trơn tru như app native. Ẩn thanh cuộn bằng `scrollbar-width: none` hoặc các plugin Tailwind.
*   **Xử lý Text quá dài:** Luôn chủ động sử dụng `truncate` (chấm lửng 1 dòng) hoặc `line-clamp-2`, `line-clamp-3` để cắt ngắn văn bản dài, tránh việc nội dung đẩy vỡ layout mobile.
*   **Ngăn chặn vỡ khung (Viewport Overflow):** 
    *   Mọi phần tử (đặc biệt hình ảnh, biểu đồ, table) phải luôn có `max-w-full`.
    *   Thiết lập `overflow-hidden` hoặc `overflow-x-hidden` ở cấp container cha (như `body` hoặc thẻ bọc ngoài cùng) để triệt tiêu scrollbar ngang.

## 4. Tối ưu hóa Hiển thị và Trải nghiệm (UI/UX Mechanics)

*   **Responsive Modals (Dialog vs Drawer):** 
    *   Trên Desktop: Dùng hộp thoại Modal / Dialog canh giữa màn hình.
    *   Trên Mobile: Các Modal nằm giữa màn hình thường mang cảm giác chật chội. Ưu tiên chuyển đổi chúng thành **Bottom Sheet (Drawer)** trượt từ dưới lên (ví dụ dùng `vaul` hoặc Drawer của Radix UI / Shadcn).
*   **Typography & Form Inputs (Zoom Bug):**
    *   Cấu hình `font-size` tối thiểu cho các thẻ `<input>`, `<textarea>`, `<select>` là **16px** (hoặc `text-base` trong Tailwind) để ngăn trình duyệt iOS Safari **tự động zoom-in** khi người dùng chạm vào ô nhập liệu.
*   **Bàn phím ảo (Virtual Keyboard):** Tránh đặt các thẻ input quan trọng sát đáy màn hình. Khi bàn phím bật lên, UI có thể bị che khuất. Nếu dùng absolute/fixed bottom, hãy cẩn thận kiểm tra hành vi đẩy layout của trình duyệt.
*   **Trạng thái Tải (Loading States):** Sử dụng Skeleton Loader cho các nội dung tải bất đồng bộ để giữ vững khung layout. Hạn chế dùng Spinner xoay giữa màn hình gây chặn tương tác (Blocking UI).

## 5. Quy trình Quét và Refactor Code (Hành động của Agent)

Khi AI/Agent tiến hành phân tích và refactor giao diện hiện có, hãy tuân thủ trình tự sau:

1.  **Kiểm tra Hardcoded (Cứng):** Tìm kiếm và loại bỏ các kích thước fix cứng (vd: `w-[800px]`, `w-96`, `style={{width: '600px'}}`) và đổi thành `w-full max-w-2xl` hoặc tỷ lệ phần trăm.
2.  **Reset & Tái cấu trúc Class (Mobile First):** Xóa các class layout cũ, viết lại chuỗi class Tailwind bắt đầu từ thiết kế dành cho mobile màn hình dọc (không prefix). Sau đó bổ sung layout mở rộng cho Tablet (`md:`, `lg:`) và Desktop (`xl:`, `2xl:`).
3.  **Tách Component Hợp lý:** Nếu một component có quá nhiều logic rẽ nhánh để phục vụ 2 UI khác nhau (vd: `isMobile ? <Drawer /> : <Dialog />`), hãy cân nhắc tách thành 2 component nhỏ (`<MobileView />` và `<DesktopView />`) thay vì cố nhồi nhét CSS phức tạp vào một nơi.
4.  **Kiểm tra Spacing & Touch-Target:** Dò tìm các nút bấm, icon, liên kết. Đảm bảo chúng có kích thước vùng chạm đủ lớn (tối thiểu `44px`). Nới lỏng `gap`, `padding` khi lên Desktop, nhưng giữ vừa vặn trên Mobile.
5.  **Review Vỡ Layout:** Luôn chú ý các chuỗi văn bản dài không có khoảng trắng (URL, email) - thêm `break-all` hoặc `truncate`. Chú ý bảng biểu và ảnh lớn. Tuyệt đối không bao giờ để trang web xuất hiện thanh cuộn ngang (Horizontal Scrollbar) ở cấp độ toàn trang (body).

## 6. Bài học & Kinh nghiệm Fix Bug Mobile Thực Tế (Lessons Learned)

Dưới đây là tổng hợp các kinh nghiệm xương máu khi xử lý lỗi giao diện trên thiết bị di động / Chrome DevTools:

*   **Lỗi lệch vị trí Popup Native (`<select>`, `<input type="date/month">`):** 
    *   *Vấn đề:* Trên trình giả lập Chrome DevTools và một số Webview di động, nếu thẻ cha (như Modal, Container) có chứa CSS `transform` (vd: các hiệu ứng `zoom-in`, `slide-in`, `scale`), popup tùy chọn mặc định của hệ điều hành sẽ bị văng ra khỏi màn hình hoặc lệch vị trí nghiêm trọng do lỗi render của trình duyệt (Stacking Context / Containing Block bug).
    *   *Cách khắc phục:* **Loại bỏ hoàn toàn** các class chứa hiệu ứng transform (như `zoom-in`, `slide-in-from-top`) khỏi các Modal/View có chứa Select hoặc Date Picker. Chỉ nên sử dụng hiệu ứng `fade-in` (chỉ dùng thuộc tính opacity) để an toàn 100%.

*   **Căn chỉnh thẳng hàng Form Input (Alignment & Shrink):**
    *   *Vấn đề:* Các nhãn (Label) và ô nhập liệu (Input) xếp ngang nhau bằng `flex items-center` rất dễ bị lệch hàng dọc trên mobile nếu độ dài chữ/icon khác nhau, vì flexbox mặc định cho phép thu hẹp (`flex-shrink: 1`).
    *   *Cách khắc phục:* Luôn gán kích thước cố định kết hợp chống co rút cho nhãn (VD: `w-20 shrink-0 md:w-auto md:shrink`) để ép tất cả các thẻ Input kề bên thẳng tắp thành một cột dọc.

*   **Xung đột Modal và Thanh điều hướng dưới (Bottom Navbar):**
    *   *Vấn đề:* Trên Mobile, Bottom Navbar thường dùng `fixed bottom-0`. Nếu Modal nằm lẫn trong cây DOM thông thường, Navbar có thể đè lên các nút thao tác ở đáy Modal.
    *   *Cách khắc phục:* Bắt buộc render toàn bộ Modal/Dialog thông qua **React Portals (`createPortal(..., document.body)`)** và gán `z-index` cực cao (vd: `z-[60]` hoặc `z-[100]`) để đảm bảo Modal nằm trên cùng của màn hình di động. Phần action (các nút Lưu/Hủy) nên được cố định ở đáy Modal để người dùng dễ bấm.

*   **Layout phần Thống kê (Stats) lộn xộn:**
    *   *Vấn đề:* Sử dụng `flex-wrap` để thả trôi các thẻ thống kê (Tổng tiền, Số hóa đơn...) thường khiến bố cục bị xô lệch, rớt dòng không đều, tạo cảm giác thiếu chuyên nghiệp.
    *   *Cách khắc phục:* Hạn chế dùng `flex-wrap` tự do. Thay vào đó, hãy dùng hệ thống lưới rõ ràng như `grid grid-cols-2 gap-3` (cho mobile) và chuyển sang `flex` hoặc `grid-cols-4` trên Desktop. Đừng quên `truncate` (cắt chữ) nếu các con số tiền tệ quá lớn để tránh phá vỡ ô lưới.
