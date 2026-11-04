---
name: ui-shot
description: Chụp ảnh và tự soát lỗi giao diện web app (React/Vite... của bất kỳ project nào, cấu hình qua ui-shot.config.json) trên mobile/tablet/desktop, sáng/tối — tràn ngang, chữ bị cắt/tràn, chồng phần tử, nội dung bị bottom nav/header che, dialog lọt màn hình, khoảng trắng thừa, chữ nhỏ, tương phản thấp, vùng bấm nhỏ, lỗi console/API — rồi NHÌN ảnh để xác nhận. Chụp được từng tab trong trang (--tabs all / theo tên), chờ API tải xong rồi mới chụp. Dùng sau khi sửa/tạo UI (component, trang, tab, dialog, bảng, card, layout), khi người dùng nói "chụp ảnh giao diện", "check UI", "xem trên mobile", "giao diện bị vỡ", "soát lỗi UI", "ui-shot", hoặc trước khi báo xong một thay đổi giao diện.
---

# ui-shot — kiểm chứng giao diện bằng ảnh

Agent không nhìn thấy giao diện khi chỉ đọc code. `ui-shot` mở app thật trong Chrome headless, chụp từng cỡ màn hình/theme, đo DOM để bắt lỗi layout, đánh dấu lỗi lên ảnh và in báo cáo có selector + tên component React. Cách làm học từ `visual-check` của archify: **số liệu tự động trước, rồi bắt buộc nhìn ảnh** — ảnh chụp thành công không có nghĩa giao diện đẹp.

Lệnh: `ui-shot` = `node <thư mục skill này>/scripts/ui-shot.mjs` (nên symlink file đó vào PATH, vd `ln -s <thư mục skill>/scripts/ui-shot.mjs ~/.local/bin/ui-shot`). Chạy **trong thư mục của project** cần chụp (hoặc đặt `UI_SHOT_PROJECT=<thư mục>`). Hướng dẫn đầy đủ + project đang nhận: `ui-shot --help`. Cần Node ≥ 18.

## Cấu hình theo project

Tool tìm `ui-shot.config.json` hoặc `.agents/ui-shot.config.json` từ thư mục hiện tại đi lên; không có thì lấy gốc git, frontend = `frontend/` nếu có `package.json`, ngược lại gốc repo. Mọi trường đều tuỳ chọn (xem `ui-shot --help`):

| Trường | Ý nghĩa | Mặc định |
|---|---|---|
| `frontendDir` | thư mục chứa `package.json`, nơi chạy dev server | `frontend` hoặc `.` |
| `baseUrl`, `devCommand` | địa chỉ app; lệnh bật dev server nếu chưa chạy (`{port}` được thay) | `http://127.0.0.1:5173`, `pnpm exec vite ...` |
| `defaultTargets` | màn hình chụp khi không truyền | `["/"]` |
| `tabs` | `{path, storageKey, names}` cho target `tab:<tên>` (app lưu tab trong localStorage) | không có |
| `tabSelectors` | `{ "<đường dẫn>": "<css nút tab>", "*": ... }` khi tự dò thanh tab trong trang sai (cho `--tabs`) | tự dò |
| `loadingSelectors`, `ignoreRequests` | thêm selector "đang tải" cần chờ tắt; regex URL không chờ (analytics, long-poll...) | có sẵn danh sách mặc định |
| `themeStorageKey` | key localStorage lưu theme (tool còn bật/tắt class `dark` trên `<html>`) | không có |
| `deployCommand`, `deployDir` | lệnh đưa code mới lên app đang chạy (build, xoá cache...) cho `--deploy`/`ui-shot deploy`; chuỗi chạy qua `$SHELL -ic` (dùng được alias), mảng chạy trực tiếp | không có, gốc project |
| `apiPrefix`, `allowedWrites` | request ghi tới `apiPrefix` bị chặn, trừ các regex trong `allowedWrites` | `/api/`, `[]` |
| `login` | `{path, email, password, submit, label?}` selector form đăng nhập; `label` = nhãn ô tài khoản trong `ui-shot login` (mặc định "Email / tên đăng nhập"); không khai báo = không đăng nhập | không có |
| `credentials` | `[{file, email, password}]`: file env + tên biến chứa tài khoản (sau `UI_SHOT_EMAIL`/`UI_SHOT_PASSWORD`); `ui-shot login` ghi vào file đầu tiên | `~/.config/ui-shot/<name>.env` |
| `cookies` | cookie đặt sẵn trước khi tải trang (định dạng `page.setCookie` của puppeteer) | `[]` |
| `uploadUrl` | server upload báo cáo (xem "Upload") | lấy từ `~/.config/ui-shot/config.json` |

Project mới: không cần cài puppeteer — tạo config (mẫu đầy đủ ở cuối `ui-shot --help`), file tài khoản để ngoài repo và `chmod 600`.

Puppeteer dùng chung một bản ở `~/.cache/ui-shot/node_modules` cho mọi project; chưa có thì lần chạy đầu tự `npm install puppeteer` vào đó và tải Chrome (một lần, cần mạng). Cài lỗi (vd mất mạng) thì dùng tạm puppeteer trong `<frontendDir>/node_modules` nếu project có.

