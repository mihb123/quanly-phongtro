import path from 'node:path'
import { CONFIG_NAMES, SHARED_MODULES_DIR, DEFAULT_BASE, DEFAULT_DEVICES, DEVICES, OUT_ROOT, PROFILE_DIR, PROJECT, UPLOAD_BASE, USER_CONFIG_FILE } from './config.mjs'

const TAB_NAMES = PROJECT.tabs?.names || []

export const HELP = `ui-shot — chụp ảnh giao diện web app trên mobile/desktop (sáng/tối), tự soát lỗi UI
(tràn ngang, chữ bị cắt/tràn, phần tử chồng nhau, nội dung bị thanh cố định che, dialog lọt màn hình,
khoảng trắng thừa, ảnh hỏng, chữ nhỏ, tương phản thấp, vùng bấm nhỏ, lỗi console/API),
gom tất cả vào một báo cáo PDF nhiều trang${UPLOAD_BASE ? ` rồi tải lên ${UPLOAD_BASE} và in link` : ' (lưu trên máy; cấu hình server upload để có link, xem UPLOAD)'}.

CÁCH DÙNG
  ui-shot [màn-hình...] [tuỳ-chọn]        gọi được từ bất kỳ thư mục nào
  ui-shot clean                           xoá mọi ảnh tạm còn lại trong ${OUT_ROOT}
  ui-shot deploy                          chỉ chạy "deployCommand" của project (build/xoá cache sau khi sửa code)
  ui-shot login [--email <email>] [--gui] [--no-verify]
                                          nhập tài khoản đăng nhập app (mật khẩu ẩn), lưu vào file credentials;
                                          không có terminal (vd "! ui-shot login" trong Claude Code) hoặc --gui
                                          thì mở cửa sổ form có ô email/tên đăng nhập + mật khẩu để điền
                                          đầu tiên (chmod 600), xoá phiên cũ rồi thử đăng nhập để kiểm tra

PROJECT HIỆN TẠI
  ${PROJECT.name} — ${PROJECT.root}
  cấu hình: ${PROJECT.configFile || `(không có — dùng mặc định; tạo ${CONFIG_NAMES.join(' hoặc ')})`}
  frontend: ${PROJECT.frontendDir}
  puppeteer: dùng chung ở ${SHARED_MODULES_DIR}/node_modules (chưa có thì tự cài lần đầu)
  upload:   ${UPLOAD_BASE || '(chưa cấu hình — chỉ lưu trên máy)'}
  deploy:   ${PROJECT.deployCommand ? `${Array.isArray(PROJECT.deployCommand) ? PROJECT.deployCommand.join(' ') : PROJECT.deployCommand} (trong ${PROJECT.deployDir})` : '(không có "deployCommand")'}

MÀN HÌNH (mặc định ${PROJECT.defaultTargets.join(' ')}, truyền nhiều cái cùng lúc được)
  tab:<tên>        tab lưu trong localStorage (khai báo "tabs" trong config)${TAB_NAMES.length ? `: ${TAB_NAMES.join(', ')}` : ''}
  /duong-dan       route của app, vd /login, /settings
  https://...      URL đầy đủ, vd bản production

VÍ DỤ (giả sử config khai báo tabs "dashboard", "orders", "settings")
  ui-shot                                     màn hình mặc định trên mobile-s + desktop, giao diện sáng
  ui-shot tab:orders --open                   chụp tab orders rồi mở báo cáo PDF
  ui-shot tab:orders tab:settings --themes light,dark
  ui-shot /products --step "click:text=Thêm sản phẩm"
                                              mở dialog "Thêm sản phẩm" rồi chụp
  ui-shot tab:orders --devices mobile-s --dpr 3 --focus "[data-slot=card]"
                                              chụp cận (phóng to) card đầu tiên
  ui-shot /login --no-login --devices 320x640m
  ui-shot tab:dashboard --no-upload --out ./anh-ui
                                              không upload, giữ ảnh trong thư mục hiện tại

TUỲ CHỌN
  --devices a,b      ${Object.entries(DEVICES).map(([k, v]) => `${k} ${v.width}x${v.height}`).join(', ')}
                     hoặc tự đặt WxH (desktop) / WxHm (mobile, có touch). Mặc định: ${DEFAULT_DEVICES.join(',')}
                     (mobile-s là màn nhỏ nhất nên lộ nhiều lỗi nhất)
  --themes a,b       light, dark. Mặc định: light
  --step <hành-động> lặp được, chạy theo thứ tự sau khi trang tải xong (mở dialog, menu, tab con...):
                       click:text=<chữ trên nút>   click:<css>   type:<css>=<giá trị>   press:<Phím>
                       hover:<css>   scroll:<css>   waitfor:<css>   wait:<ms>
  --focus <css>      chụp cận phần tử đầu tiên khớp selector (lặp được; nên kèm --dpr 2 hoặc 3)
  --storage k=v      ghi localStorage trước khi tải trang (lặp được)
  --pages <n>        số màn hình cuộn tối đa mỗi lần chụp (mặc định mobile 4, desktop 2)
  --dpr <n>          độ nét ảnh (deviceScaleFactor), mặc định 1
  --base <url>       địa chỉ app, mặc định ${DEFAULT_BASE} (tự chạy devCommand nếu chưa chạy, tắt khi xong)
  --out <dir>        lưu vào thư mục này và giữ lại (mặc định ${OUT_ROOT}/<thời-gian>-<màn-hình>, tự xoá sau upload)
  --deploy           chạy "deployCommand" trước khi chụp (sau khi sửa code, để ảnh phản ánh code mới)
  --keep             giữ file trên máy sau khi upload (mặc định upload xong là xoá)
  --no-upload        không upload, giữ file trên máy (mặc định khi chưa cấu hình server upload)
  --open             mở báo cáo PDF khi xong (link đã upload, hoặc file trên máy)
  --no-login         không tự đăng nhập (màn hình công khai)
  --allow-writes     cho phép request ghi (POST/PUT/PATCH/DELETE) tới ${PROJECT.apiPrefix} — mặc định CHẶN để không
                     sửa dữ liệu thật khi bấm thử; chỉ các URL trong "allowedWrites" (đăng nhập...) được đi qua
  --json             in toàn bộ báo cáo JSON ra stdout
  --tabs <ds>        chụp từng tab trong trang: "all" = mọi tab; hoặc tên/số thứ tự cách nhau dấu phẩy,
                     vd --tabs all · --tabs "Dashboard,Công nợ Sale" · --tabs 1,3 (tên khớp đúng hoặc chứa)
  --list-tabs        chỉ dò và in danh sách tab của màn hình (không chụp) — để biết tên/số thứ tự
  --tab-selector <css>  selector các nút tab khi tự dò sai (hoặc khai báo "tabSelectors" trong config)
  --wait-api <giây>  thời gian tối đa chờ API (XHR/fetch) tải xong + spinner/skeleton tắt trước khi chụp,
                     mặc định 60; hết giờ vẫn chụp và báo cảnh báo page/api-pending.
                     Có --tabs: mỗi thiết bị × theme tải trang 1 lần rồi bấm lần lượt từng tab
  -q, --quiet        không in tiến độ từng bước (mặc định in ra stderr: [thời gian] bước đang chạy)
  -h, --help         xem hướng dẫn này

KẾT QUẢ
  report.pdf                             báo cáo nhiều trang: trang 1 tổng hợp lỗi, mỗi thiết bị × theme
                                         một phần riêng (danh sách lỗi, ảnh đánh dấu, ảnh từng màn cuộn)
  <màn-hình>.<thiết-bị>.<theme>.pN.png   ảnh từng màn hình cuộn (p1 = đầu trang)
  *.annotated.png                        khung đỏ = lỗi, vàng = cảnh báo, số [#n] khớp với báo cáo
  *.focusN.png                           ảnh cận cảnh từ --focus
  report.json                            dữ liệu đầy đủ (selector, component React, toạ độ)
  Chưa cấu hình server upload: giữ file trên máy và in đường dẫn.

UPLOAD (tuỳ chọn)
  Server lấy theo thứ tự: biến UI_SHOT_UPLOAD_URL → "uploadUrl" trong config project
  → "uploadUrl" trong ${USER_CONFIG_FILE}. Có server thì upload cả thư mục, in link (PDF + thư mục)
  rồi xoá bản trên máy; upload lỗi thì giữ file và báo đường dẫn.
  Server cần hỗ trợ: POST <url>/upload/dir?name=<tên> (body tar.gz, trả về text chứa link .../d/...;
  GET link đó trả danh sách link file) và POST <url>/upload (header X-Filename, trả về text chứa link).
  Link upload thường công khai với ai có link — ảnh có thể chứa dữ liệu thật.

SO SÁNH TRƯỚC/SAU KHI SỬA CODE
  "deployCommand" là lệnh đưa code mới lên app đang chạy (build frontend, xoá cache...). Dạng chuỗi chạy qua
  $SHELL -ic nên dùng được alias của bạn; dạng mảng chạy trực tiếp. Chạy trong "deployDir" (mặc định gốc project).
  Output ghi vào ${OUT_ROOT}-deploy-<project>.log; lệnh lỗi thì in 20 dòng cuối và thoát mã 2.
    ui-shot tab:x --deploy --no-upload --out /tmp/before-x    trước khi sửa
    ui-shot tab:x --deploy --no-upload --out /tmp/after-x     sau khi sửa

ĐĂNG NHẬP (chỉ khi config có "login")
  Lấy tài khoản theo thứ tự: biến môi trường UI_SHOT_EMAIL + UI_SHOT_PASSWORD → các file trong "credentials"
  ${PROJECT.credentials.map((c) => `${c.file} (${c.email}/${c.password})`).join(' → ') || '(chưa khai báo)'}.
  Phiên đăng nhập được giữ ở ${PROFILE_DIR} nên các lần sau không cần đăng nhập lại.
  Đặt/đổi tài khoản: ui-shot login (ghi vào ${PROJECT.credentials[0].file}).
  Không có terminal: nếu stdin có dữ liệu thì đọc (dòng 1 email, dòng 2 mật khẩu), vd
    printf '%s\\n%s\\n' "$EMAIL" "$PASS" | ui-shot login
  stdin rỗng thì mở cửa sổ form để điền (cần màn hình). Nhãn ô tài khoản: "login.label" trong config.

CẤU HÌNH PROJECT (${CONFIG_NAMES.join(' hoặc ')} ở gốc repo, tìm từ thư mục hiện tại đi lên;
đặt UI_SHOT_PROJECT=<thư mục> để chọn project khác). Mọi trường đều tuỳ chọn:
  { "name": "my-app", "frontendDir": "frontend", "baseUrl": "http://127.0.0.1:5173",
    "devCommand": ["pnpm", "exec", "vite", "--host", "127.0.0.1", "--port", "{port}", "--strictPort"],
    "deployCommand": "npm run build", "deployDir": ".", "uploadUrl": "https://files.example.com",
    "defaultTargets": ["/"], "themeStorageKey": "color-mode", "apiPrefix": "/api/",
    "allowedWrites": ["/api/v1/auth/login$"],
    "tabs": { "path": "/", "storageKey": "home_active_tab", "names": ["dashboard", "settings"] },
    "login": { "path": "/login", "email": "#email", "password": "#password", "submit": "button[type=\\"submit\\"]" },
    "credentials": [{ "file": "~/.config/my-app/ui-shot.env", "email": "UI_SHOT_EMAIL", "password": "UI_SHOT_PASSWORD" }],
    "cookies": [{ "name": "locale", "value": "vi", "url": "http://127.0.0.1:5173" }] }

MÃ THOÁT
  0 không có lỗi mức error · 1 có lỗi UI · 2 lỗi môi trường/tool (không mở được app, upload lỗi, sai tham số...)
`

