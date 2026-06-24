# Tiêu chuẩn Web App Mobile-First

Tài liệu này định nghĩa rule để thiết kế, review và refactor giao diện web app theo hướng mobile-first. Mục tiêu không chỉ là "nhìn được trên điện thoại", mà là đạt chuẩn sử dụng thực tế: responsive, accessible, nhanh, ổn định, an toàn và dễ bảo trì.

Nguồn chuẩn chính dùng để cải thiện rule: WCAG 2.2, MDN Web Docs, web.dev Core Web Vitals, OWASP ASVS/Cheat Sheets, Material Design và tài liệu PWA của MDN/web.dev.

---

## 0. Chuẩn nền bắt buộc

* **Mobile là baseline:** CSS không prefix luôn là giao diện mobile dọc. Chỉ dùng `sm:`, `md:`, `lg:`, `xl:` để mở rộng dần lên màn hình lớn.
  * Đúng: `flex flex-col gap-3 md:flex-row md:items-center`
  * Sai: `flex-row md:flex-col` khi ý đồ thực tế là sửa layout desktop cho mobile.
* **Viewport đúng:** App phải có `<meta name="viewport" content="width=device-width, initial-scale=1" />`. Không dùng `maximum-scale=1` hoặc `user-scalable=no` vì làm hỏng khả năng zoom của người dùng.
* **Không có horizontal scroll toàn trang:** `body` không được xuất hiện scrollbar ngang. Nếu bảng, biểu đồ hoặc media cần cuộn ngang, cuộn phải nằm trong container riêng có nhãn/ngữ cảnh rõ ràng.
* **Accessible theo WCAG 2.2 AA:** Giao diện phải dùng được bằng bàn phím, screen reader, zoom text và pointer/touch. Không thêm UI mới nếu không có focus state, label, trạng thái lỗi và semantic HTML phù hợp.
* **Core Web Vitals tốt:** Nhắm LCP <= 2.5s, INP <= 200ms, CLS <= 0.1 ở trải nghiệm người dùng thật, không chỉ trên máy dev.
* **Security baseline:** Không đưa dữ liệu nhạy cảm vào URL, localStorage hoặc client logs. Ưu tiên cookie `HttpOnly; Secure; SameSite=Lax/Strict` cho session nếu backend hỗ trợ. Có CSRF protection cho request thay đổi dữ liệu.

## 1. Layout, Breakpoints và Container

* **Fluid trước, breakpoint sau:** Ưu tiên `w-full`, `max-w-*`, `minmax()`, `%`, `fr`, `clamp()` và `aspect-*` thay vì width/height cố định.
* **Không fix cứng kích thước lớn:** Tránh `w-[500px]`, `w-96`, `h-[300px]`, `style={{ width: 600 }}` trên component dùng lại. Nếu bắt buộc, phải có `max-w-full`, `min-w-0` hoặc responsive constraint đi kèm.
* **Grid hữu cơ:** Dùng `grid-cols-[repeat(auto-fit,minmax(16rem,1fr))]` hoặc biến thể tương đương cho danh sách card. Với thông tin so sánh chặt chẽ, dùng grid cột rõ ràng như `grid-cols-2 md:grid-cols-4`.
* **Container queries cho component độc lập:** Khi layout phụ thuộc vào kích thước vùng chứa, dùng `@container`/container query thay vì media query toàn viewport.
* **Chặn overflow từ flex/grid:** Các item chứa text dài phải có `min-w-0`; media phải có `max-w-full`; card/list item không được tự đẩy rộng viewport.
* **Safe viewport units:** Với màn hình mobile hiện đại, ưu tiên `min-h-dvh`/`h-dvh` cho full-screen panel thay vì chỉ `100vh` khi app có header, footer hoặc bàn phím ảo.

## 2. Accessibility và Semantic UI

