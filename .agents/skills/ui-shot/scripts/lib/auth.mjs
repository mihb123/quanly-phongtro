import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import readline from 'node:readline'
import { PROFILE_DIR, PROJECT } from './config.mjs'
import { readEnvFile, resolvePuppeteer, sandboxBlocked } from './env.mjs'

const FORM_TIMEOUT_MS = 10 * 60 * 1000

function ask(question, { hidden = false } = {}) {
  return new Promise((resolve) => {
    const rl = readline.createInterface({ input: process.stdin, output: process.stderr, terminal: true })
    let muted = false
    rl._writeToOutput = (s) => { if (!muted) process.stderr.write(s) }
    process.stderr.write(question)
    muted = hidden
    rl.question('', (answer) => {
      if (hidden) process.stderr.write('\n')
      rl.close()
      resolve(answer)
    })
  })
}

// Đọc stdin khi không có terminal. Trả null nếu stdin rỗng (/dev/null, `! ui-shot login` trong
// Claude Code...) hoặc không có dữ liệu nào trong waitMs — khi đó mở form thay vì treo chờ.
function readPipedLines(waitMs = 1500) {
  return new Promise((resolve) => {
    let data = ''
    let finished = false
    const finish = (value) => {
      if (finished) return
      finished = true
      clearTimeout(timer)
      resolve(value)
    }
    const timer = setTimeout(() => {
      if (data) return
      process.stdin.destroy()
      finish(null)
    }, waitMs)
    process.stdin.setEncoding('utf8')
    process.stdin.on('data', (chunk) => { data += chunk })
    process.stdin.on('end', () => finish(data ? data.split(/\r?\n/) : null))
    process.stdin.on('error', () => finish(null))
  })
}

function escapeHtml(value) {
  return String(value ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c])
}

function formHtml({ label, email, hasPassword, file, base }) {
  return `<!doctype html><html lang="vi"><head><meta charset="utf-8"><title>ui-shot login — ${escapeHtml(PROJECT.name)}</title>
<style>
  :root { color-scheme: light dark; --bg:#f6f7f9; --card:#fff; --text:#1f2933; --soft:#5f6b7a; --line:#d5dae1; --accent:#2563eb; --err:#b91c1c; }
  @media (prefers-color-scheme: dark) { :root { --bg:#14171c; --card:#1d2128; --text:#e6e9ee; --soft:#9aa4b2; --line:#343a44; --accent:#60a5fa; --err:#f87171; } }
  * { box-sizing: border-box; }
  body { margin:0; min-height:100vh; display:grid; place-items:center; background:var(--bg); color:var(--text); font:15px/1.45 system-ui, sans-serif; }
  form { width:min(400px, calc(100vw - 32px)); background:var(--card); border:1px solid var(--line); border-radius:12px; padding:24px; }
  h1 { margin:0 0 4px; font-size:18px; }
  p.meta { margin:0 0 18px; color:var(--soft); font-size:13px; word-break:break-all; }
  label { display:block; margin:14px 0 6px; font-weight:600; font-size:14px; }
  input[type=text], input[type=password] { width:100%; height:42px; padding:0 12px; border:1px solid var(--line); border-radius:8px; background:transparent; color:inherit; font:inherit; }
  input:focus { outline:2px solid var(--accent); outline-offset:1px; }
  .row { display:flex; align-items:center; gap:8px; margin-top:10px; color:var(--soft); font-size:13px; }
  .hint { color:var(--soft); font-size:12px; margin-top:4px; }
  .err { color:var(--err); font-size:13px; min-height:18px; margin-top:12px; }
  button { margin-top:8px; width:100%; height:44px; border:0; border-radius:8px; background:var(--accent); color:#fff; font:inherit; font-weight:600; cursor:pointer; }
</style></head><body>
<form id="f" autocomplete="off">
  <h1>Tài khoản đăng nhập ${escapeHtml(PROJECT.name)}</h1>
  <p class="meta">App: ${escapeHtml(base)}<br>Lưu vào: ${escapeHtml(file)} (chmod 600)</p>
  <label for="email">${escapeHtml(label)}</label>
  <input id="email" type="text" value="${escapeHtml(email)}" spellcheck="false" autocapitalize="off">
  <label for="password">Mật khẩu</label>
  <input id="password" type="password">
  ${hasPassword ? '<div class="hint">Để trống = giữ mật khẩu đã lưu.</div>' : ''}
  <div class="row"><input id="show" type="checkbox"><label for="show" style="margin:0;font-weight:400">Hiện mật khẩu</label></div>
  <div class="err" id="err"></div>
  <button type="submit">Lưu tài khoản</button>
</form>
<script>
  const $ = (id) => document.getElementById(id)
  $(${JSON.stringify(email ? 'password' : 'email')}).focus()
  $('show').onchange = () => { $('password').type = $('show').checked ? 'text' : 'password' }
  $('f').onsubmit = async (e) => {
    e.preventDefault()
    const email = $('email').value.trim()
    const password = $('password').value
    if (!email) return ($('err').textContent = 'Nhập ${escapeHtml(label).toLowerCase()}.')
    if (!password && !${hasPassword}) return ($('err').textContent = 'Nhập mật khẩu.')
    $('err').textContent = ''
    document.querySelector('button').textContent = 'Đã lưu, đang đóng…'
    await window.uiShotSubmit(email, password)
  }
</script></body></html>`
}

