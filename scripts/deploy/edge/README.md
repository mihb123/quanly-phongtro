# Edge cho quanly-phongtro

Đưa traffic của các domain trong `sites.conf` đi qua một máy chủ ngoài (edge) thay vì Cloudflare, rồi về server nhà bằng tunnel SSH ngược. Lý do: Cloudflare đưa nhà mạng Việt Nam tới colo ở xa và mất gói nặng; upload 788KB từng mất 30–70s. Qua edge `ocl`, upload 3 ảnh CCCD mất dưới 1s.

```
Trình duyệt ──HTTPS──▶ edge: nginx :443 ──▶ 127.0.0.1:<port> (sshd) ══ tunnel SSH ══▶ server nhà: app :<port>
                                                                  ▲
                              server nhà tự mở kết nối SSH ra edge (router nhà không cần mở port)
```

Hiện tại: edge `ocl` (149.118.154.51, Oracle Johor) phục vụ cả `quanly.ptro.site` và `pt.ptro.site`. Sơ đồ chi tiết và số đo: `Documents/infra/edge-architecture.html`.

## Các file

| File | Chạy ở đâu | Vai trò |
|---|---|---|
| `deploy-edge.sh` | server nhà | Lệnh chính: `install`, `cert`, `status`, `remove` |
| `remote-setup.sh` | máy edge (root) | `deploy-edge.sh` đẩy file này qua SSH và chạy; không cần copy tay |
| `sites.conf` | — | Mỗi dòng `domain cổng-app`, ví dụ `quanly.ptro.site 9185` |
| `qlpt-edge-tunnel@.service` | server nhà | Systemd **user** unit dạng template; `install` cài vào `~/.config/systemd/user/` |

## Yêu cầu

Máy edge:
- Debian hoặc Ubuntu, có IP public.
- Có alias trong `~/.ssh/config` của server nhà, đăng nhập bằng key, là root hoặc có `sudo` không mật khẩu.
- Firewall của nhà cung cấp cloud (OCI Security List, AWS Security Group...) mở TCP 80 và 443. Script chỉ tự mở được `ufw`.
- Cổng 443 hoặc do nginx giữ, hoặc còn trống. Nếu một phần mềm khác (Caddy, Apache) đang giữ 443, script sẽ không chạy được.

Server nhà:
- App đang chạy ở các cổng trong `sites.conf`.
- `systemctl --user` hoạt động và đã bật linger (`loginctl show-user $USER -p Linger` ra `Linger=yes`), để tunnel tự chạy khi máy khởi động.
- Có `dig`. Có `cloudflared` nếu cần rollback về Cloudflare.
- `TRUSTED_PROXIES` trong `.env` chứa `127.0.0.1/32`. Kết nối qua tunnel đi vào app từ localhost nên app tin các header `CF-Connecting-IP` và `X-Forwarded-*` mà nginx gửi.

## Triển khai lên một máy edge mới

Ví dụ alias của máy edge là `ocl`. Chạy từ thư mục gốc của repo.

1. **Cài edge và bật tunnel**

   ```bash
   scripts/deploy/edge/deploy-edge.sh install ocl
   ```

   Kết quả mong đợi: mỗi cổng in ra `tunnel 127.0.0.1:<port> -> app: /health 200`. Nếu cổng nào ra `000`, app ở cổng đó chưa chạy (thường gặp với bản dev ở 9180). Chạy lại `install` nhiều lần không sao, nhưng mỗi lần chạy sẽ khởi động lại tunnel, nên các domain đang đi qua edge đó bị 502 vài giây.

2. **Chạy lệnh cert *trước*, rồi mới đổi DNS**

   ```bash
   scripts/deploy/edge/deploy-edge.sh cert ocl pt.ptro.site
   ```

   Lệnh này kiểm tra DNS (qua 1.1.1.1) 5 giây một lần, tối đa `DNS_WAIT_SECONDS` (mặc định 600s). Để nó chạy, rồi sang Cloudflare → `ptro.site` → DNS:
   - Xoá bản ghi CNAME của domain (đang trỏ tới `…cfargotunnel.com`, Proxied).
   - Tạo bản ghi `A`, tên domain, nội dung là IP của edge, Proxy status **DNS only** (mây xám).

   Ngay khi DNS đổi, lệnh tự xin chứng chỉ Let's Encrypt, bật HTTPS và in `https://<domain>/health -> 200`.

   > Đừng đổi DNS trước rồi để lâu mới chạy `cert`. Trong khoảng đó edge chưa có chứng chỉ cho domain, nên trình duyệt sẽ báo lỗi (ngày 2026-10-10 `quanly.ptro.site` đã bị lỗi đúng như vậy).