* **Semantic HTML trước ARIA:** Dùng đúng `<button>`, `<a>`, `<label>`, `<fieldset>`, `<dialog>`, heading order. Chỉ dùng ARIA khi semantic HTML không đủ.
* **Keyboard đầy đủ:** Tất cả hành động phải dùng được bằng `Tab`, `Enter`, `Space`, `Escape` khi phù hợp. Không tạo keyboard trap trong modal, drawer, popover.
* **Focus visible:** Không xóa outline nếu không thay bằng focus style rõ ràng. Focus ring phải đủ tương phản và không bị che bởi overflow.
* **Touch target:** WCAG 2.2 AA đặt sàn tối thiểu 24x24 CSS px với khoảng cách hợp lệ. Với web app mobile, dùng chuẩn sản phẩm cao hơn: vùng chạm tối thiểu 44x44px, hoặc 48x48px nếu app theo Material-style.
* **Text scaling:** Layout phải chịu được zoom trình duyệt và tăng font đến 200% mà không mất nội dung hoặc che nút chính.
* **Contrast và trạng thái:** Text, icon quan trọng, border input lỗi và trạng thái disabled phải đủ tương phản. Không truyền ý nghĩa chỉ bằng màu.
* **Reduced motion:** Animation, transition và parallax phải tôn trọng `prefers-reduced-motion`. Motion trang trí không được là điều kiện để hiểu hoặc thao tác.
* **Tên truy cập:** Icon button phải có accessible name qua `aria-label`, text ẩn hợp lệ hoặc tooltip không thay thế label.

## 3. Touch, Navigation và Interaction

* **Thumb zone:** Primary action trên mobile nên nằm trong vùng dễ chạm ở nửa dưới màn hình, đặc biệt với flow nhập liệu hoặc tác vụ lặp lại.
* **Navigation mobile:** Navbar ngang phức tạp phải chuyển thành bottom navigation, drawer, menu hoặc tab list có cuộn ngang kiểm soát. Không nhồi quá nhiều item vào một hàng.
* **Safe areas:** Header/footer/floating action phải tính `env(safe-area-inset-*)`, đặc biệt với bottom nav và nút cố định đáy.
* **Không phụ thuộc hover:** Mọi thông tin hoặc hành động xuất hiện khi hover phải có đường truy cập bằng tap/click/keyboard.
* **Touch feedback:** Button, list item và icon button phải có phản hồi `active:`, pressed state hoặc ripple phù hợp. Có thể dùng `touch-manipulation` cho phần tử tương tác.
* **Popover trên mobile:** Dropdown dài nên đổi thành bottom sheet, full-screen picker hoặc trang chọn riêng. Menu không được bị bàn phím hoặc bottom nav che mất hành động chính.

## 4. Forms và Nhập liệu

* **Input font tối thiểu 16px:** `<input>`, `<textarea>`, `<select>` dùng `text-base` hoặc tương đương để tránh iOS Safari tự zoom khi focus.
* **Label thật:** Mỗi field phải có label liên kết bằng `htmlFor`/`id` hoặc semantic tương đương. Placeholder không được thay thế label.
* **Keyboard phù hợp:** Dùng `type`, `inputmode`, `autocomplete`, `enterkeyhint`, `pattern` khi có lợi cho mobile keyboard và autofill.
* **Validation rõ ràng:** Lỗi phải nằm gần field, đọc được bằng screen reader, không chỉ đổi màu. Form submit lỗi phải focus tới vùng lỗi đầu tiên hoặc summary.
* **Bàn phím ảo:** Tránh đặt input quan trọng sát đáy khi có footer fixed. Với action bar fixed, phải kiểm tra khi keyboard mở trên iOS Safari và Android Chrome.
* **Chống double submit:** Nút submit cần trạng thái loading/disabled có chủ đích, idempotency hoặc guard phía client khi thao tác tạo/sửa dữ liệu.

## 5. Dữ liệu Phức tạp, Bảng và Văn bản

* **Data table:** Trên mobile, ưu tiên card view cho dữ liệu cần đọc theo từng dòng. Nếu giữ table, bọc bằng `overflow-x-auto overscroll-x-contain` và giữ header/label đủ ngữ cảnh.
* **Danh sách dài:** Dùng pagination, infinite scroll có mốc tải rõ, hoặc virtualization khi số item lớn. Không render hàng nghìn row trên mobile nếu không cần.
* **Scroll snap có kiểm soát:** Carousel, gallery, tab list dùng `overflow-x-auto snap-x` nhưng vẫn phải thao tác được bằng keyboard và screen reader.
* **Text dài:** Dùng `min-w-0`, `truncate`, `line-clamp-*`, `break-words` hoặc `break-all` tùy loại dữ liệu. URL/email/mã hóa đơn không được làm vỡ card.
* **Số liệu và tiền tệ:** Card thống kê cần grid ổn định, `tabular-nums` khi có thể, và strategy truncate/wrap cho số rất lớn.

## 6. Performance, Media và Stability