export function parseArgs(argv) {
  const opts = {
    command: 'shot', targets: [], devices: [...DEFAULT_DEVICES], themes: ['light'], steps: [], focus: [], storage: {},
    pages: null, dpr: 1, base: process.env.UI_SHOT_BASE || DEFAULT_BASE, baseExplicit: Boolean(process.env.UI_SHOT_BASE),
    out: null, deploy: false, login: true, allowWrites: false, json: false, open: false, upload: Boolean(UPLOAD_BASE), keep: !UPLOAD_BASE,
    email: null, verify: true, gui: false, quiet: false, tabs: null, listTabs: false, tabSelector: null, waitApi: 60,
  }
  if (argv[0] === 'clean') return { ...opts, command: 'clean' }
  if (argv[0] === 'deploy') return { ...opts, command: 'deploy' }
  if (argv[0] === 'login') {
    opts.command = 'login'
    argv = argv.slice(1)
  }
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i]
    const next = () => {
      const value = argv[++i]
      if (value === undefined) throw new Error(`Thiếu giá trị cho ${arg}`)
      return value
    }
    switch (arg) {
      case '-h': case '--help': console.log(HELP); process.exit(0); break
      case '--devices': opts.devices = next().split(',').map((s) => s.trim()).filter(Boolean); break
      case '--themes': opts.themes = next().split(',').map((s) => s.trim()).filter(Boolean); break
      case '--step': opts.steps.push(next()); break
      case '--focus': opts.focus.push(next()); break
      case '--storage': {
        const raw = next()
        const eq = raw.indexOf('=')
        if (eq < 1) throw new Error(`--storage cần dạng key=value: ${raw}`)
        opts.storage[raw.slice(0, eq)] = raw.slice(eq + 1)
        break
      }
      case '--pages': opts.pages = Math.max(1, Number.parseInt(next(), 10) || 1); break
      case '--dpr': opts.dpr = Math.max(1, Number(next()) || 1); break
      case '--base': opts.base = next().replace(/\/$/, ''); opts.baseExplicit = true; break
      case '--out': opts.out = path.resolve(next()); break
      case '--deploy': opts.deploy = true; break
      case '--keep': opts.keep = true; break
      case '--no-upload': opts.upload = false; opts.keep = true; break
      case '--no-login': opts.login = false; break
      case '--allow-writes': opts.allowWrites = true; break
      case '--json': opts.json = true; break
      case '-q': case '--quiet': opts.quiet = true; break
      case '--tabs': opts.tabs = next().split(',').map((s) => s.trim()).filter(Boolean); break
      case '--list-tabs': opts.listTabs = true; break
      case '--tab-selector': opts.tabSelector = next(); break
      case '--wait-api': opts.waitApi = Math.max(1, Number(next()) || 60); break
      case '--open': opts.open = true; break
      case '--email': opts.email = next().trim(); break
      case '--no-verify': opts.verify = false; break
      case '--gui': opts.gui = true; break
      default:
        if (arg.startsWith('--')) throw new Error(`Tuỳ chọn không hỗ trợ: ${arg}`)
        opts.targets.push(arg)
    }
  }
  if (opts.command === 'login') {
    if (opts.targets.length) throw new Error(`ui-shot login không nhận màn hình: ${opts.targets.join(' ')}`)
    return opts
  }
  if (opts.email || !opts.verify || opts.gui) throw new Error('--email, --gui và --no-verify chỉ dùng với ui-shot login')
  if (!opts.targets.length) opts.targets.push(...PROJECT.defaultTargets)
  for (const theme of opts.themes) {
    if (!['light', 'dark'].includes(theme)) throw new Error(`Theme không hợp lệ: ${theme}`)
  }
  return opts
}