3. **Kiểm tra**

   ```bash
   scripts/deploy/edge/deploy-edge.sh status
   ```

   Tunnel `qlpt-edge-tunnel@ocl.service` phải ở trạng thái `running`, và mỗi domain đã chuyển phải hiện `edge ocl`.

4. **Chuyển domain production theo cách tương tự.** Nên chuyển `pt.ptro.site` (bản dev) trước để thử, rồi mới tới `quanly.ptro.site`.

## Chuyển sang một máy edge khác

Các bước này không gây gián đoạn, vì có thể chạy tunnel tới nhiều edge cùng lúc:

```bash
scripts/deploy/edge/deploy-edge.sh install <edge-mới>
scripts/deploy/edge/deploy-edge.sh cert <edge-mới> pt.ptro.site       # rồi đổi bản ghi A sang IP mới
scripts/deploy/edge/deploy-edge.sh cert <edge-mới> quanly.ptro.site
scripts/deploy/edge/deploy-edge.sh remove <edge-cũ>
```

`remove` từ chối chạy nếu vẫn còn domain trỏ về edge cũ. Chỉ dùng `FORCE=1` khi bạn chắc chắn muốn gỡ.

## Thêm hoặc bớt domain

1. Sửa `sites.conf`, mỗi dòng `domain cổng`. Nhiều domain có thể dùng chung một cổng.
2. Chạy `deploy-edge.sh install <edge>`. Lệnh này cập nhật cổng user tunnel được mở, tham số `-R` của tunnel và cấu hình nginx.
3. Với domain mới, làm tiếp bước 2 ở phần trên (`cert` rồi đổi DNS).

## Rollback về Cloudflare

cloudflared vẫn chạy trên server nhà, nên chỉ cần trả DNS về tunnel:

```bash
cloudflared tunnel route dns --overwrite-dns 8031e4bd-97d2-4847-a6d6-5c57b2d9475d quanly.ptro.site
```

Hoặc làm trong dashboard: xoá bản ghi A, tạo lại CNAME `<tên>` → `8031e4bd-97d2-4847-a6d6-5c57b2d9475d.cfargotunnel.com`, Proxied. Máy nào còn cache DNS cũ thì vẫn đi qua edge tối đa khoảng 5 phút, và vẫn chạy được.

## Script thay đổi những gì

Trên máy edge (`install`/`cert`):

| Đường dẫn / thành phần | Nội dung |
|---|---|
| gói `nginx`, `certbot`, `curl` | Chỉ cài nếu chưa có |
| user `qlpt-tunnel` (`/var/lib/qlpt-tunnel`) | Không có shell; `authorized_keys` có `restrict`, chỉ cho mở `permitlisten="127.0.0.1:<port>"` |
| `/etc/ssh/sshd_config.d/60-qlpt-tunnel.conf` | `Match User qlpt-tunnel`: không đăng nhập bằng mật khẩu, không TTY, chỉ forward chiều remote, ngắt phiên chết sau 45s. Script kiểm tra cấu hình không ảnh hưởng tới user khác; nếu có thì tự gỡ |
| `ufw allow 80/tcp`, `443/tcp` | Chỉ thêm khi ufw đang bật |
| `/etc/nginx/snippets/qlpt-edge-proxy.conf` | Header gửi về app; ghi đè `CF-Connecting-IP`, `X-Forwarded-*` bằng IP thật của client; xoá `True-Client-IP` và `CF-Visitor` để client không giả mạo được |
| `/etc/nginx/sites-available/qlpt-edge.conf` (+ symlink) | Cổng 80: phục vụ ACME rồi redirect sang HTTPS. Cổng 443: TLS, gzip, upstream keepalive, `/assets/` cache 1 năm, body tối đa 20MB, không buffer request |
| `/var/www/qlpt-acme` | Webroot để Let's Encrypt kiểm tra |
| `/etc/letsencrypt/live/<domain>/` | Chứng chỉ; `certbot.timer` tự gia hạn và reload nginx |

Script luôn chạy `nginx -t` trước khi reload; nếu lỗi thì trả lại cấu hình cũ, nên các site khác trên máy edge không bị ảnh hưởng.

Trên server nhà:

| Đường dẫn | Nội dung |
|---|---|
| `~/.ssh/qlpt_tunnel_ed25519` | Key riêng cho tunnel; `install` tự tạo nếu chưa có |
| `~/.config/systemd/user/qlpt-edge-tunnel@.service` | Unit template |
| `~/.config/qlpt-edge/<edge>.env` | `EDGE_HOST`, `TUNNEL_USER`, `TUNNEL_KEY`, `FORWARDS` |
| service `qlpt-edge-tunnel@<edge>` | Đã enable, tự kết nối lại sau 5s, ping server mỗi 15s |

`remove` gỡ site nginx, snippet, webroot, user tunnel, cấu hình sshd ở máy edge và tắt tunnel ở máy nhà. Chứng chỉ trong `/etc/letsencrypt` và các gói đã cài thì giữ nguyên.

## Máy edge có SNI router (như `ocl`)

Nếu `/etc/nginx/stream-enabled/*.conf` có `ssl_preread on` (cổng 443 chỉ đọc tên miền rồi chuyển nguyên gói TLS sang các backend), script tự:
- nghe ở backend `default` của router (trên `ocl` là `127.0.0.1:8444 ssl proxy_protocol`), không sửa file router;
- lấy IP client từ `$proxy_protocol_addr`.

Máy không có router thì script nghe thẳng `443 ssl http2` và lấy IP client từ `$remote_addr`.

Nếu tự nhận diện sai, đặt biến ghi đè **mỗi lần** chạy `install` và `cert`:

```bash
EDGE_TLS_LISTEN='127.0.0.1:8445 ssl proxy_protocol' EDGE_CLIENT_IP='$proxy_protocol_addr' \
  scripts/deploy/edge/deploy-edge.sh install <edge>
```

## Biến môi trường

| Biến | Mặc định | Dùng cho |
|---|---|---|
| `TUNNEL_USER` | `qlpt-tunnel` | Tên user tunnel trên máy edge |
| `TUNNEL_KEY` | `~/.ssh/qlpt_tunnel_ed25519` | Key SSH của tunnel |
| `DNS_WAIT_SECONDS` | `600` | `cert` chờ DNS tối đa bao lâu |
| `FORCE` | — | `FORCE=1` cho phép `remove` khi domain vẫn trỏ về edge đó |
| `EDGE_TLS_LISTEN`, `EDGE_CLIENT_IP` | tự nhận diện | Ghi đè chế độ nghe TLS và nguồn lấy IP client |

## Vận hành và xử lý sự cố

```bash
scripts/deploy/edge/deploy-edge.sh status
systemctl --user status qlpt-edge-tunnel@ocl
journalctl --user -u qlpt-edge-tunnel@ocl -f
ssh ocl 'sudo tail -f /var/log/nginx/access.log /var/log/nginx/error.log'
```

| Triệu chứng | Nguyên nhân thường gặp | Cách xử lý |
|---|---|---|
| Trình duyệt báo sai chứng chỉ (ví dụ chứng chỉ của `nc.mihb.site`) | DNS đã trỏ về edge nhưng chưa chạy `cert` | `deploy-edge.sh cert <edge> <domain>` |
| 502 Bad Gateway | Tunnel đứt, hoặc app ở cổng đó không chạy | Xem `status` và journal của tunnel; kiểm tra `curl http://127.0.0.1:<port>/health` trên máy nhà |
| Tunnel liên tục khởi động lại với lỗi `remote port forwarding failed` | Phiên SSH cũ trên edge vẫn giữ cổng | Tự hết sau khoảng 45s (`ClientAliveInterval 15` × 3) |
| `install` báo `tunnel chưa mở 127.0.0.1:<port>` | Key chưa được nhận, hoặc sshd chưa reload | `journalctl --user -u qlpt-edge-tunnel@<edge>`; thử `ssh -i ~/.ssh/qlpt_tunnel_ed25519 qlpt-tunnel@<ip>` |
| `cert` chờ mãi | DNS chưa đổi, hoặc vẫn để Proxied (mây cam) | Bản ghi phải là `A` → IP edge, chọn DNS only |
| certbot báo lỗi xác thực | Cổng 80 bị chặn ở firewall cloud | Mở TCP 80 trong Security List/Security Group |
| Đăng nhập lỗi 401 DPoP, hoặc mọi request có chung một IP | App không tin header từ proxy | `TRUSTED_PROXIES` phải chứa `127.0.0.1/32` |
| `nginx -t thất bại, đã trả lại cấu hình cũ` | Cấu hình sinh ra xung đột với site có sẵn | Đọc 3 dòng lỗi script in ra; đặt `EDGE_TLS_LISTEN` nếu xung đột ở `listen` |