* **Core Web Vitals là tiêu chí review:** LCP <= 2.5s, INP <= 200ms, CLS <= 0.1. Nếu thay đổi layout, ảnh, font, chart hoặc modal làm xấu các chỉ số này, phải điều chỉnh trước khi giao.
* **Ảnh responsive:** Dùng kích thước ảnh phù hợp, `srcset`/`sizes` khi cần, định dạng WebP/AVIF nếu pipeline hỗ trợ, và luôn khai báo width/height hoặc aspect ratio để tránh CLS.
* **Lazy load có chọn lọc:** Ảnh dưới fold dùng `loading="lazy"`. Ảnh hero/LCP không lazy load bừa bãi, cần ưu tiên tải đúng cách.
* **Giảm main-thread work:** Tránh render lại danh sách lớn, animation JS nặng, formatter chạy trên mỗi render và handler blocking. Tối ưu INP bằng memoization, debounce hoặc tách component khi thực sự cần.
* **Skeleton thay spinner chặn:** Nội dung bất đồng bộ nên có skeleton hoặc placeholder giữ kích thước. Spinner full-screen chỉ dùng khi không có nội dung có thể tương tác.
* **Font ổn định:** Dùng `font-display: swap/optional` nếu tự host font. Tránh đổi font late làm nhảy layout.

## 7. PWA, Offline và Reliability

* **Progressive enhancement:** App phải dùng được ở mức cơ bản nếu một enhancement thất bại. Không để màn hình trắng khi API, service worker hoặc asset phụ lỗi.
* **Offline state:** Nếu app có service worker/PWA, phải có offline fallback, cache strategy rõ và thông báo khi dữ liệu có thể đã cũ.
* **Manifest đúng:** PWA cần web app manifest với name, icons, theme/background color và display mode phù hợp. Icon phải đủ kích thước cho install.
* **Update strategy:** Service worker cần chiến lược update rõ để tránh người dùng kẹt ở bản cũ hoặc mất dữ liệu form đang nhập.
* **Network errors:** Mọi request quan trọng phải có loading, success, empty, error và retry state. Không nuốt lỗi API.

## 8. Security và Privacy cho Frontend

* **XSS:** Không render HTML chưa sanitize. Tránh `dangerouslySetInnerHTML`; nếu bắt buộc, sanitize bằng thư viện đáng tin và giới hạn tag/attribute.
* **CSP:** Web app production nên có Content-Security-Policy phù hợp để giảm rủi ro XSS và script injection.
* **Session/cookie:** Cookie phiên đăng nhập phải dùng `Secure`, `HttpOnly`, `SameSite=Lax` hoặc `Strict` khi phù hợp. `SameSite=None` bắt buộc đi cùng `Secure`.
* **CSRF:** Request thay đổi dữ liệu cần CSRF token hoặc cơ chế bảo vệ tương đương nếu dùng cookie-based auth.
* **Secrets:** Không commit hoặc embed API secret/private key vào frontend. Public env var vẫn là public.
* **PII:** Không ghi dữ liệu cá nhân, token, số điện thoại, email nhạy cảm vào console, analytics event hoặc query string nếu không có lý do rõ.

## 9. Quy trình Agent Khi Refactor UI

1. **Đọc ngữ cảnh trước:** Xác định component, layout cha, design system, framework và pattern có sẵn. Không tạo style mới nếu project đã có primitive/helper tương đương.
2. **Kiểm tra baseline:** Xác nhận viewport meta, mobile class không prefix, không có fixed width lớn, không có horizontal scroll body.
3. **Sửa layout mobile trước:** Viết lại cấu trúc mobile dọc, sau đó thêm breakpoint cho tablet/desktop. Giữ control flow và component boundary rõ ràng.
4. **Audit accessibility:** Kiểm tra semantic, label, focus, keyboard, touch target, contrast, reduced motion và screen-reader name cho icon button.
5. **Audit form và data:** Kiểm tra input 16px, keyboard mobile, validation, table/card view, text dài, empty/error/loading state.
6. **Audit performance:** Kiểm tra ảnh, skeleton, layout shift, render list, animation và handler có thể ảnh hưởng LCP/INP/CLS.
7. **Audit security frontend:** Kiểm tra HTML injection, token storage, URL chứa dữ liệu nhạy cảm, CSRF/cookie assumption và CSP nếu chạm vào shell/app config.
8. **Validate trên viewport thật:** Kiểm tra ít nhất 360x640, 390x844, 768x1024 và desktop. Nếu có thể, test bằng Chrome DevTools mobile và một trình duyệt mobile thật.
9. **Giữ thay đổi surgical:** Chỉ chạm file liên quan. Nếu thấy dead code hoặc vấn đề ngoài phạm vi, ghi chú riêng thay vì refactor lan rộng.