Mã nguồn: `scripts/ui-shot.mjs` (điều phối) + `scripts/lib/`: `cli` (tham số, help), `config` (đọc config project), `env` (tài khoản, dev server, Chrome), `page` (chuẩn bị trang, theo dõi request + chờ API/spinner, step, đăng nhập, dò/bấm tab), `audit` (luật đo DOM, chạy trong trình duyệt), `capture` (chụp, cuộn, đánh dấu; nhiều tab trên cùng 1 trang), `auth` (`ui-shot login`: hỏi trong terminal hoặc cửa sổ form), `progress` (log tiến độ ra stderr), `report` (PDF + tóm tắt), `upload`.

## Kết quả mặc định

- Trong lúc chạy, tiến độ từng bước in ra **stderr** dạng `[  9.1s] [1/2] /màn · thiết bị · theme: bước đang làm` (chuẩn bị Chrome, deploy, bật app, đăng nhập, tải trang, step, soát lỗi, cuộn chụp, PDF, upload); stdout chỉ chứa kết quả. Tắt bằng `--quiet`.
- Thiết bị mặc định `mobile-s` (360×740, màn nhỏ nhất nên lộ nhiều lỗi nhất) + `desktop` (1440×900).
- Sinh `report.pdf` nhiều trang (A4 ngang): trang 1 tổng hợp, mỗi thiết bị × theme một phần (ảnh đánh dấu cạnh danh sách lỗi, rồi ảnh từng màn cuộn), kèm `report.json` và các PNG.
- Chưa cấu hình server upload: giữ file trên máy (thư mục tạm `<tmpdir>/ui-shot/...`) và in đường dẫn.
- Có server upload: thử upload cả thư mục (server không giải nén được thì upload `report.pdf` — đã chứa mọi ảnh — và `report.json`), in link, rồi **xoá bản trên máy**. Upload lỗi thì giữ file và exit 2.
- `--keep` giữ file sau khi upload; `--no-upload` không upload; `--out <dir>` lưu vào thư mục chỉ định và luôn giữ; `ui-shot clean` xoá mọi thư mục tạm.

## Upload (tuỳ chọn)

Server lấy theo thứ tự: biến `UI_SHOT_UPLOAD_URL` → `uploadUrl` trong config project → `uploadUrl` trong `~/.config/ui-shot/config.json` (file cá nhân, nằm ngoài skill):

```json
{ "uploadUrl": "https://files.example.com" }
```

Server cần có `POST <url>/upload/dir?name=<tên>` (body tar.gz, trả về text chứa link `.../d/...`; GET link đó trả danh sách link file) và `POST <url>/upload` (header `X-Filename`, trả về text chứa link). Không có server như vậy thì bỏ qua mục này.

## Quy trình sau khi sửa UI (agent)

1. Chọn màn hình bị ảnh hưởng: `tab:<tên>` (nếu project khai báo `tabs`), route (`/login`) hoặc URL. Cần dialog/sheet/menu thì thêm `--step "click:text=<chữ trên nút>"`. Đổi màu/tương phản thì thêm `--themes light,dark`.
2. Trang có nhiều tab trong trang (nút tab, `[role=tab]`, nav-tabs): `ui-shot <màn> --list-tabs` để xem tên/số thứ tự, rồi `--tabs all` hoặc `--tabs "Công nợ Sale,4"` — mỗi thiết bị × theme tải trang 1 lần rồi bấm lần lượt từng tab, mỗi tab 1 kết quả (`<màn> › <tab>`). Dò sai thanh tab thì `--tab-selector <css>` hoặc `tabSelectors` trong config.
   Trước khi chụp tool chờ mọi request XHR/fetch xong + spinner/skeleton (trong màn hình) tắt, tối đa `--wait-api` giây (mặc định 60); hết giờ vẫn chụp và báo `page/api-pending` kèm request còn treo.
3. Chạy kèm `--keep` để còn file PNG mà xem (PNG đọc trực tiếp được, PDF thì không phải công cụ nào cũng render):
   ```bash
   ui-shot tab:orders --themes light,dark --keep
   ui-shot /products --step "click:text=Thêm sản phẩm" --keep
   ```