export function resolveDevice(name) {
  if (DEVICES[name]) return { name, ...DEVICES[name] }
  const match = /^(\d{3,4})x(\d{3,4})(m?)$/.exec(name)
  if (!match) throw new Error(`Thiết bị không hợp lệ: ${name}`)
  return { name, width: Number(match[1]), height: Number(match[2]), mobile: match[3] === 'm' }
}

export function resolveTarget(raw, base) {
  if (raw.startsWith('tab:')) {
    const tab = raw.slice(4)
    if (!PROJECT.tabs) throw new Error('Project chưa khai báo "tabs" trong ui-shot.config.json')
    if (TAB_NAMES.length && !TAB_NAMES.includes(tab)) throw new Error(`Tab không hợp lệ: ${tab} (có: ${TAB_NAMES.join(', ')})`)
    return { raw, slug: `tab-${tab}`, url: `${base}${PROJECT.tabs.path || '/'}`, storage: { [PROJECT.tabs.storageKey]: tab } }
  }
  if (/^https?:\/\//.test(raw)) return { raw, slug: slugify(new URL(raw).pathname || 'root'), url: raw, storage: {} }
  if (raw.startsWith('/')) return { raw, slug: slugify(raw) || 'root', url: `${base}${raw}`, storage: {} }
  throw new Error(`Target không hợp lệ: ${raw}`)
}

export function slugify(value) {
  return value.normalize('NFD').replace(/[\u0300-\u036f]/g, '').replace(/đ/g, 'd').replace(/Đ/g, 'D')
    .replace(/^\/+|\/+$/g, '').replace(/[^a-zA-Z0-9_-]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 60)
}

export function timestamp() {
  const d = new Date()
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}`
}