## 10. Checklist Review Nhanh

* Không có `w-[500px]`, `min-w-[600px]`, `h-screen` gây lỗi mobile khi có keyboard.
* Không có text hoặc media làm vỡ card/list/table.
* Touch target đạt tối thiểu 44x44px cho app mobile.
* Focus state rõ, tab order hợp lý, `Escape` đóng modal/drawer khi phù hợp.
* Input có label, lỗi, `autocomplete`/`inputmode` phù hợp và font >= 16px.
* Modal/drawer không bị bottom nav, safe area hoặc keyboard che nút chính.
* Không phụ thuộc hover để lộ hành động quan trọng.
* Loading, empty, error, retry và offline state được xử lý.
* Ảnh có dimension/aspect ratio, không gây CLS.
* Không render HTML chưa sanitize, không lưu secret/token nhạy cảm ở client storage.

## 11. Bài học Fix Bug Mobile Thực Tế

* **Lỗi lệch vị trí popup native (`<select>`, `<input type="date/month">`):**
  * Vấn đề: Trên Chrome DevTools mobile và một số WebView, nếu thẻ cha như modal/container có CSS `transform` (`zoom-in`, `slide-in`, `scale`), popup native có thể lệch khỏi màn hình do stacking context/containing block.
  * Cách khắc phục: Loại bỏ transform khỏi modal/view chứa select/date picker. Ưu tiên `fade-in` chỉ dùng opacity.
* **Căn chỉnh form input bị lệch:**
  * Vấn đề: Label và input xếp ngang bằng `flex items-center` dễ lệch trên mobile khi label dài hoặc icon khác kích thước.
  * Cách khắc phục: Gán label `shrink-0`, width ổn định ở mobile như `w-24 shrink-0`, hoặc chuyển form thành layout dọc rõ ràng.
* **Modal bị bottom navigation đè:**
  * Vấn đề: Bottom nav `fixed bottom-0` có thể đè action trong modal nếu modal không nằm ở layer cao.
  * Cách khắc phục: Render modal/dialog qua portal vào `document.body`, quản lý z-index thống nhất và cộng safe area cho action bar.
* **Stats layout lộn xộn:**
  * Vấn đề: `flex-wrap` tự do làm card thống kê rớt dòng không đều, nhất là khi số tiền dài.
  * Cách khắc phục: Dùng grid rõ ràng như `grid grid-cols-2 gap-3 md:grid-cols-4`, thêm `min-w-0`, `truncate` hoặc `tabular-nums`.
* **Keyboard che CTA fixed:**
  * Vấn đề: Action bar fixed đáy bị bàn phím ảo che hoặc đẩy layout khác nhau giữa iOS và Android.
  * Cách khắc phục: Test trên viewport thật, dùng `dvh`, safe area, scroll container đúng và tránh đặt field cuối sát action fixed.

## 12. Nguồn Tham Khảo Chính

* W3C WCAG 2.2: https://www.w3.org/TR/WCAG22/
* W3C WCAG 2.2 Target Size Minimum: https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html
* MDN Responsive Design: https://developer.mozilla.org/en-US/docs/Learn_web_development/Core/CSS_layout/Responsive_Design
* MDN Viewport Meta: https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/meta/name/viewport
* MDN Container Queries: https://developer.mozilla.org/en-US/docs/Web/CSS/Guides/Containment/Container_queries
* MDN CSS Values and Units (dynamic viewport units): https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/Values/length
* MDN prefers-reduced-motion: https://developer.mozilla.org/en-US/docs/Web/CSS/Reference/At-rules/%40media/prefers-reduced-motion
* MDN inputmode: https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Global_attributes/inputmode
* web.dev Web Vitals: https://web.dev/articles/vitals
* MDN Progressive Web Apps: https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps
* MDN Web App Manifest: https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Manifest
* web.dev Service Workers: https://web.dev/learn/pwa/service-workers
* MDN Content Security Policy: https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CSP
* OWASP ASVS: https://owasp.org/www-project-application-security-verification-standard/
* OWASP Session Management Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
* OWASP CSRF Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
* Material Design Accessibility Structure: https://m3.material.io/foundations/designing/structure