// Mở một cửa sổ Chrome (puppeteer, có giao diện) chứa form email + mật khẩu.
// Trả { email, password } hoặc null nếu người dùng đóng cửa sổ.
async function askInWindow({ label, email, hasPassword, file, base }) {
  if (!process.env.DISPLAY && !process.env.WAYLAND_DISPLAY) {
    throw new Error('không có terminal và không có màn hình (DISPLAY/WAYLAND_DISPLAY) để mở form. '
      + 'Chạy trong terminal: ui-shot login, hoặc qua stdin: printf \'%s\\n%s\\n\' "$EMAIL" "$PASS" | ui-shot login')
  }
  const puppeteer = await resolvePuppeteer()
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-shot-login-form-'))
  const args = ['--app=data:text/html,', '--window-size=480,600', '--ozone-platform-hint=auto', '--no-first-run', '--no-default-browser-check']
  if (sandboxBlocked()) args.push('--no-sandbox')
  console.error('Đã mở cửa sổ "ui-shot login" — điền tài khoản rồi bấm Lưu (đóng cửa sổ = huỷ).')
  const browser = await puppeteer.launch({ headless: false, userDataDir: profile, defaultViewport: null, args, ignoreDefaultArgs: ['--enable-automation'] })
  try {
    let done
    const result = new Promise((resolve) => { done = resolve })
    const timer = setTimeout(() => done('timeout'), FORM_TIMEOUT_MS)
    browser.on('disconnected', () => done(null))
    const page = (await browser.pages())[0] || await browser.newPage()
    await page.exposeFunction('uiShotSubmit', (e, p) => { done({ email: e, password: p }); return true })
    await page.setContent(formHtml({ label, email, hasPassword, file, base }))
    const value = await result
    clearTimeout(timer)
    if (value === 'timeout') throw new Error('hết 10 phút chưa điền form')
    return value
  } finally {
    await browser.close().catch(() => {})
    fs.rmSync(profile, { recursive: true, force: true })
  }
}

function quote(value) {
  return value.includes("'") ? value : `'${value}'`
}

export function writeCredentials(source, email, password) {
  const lines = fs.existsSync(source.file) ? fs.readFileSync(source.file, 'utf8').split('\n') : []
  const values = { [source.email]: email, [source.password]: password }
  const pending = new Set(Object.keys(values))
  const out = lines.map((line) => {
    const key = /^\s*([A-Z0-9_]+)\s*=/.exec(line)?.[1]
    if (!key || !pending.has(key)) return line
    pending.delete(key)
    return `${key}=${quote(values[key])}`
  })
  while (out.length && out[out.length - 1] === '') out.pop()
  for (const key of pending) out.push(`${key}=${quote(values[key])}`)
  fs.mkdirSync(path.dirname(source.file), { recursive: true, mode: 0o700 })
  const tmp = `${source.file}.tmp-${process.pid}`
  fs.writeFileSync(tmp, `${out.join('\n')}\n`, { mode: 0o600 })
  fs.renameSync(tmp, source.file)
  fs.chmodSync(source.file, 0o600)
}

export async function promptCredentials(opts) {
  if (!PROJECT.login) throw new Error('Project chưa khai báo "login" trong ui-shot.config.json nên không cần tài khoản')
  const source = PROJECT.credentials[0]
  const current = readEnvFile(source.file)
  const label = PROJECT.login.label || 'Email / tên đăng nhập'
  let email = opts.email
  let password
  const piped = !opts.gui && !process.stdin.isTTY ? await readPipedLines() : null
  if (piped) {
    if (!email) email = (piped.shift() || '').trim()
    password = piped[0] || ''
  } else if (opts.gui || !process.stdin.isTTY) {
    const value = await askInWindow({
      label,
      email: email || current[source.email] || '',
      hasPassword: Boolean(current[source.password]),
      file: source.file,
      base: opts.base,
    })
    if (!value) throw new Error('đã đóng cửa sổ, không lưu gì')
    email = value.email.trim()
    password = value.password || current[source.password]
  } else {
    console.error(`Lưu tài khoản đăng nhập cho ${PROJECT.name} vào ${source.file} (chmod 600)`)
    if (!email) {
      const hint = current[source.email] ? ` [${current[source.email]}]` : ''
      email = (await ask(`${label}${hint}: `)).trim() || current[source.email]
    }
    const keep = current[source.password] ? ' (Enter = giữ mật khẩu cũ)' : ''
    password = (await ask(`Mật khẩu${keep}: `, { hidden: true })) || current[source.password]
  }
  if (!email || !password) throw new Error('Thiếu email hoặc mật khẩu')
  if (/[\r\n]/.test(email + password)) throw new Error('Email/mật khẩu không được chứa xuống dòng')
  writeCredentials(source, email, password)
  fs.rmSync(PROFILE_DIR, { recursive: true, force: true })
  return source
}
