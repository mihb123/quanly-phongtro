# PWA Assets — cần thay bằng logo thật

Các file icon dưới đây hiện đang là **placeholder** (mượn tạm từ project UpSnap).
Hãy thay bằng logo/branding thật của "Quản lý phòng trọ" — giữ nguyên tên file và
kích thước để không phải sửa `manifest.webmanifest` và `index.html`.

| File                     | Kích thước | Dùng để làm gì |
|--------------------------|------------|----------------|
| `pwa-icon-192.png`       | 192×192    | Icon app trên Android/Chrome (màn hình chính, purpose `any`). |
| `pwa-icon-512.png`       | 512×512    | Icon app độ phân giải cao + dùng dựng splash screen trên Android. |
| `pwa-maskable-192.png`   | 192×192    | Icon adaptive/maskable Android — cần chừa "safe zone" ~20% quanh mép để không bị cắt khi bo tròn. |
| `pwa-maskable-512.png`   | 512×512    | Icon maskable độ phân giải cao (Android). |
| `apple-touch-icon.png`   | 180×180*   | Icon khi "Thêm vào Màn hình chính" trên iOS/Safari. (*hiện là 192×192, iOS tự co lại; nên xuất đúng 180×180.) |
| `favicon.ico`            | —          | Favicon tab trình duyệt (đã có sẵn, không phải placeholder). |

## Chưa có (bổ sung sau nếu muốn)
- **Splash screen iOS ("ảnh khi mở app")**: iOS cần nhiều ảnh `apple-touch-startup-image`
  theo đúng kích thước từng dòng máy (media query riêng). Chưa cấu hình — khi có logo
  thật nên dùng công cụ như [pwa-asset-generator](https://github.com/elegantapp/pwa-asset-generator)
  để sinh trọn bộ icon + splash rồi khai báo vào `index.html`.

## Cách hoạt động
Vite copy toàn bộ `public/` vào `dist/` khi build, backend Go nhúng `dist/` và phục vụ
các file này ở gốc origin (VD `/manifest.webmanifest`). Không cần sửa code backend.