4. Đọc stdout: `✗` = error (phải sửa), `!` = warn (xem ảnh để quyết), `·` = info. Mỗi mục có `[#n]` khớp số trên `*.annotated.png`, selector, `<Component < Cha>` và toạ độ để tìm code.
5. **Bắt buộc dùng Read xem ảnh** trong thư mục in ở dòng `trên máy:`: mọi `*.annotated.png`, `*.p1.png` và các `*.pN.png` của mobile-s. Nghi ngờ chi tiết thì chụp cận: `--focus "<css>" --dpr 3`.
6. Sửa, chạy lại tới khi hết `✗` và ảnh nhìn ổn. Warn chỉ bỏ qua khi ảnh cho thấy không phải lỗi (ví dụ chữ `…` có chủ đích).
7. Có server upload: lần chạy cuối dùng lệnh mặc định (không `--keep`) để có link báo cáo gửi người dùng, rồi `ui-shot clean` xoá ảnh tạm còn lại. Không có server: gửi đường dẫn `report.pdf` trên máy.
8. Người dùng muốn so sánh trước/sau: chụp bản "trước" bằng `--no-upload --out <scratchpad>/before-<màn>` **trước khi sửa**, bản "sau" vào `after-<màn>`, rồi ghép cặp ảnh cùng màn (cùng thiết bị/theme/trang cuộn) thành một ảnh có nhãn TRƯỚC/SAU. Project có `deployCommand` (`ui-shot --help` dòng `deploy:`) thì app đang chạy KHÔNG tự nhận code mới — thêm `--deploy` cho cả hai lần chụp (lần "trước" để build khớp code gốc, lần "sau" để đưa code đã sửa lên); deploy lỗi thì tool dừng, in log, exit 2 — sửa rồi chạy lại, không chụp tiếp với bản build cũ.

## Báo kết quả

Gửi link `báo cáo PDF` (hoặc đường dẫn `report.pdf` trên máy nếu không upload) cho người dùng. Ghi rõ đã chạy lệnh nào, số lỗi/cảnh báo còn lại, đã xem những ảnh nào và thấy gì. Chưa mở ảnh thì nói "chỉ có kiểm tra tự động, chưa duyệt bằng mắt" — không được nói giao diện ổn chỉ vì `UI-SHOT PASS`.

## Lưu ý

- Chưa có dev server ở `baseUrl` thì tool tự chạy `devCommand` rồi tắt khi xong. Dev server có thể proxy API về backend thật, kể cả **production** → tool **chặn mọi request ghi** tới `apiPrefix` (trừ `allowedWrites`), báo trong `network/blocked-write`. Chỉ dùng `--allow-writes` khi người dùng đồng ý.
- Ảnh có thể chứa dữ liệu thật và link upload là công khai với ai có link.
- Không bấm nút lưu/xóa/gửi trong `--step`; chỉ mở UI (dialog, tab con, menu, dropdown).
- Đăng nhập (khi config có `login`): `UI_SHOT_EMAIL`/`UI_SHOT_PASSWORD` → các file trong `credentials`. Phiên giữ ở `~/.cache/ui-shot/<project>/profile`. Không in credential.
- Thiếu/sai tài khoản (`page/redirected-login`, "thiếu tài khoản") → nhờ người dùng tự chạy `! ui-shot login` (không có terminal nên tool mở cửa sổ form có ô email/tên đăng nhập + mật khẩu; trong terminal thật thì hỏi trực tiếp, mật khẩu ẩn; `--gui` luôn mở form. Ghi file credentials đầu tiên chmod 600, xoá phiên cũ, đăng nhập thử). Agent không tự nhập mật khẩu hộ.
- Máy bật AppArmor chặn sandbox Chrome → tool tự thêm `--no-sandbox` (giống archify).
- Khi có modal mở, tool chỉ soát nội dung trong modal/popup.
- Exit code: 0 không có error, 1 có lỗi UI, 2 lỗi môi trường/tham số/upload.

## Các mã lỗi

| Mã | Mức | Ý nghĩa |
|---|---|---|
| `layout/overflow-x` | error | Trang tràn ngang, phải vuốt ngang; liệt kê phần tử gây tràn |
| `layout/cut-at-edge` | error | Phần tử bị khung `overflow:hidden` cắt ở mép màn hình |
| `layout/covered-by-fixed` | error | Ở đầu/cuối trang, nội dung bị header/bottom nav/nút nổi che (thiếu padding) |
| `text/clipped` | error | Chữ bị cắt mất (khung cố định chiều cao/rộng, không có `…`) |
| `text/invisible` | error | Tương phản < 2:1, gần như không đọc được (hay gặp ở dark mode) |
| `overlay/offscreen` | error | Dialog/popover vượt khỏi màn hình và không cuộn được |
| `media/broken-image`, `page/blank`, `runtime/page-error`, `page/theme-state`, `page/redirected-login`, `tool/step-failed` | error | Ảnh hỏng, trang trắng, lỗi JS, sai theme, chưa đăng nhập, step không chạy được |
| `layout/overlap` | warn | Chữ/nút chồng lên nhau |
| `layout/empty-tail` | warn | Cuộn tới cuối còn khoảng trắng lớn |
| `text/spill` | warn | Chữ dài tràn khỏi khung |
| `text/low-contrast` | warn | Dưới chuẩn WCAG AA (4.5:1, chữ lớn 3:1) |
| `text/too-small`, `touch/small-target` | warn | Chữ < 12px, vùng bấm < 24px trên mobile |
| `page/api-pending` | warn | Hết `--wait-api` mà còn request API chưa xong / spinner chưa tắt — ảnh có thể thiếu dữ liệu |
| `page/still-loading`, `runtime/console-error`, `network/failed` | warn | Skeleton chưa tắt, lỗi console, API lỗi |
| `text/truncated`, `layout/horizontal-scroller`, `network/blocked-write` | info | Chữ rút gọn `…`, vùng cuộn ngang, request ghi bị chặn |
