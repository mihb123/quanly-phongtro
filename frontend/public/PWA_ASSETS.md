# PWA Assets

Bộ icon dùng biểu tượng ngôi nhà trắng tối giản (`Home09` của Hugeicons, đồng bộ
với logo trong `MobileHeader`) trên nền sky. Hình nhà đơn giản giúp icon dễ nhận
ra khi hiển thị nhỏ trên màn hình chính, favicon và thanh trạng thái.

## Nhận diện

- Sky `#0069a8`: màu thương hiệu chính, lấy từ palette sky của ứng dụng; tạo cảm
  giác tin cậy, sạch và hiện đại.
- Off-white `#f8fafc`: nét ngôi nhà có độ tương phản cao.
- `pwa-logo.svg`: bản vector master do script xuất để xem/đối chiếu. Không sửa
  riêng file này hoặc các file PNG vì lần xuất tiếp theo sẽ ghi đè chúng.

Logo `any` có khung bo tròn và khoảng thở; logo `maskable` dùng nền tràn viền và
thu nhỏ mark để toàn bộ chi tiết quan trọng nằm trong safe zone của Android.

| File                     | Kích thước | Dùng để làm gì |
|--------------------------|------------|----------------|
| `pwa-icon-192.png`       | 192×192    | Icon app trên Android/Chrome (màn hình chính, purpose `any`). |
| `pwa-icon-512.png`       | 512×512    | Icon app độ phân giải cao + dùng dựng splash screen trên Android. |
| `pwa-maskable-192.png`   | 192×192    | Icon adaptive/maskable Android — cần chừa "safe zone" ~20% quanh mép để không bị cắt khi bo tròn. |
| `pwa-maskable-512.png`   | 512×512    | Icon maskable độ phân giải cao (Android). |
| `apple-touch-icon.png`   | 180×180    | Icon khi "Thêm vào Màn hình chính" trên iOS/Safari. |
| `favicon.ico`            | 64×64      | Favicon tab trình duyệt, ICO chứa PNG để giữ alpha và cạnh sắc. |

## Xuất lại asset

Chạy từ thư mục `frontend/`:

```bash
pnpm assets:pwa
```

Script `scripts/generate-pwa-assets.js` xuất toàn bộ kích thước từ cùng một cấu
trúc SVG bằng Chromium/Puppeteer. Khi đổi màu hoặc hình, cập nhật template trong
script rồi chạy lại; giữ nguyên tên file để không phải sửa `manifest.webmanifest`
và `index.html`.

Trình duyệt cache favicon rất lâu. Khi thay thiết kế favicon, tăng giá trị `v`
trong URL `/favicon.ico?v=...` ở `index.html` để buộc trình duyệt tải file mới.

## Chưa có (bổ sung sau nếu muốn)
- **Splash screen iOS ("ảnh khi mở app")**: iOS cần nhiều ảnh `apple-touch-startup-image`
  theo đúng kích thước từng dòng máy (media query riêng). Chưa cấu hình — khi có logo
  thật nên dùng công cụ như [pwa-asset-generator](https://github.com/elegantapp/pwa-asset-generator)
  để sinh trọn bộ icon + splash rồi khai báo vào `index.html`.

## Cách hoạt động
Vite copy toàn bộ `public/` vào `dist/` khi build, backend Go nhúng `dist/` và phục vụ
các file này ở gốc origin (VD `/manifest.webmanifest`). Không cần sửa code backend.
