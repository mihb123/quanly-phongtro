import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { homedir } from 'node:os'
import path from 'node:path'
import { randomBytes, randomInt } from 'node:crypto'
import { it } from 'node:test'
import puppeteer from 'puppeteer'
import { ARTIFACTS_DIR, BASE_URL, DOWNLOADS_DIR, HEADFUL, PASSWORD, PASSWORD_HASH, SERVER_LOG, SLOWMO, TIMEZONE } from './config.mjs'
import { execSql, value, waitFor } from './db.mjs'

export const PLUGIN_NAME = 'Thu chi cá nhân'

export const GEO = { latitude: 21.028511, longitude: 105.854167 }
const VIEWPORT = { width: 430, height: 932 }
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

function findChrome() {
  if (process.env.PUPPETEER_EXECUTABLE_PATH) return process.env.PUPPETEER_EXECUTABLE_PATH
  const base = path.join(homedir(), '.cache/puppeteer/chrome')
  if (!existsSync(base)) return undefined
  for (const v of readdirSync(base).sort().reverse()) {
    const p = path.join(base, v, 'chrome-linux64/chrome')
    if (existsSync(p)) return p
  }
  return undefined
}

let browserPromise = null
const sessions = new Set()

function getBrowser() {
  browserPromise ??= puppeteer.launch({
    headless: !HEADFUL,
    slowMo: SLOWMO,
    executablePath: findChrome(),
    args: ['--no-sandbox', '--lang=vi-VN', '--disable-features=Translate', '--no-default-browser-check'],
  })
  return browserPromise
}

export async function closeAll() {
  const errors = []
  for (const s of [...sessions]) {
    errors.push(...s.pageErrors.map((e) => `[${s.label}] ${e}`))
    await s.close()
  }
  if (browserPromise) {
    const b = await browserPromise
    browserPromise = null
    await b.close()
  }
  if (errors.length) throw new Error(`Trang có lỗi JavaScript chưa bắt:\n${errors.join('\n')}`)
}

function slug(s) {
  return s
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/đ/gi, 'd')
    .replace(/[^a-zA-Z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
    .slice(0, 80)
}

async function dumpArtifacts(name, err) {
  mkdirSync(ARTIFACTS_DIR, { recursive: true })
  const base = path.join(ARTIFACTS_DIR, `${slug(name)}-${Date.now()}`)
  for (const s of sessions) {
    try {
      await s.page.screenshot({ path: `${base}-${s.label}.png`, fullPage: true })
      writeFileSync(`${base}-${s.label}.txt`, [`URL: ${s.page.url()}`, `Lỗi: ${err?.stack ?? err}`, '', '--- console ---', ...s.logs, '', '--- body ---', await s.page.evaluate(() => document.body.innerText).catch(() => '')].join('\n'))
    } catch {
      /* page đã đóng */
    }
  }
  return base
}

export function step(name, fn, opts = {}) {
  return it(name, opts, async (t) => {
    try {
      await fn(t)
    } catch (err) {
      const base = await dumpArtifacts(name, err)
      err.message += `\n  ảnh chụp/log: ${base}-*.png|txt`
      throw err
    }
  })
}

export function uniqueEmail(prefix = 'e2e') {
  return `${prefix}-${Date.now().toString(36)}-${randomBytes(3).toString('hex')}@example.com`
}

function fakeIP() {
  return `10.${randomInt(1, 254)}.${randomInt(1, 254)}.${randomInt(1, 254)}`
}

export async function readOtp(email, { timeout = 15000 } = {}) {
  const re = new RegExp(`OTP for ${email.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}: (\\d{6})`, 'g')
  return waitFor(
    () => {
      const log = existsSync(SERVER_LOG) ? readFileSync(SERVER_LOG, 'utf8') : ''
      const all = [...log.matchAll(re)]
      return all.length ? all[all.length - 1][1] : null
    },
    { timeout, message: `mã OTP của ${email} trong log server` },
  )
}

function installHelpers() {
  document.addEventListener('DOMContentLoaded', () => {
    const style = document.createElement('style')
    style.textContent = '[data-sonner-toaster], [data-sonner-toaster] * { pointer-events: none !important; }'
    document.head.appendChild(style)
  })
  const norm = (s) => (s ?? '').replace(/\s+/g, ' ').trim()
  const visible = (el) => {
    if (!el || !el.isConnected) return false
    const r = el.getBoundingClientRect()
    if (r.width === 0 || r.height === 0) return false
    const st = getComputedStyle(el)
    return st.visibility !== 'hidden' && st.display !== 'none'
  }
  const layers = () => [...document.querySelectorAll('[data-slot="sheet-content"]')].filter((el) => !el.hasAttribute('data-closed') && !el.hasAttribute('data-ending-style') && visible(el))
  const app = () => document.querySelector('[data-thuchi]')
  const scoped = (sel) => (app() && /^(nav|header|main)\b/.test(sel) ? sel.split(',').map((x) => `[data-thuchi] ${x.trim()}`).join(', ') : sel)
  const root = (within) => {
    if (within === 'page') return document.body
    if (within) return [...document.querySelectorAll(scoped(within))].filter(visible).at(-1) ?? null
    return layers().at(-1) ?? document.body
  }
  const textOf = (el) => norm(el.innerText ?? el.textContent)
  const matches = (el, spec) => {
    if (spec.label !== undefined && el.getAttribute('aria-label') !== spec.label) return false
    if (spec.text === undefined) return true
    const t = textOf(el)
    if (spec.regex) return new RegExp(spec.regex, 'i').test(t)
    return spec.exact === false ? t.includes(norm(spec.text)) : t === norm(spec.text)
  }
  window.__e2e = {
    norm,
    layers,
    find(spec) {
      const r = root(spec.within)
      if (!r) return null
      const sel = spec.selector ?? (spec.label !== undefined ? '[aria-label]' : 'button, a, [role="option"], [role="switch"], [role="checkbox"]')
      const all = [...r.querySelectorAll(r === document.body ? scoped(sel) : sel)].filter((el) => visible(el) && matches(el, spec))
      return all[spec.nth ?? 0] ?? null
    },
    findTrigger(spec) {
      const r = root(spec.within)
      if (!r) return null
      if (spec.label) {
        const lab = [...r.querySelectorAll('label')].find((l) => visible(l) && norm(l.childNodes[0]?.textContent ?? l.innerText) === norm(spec.label))
        let p = lab?.parentElement
        while (p && p !== r.parentElement && !p.querySelector('[data-slot="select-trigger"]')) p = p.parentElement
        return p?.querySelector('[data-slot="select-trigger"]') ?? null
      }
      const triggers = [...r.querySelectorAll('[data-slot="select-trigger"]')].filter(visible)
      if (spec.current !== undefined) return triggers.find((t) => textOf(t) === norm(spec.current)) ?? null
      return triggers[spec.nth ?? 0] ?? null
    },
    findSwitch(spec) {
      const r = root(spec.within)
      if (!r) return null
      const sel = spec.kind === 'checkbox' ? '[data-slot="checkbox"]' : '[data-slot="switch"]'
      return (
        [...r.querySelectorAll(sel)].filter(visible).find((s) => {
          let p = s.parentElement
          for (let i = 0; i < 4 && p; i++, p = p.parentElement) if (textOf(p).includes(norm(spec.text))) return true
          return false
        }) ?? null
      )
    },
    text(within) {
      const r = root(within)
      return r ? textOf(r) : ''
    },
    section(title) {
      const h = [...(app() ?? document).querySelectorAll('section h2, section p')].find((el) => visible(el) && norm(el.innerText).startsWith(norm(title)))
      return h ? textOf(h.closest('section')) : null
    },
  }
}

export class Session {
  static async create({ email, password, label, ip }) {
    const browser = await getBrowser()
    const context = await browser.createBrowserContext()
    await context.overridePermissions(BASE_URL, ['geolocation', 'clipboard-read', 'clipboard-write'])
    const page = await context.newPage()
    const s = new Session({ context, page, email, password, label, ip })
    await s.setupPage()
    sessions.add(s)
    return s
  }

  constructor({ context, page, email, password, label, ip }) {
    this.context = context
    this.page = page
    this.email = email
    this.password = password
    this.label = label ?? 'main'
    this.ip = ip ?? fakeIP()
    this.owner = null
    this.accessToken = null
    this.logs = []
    this.pageErrors = []
  }

  async setupPage() {
    const p = this.page
    await p.setViewport(VIEWPORT)
    await p.emulateTimezone(TIMEZONE)
    await p.setGeolocation(GEO)
    await p.setExtraHTTPHeaders({ 'X-Forwarded-For': this.ip })
    await p.evaluateOnNewDocument(installHelpers)
    p.setDefaultTimeout(15000)
    p.on('console', (m) => {
      this.logs.push(`[${m.type()}] ${m.text()}`)
      if (this.logs.length > 300) this.logs.shift()
    })
    p.on('pageerror', (e) => this.pageErrors.push(e.message))
    p.on('request', (r) => {
      const auth = r.headers().authorization
      if (auth?.startsWith('DPoP ')) this.accessToken = auth.slice(5)
    })
  }

  async close() {
    sessions.delete(this)
    await this.context.close().catch(() => undefined)
  }

  async newDevice(label = 'device2') {
    const s = await Session.create({ email: this.email, password: this.password, label, ip: this.ip })
    s.owner = this.owner
    await s.login()
    return s
  }

  async open(pathname = '/') {
    await this.page.goto(BASE_URL + pathname, { waitUntil: 'domcontentloaded' })
  }

  async waitUrl(pathname, timeout = 15000) {
    await this.page.waitForFunction((p) => location.pathname === p, { timeout }, pathname)
  }

  async register({ enable = true } = {}) {
    if (this.password !== PASSWORD) throw new Error('register() chỉ hỗ trợ mật khẩu mặc định PASSWORD')
    await execSql`insert into users (email, password_hash, role, full_name, is_activated) values (${this.email}, ${PASSWORD_HASH}, 'MANAGER', 'Chủ trọ E2E', true)`
    this.owner = await value`select id from users where email = ${this.email}`
    await this.login({ open: false })
    if (enable) {
      await this.enablePlugin()
      await this.waitReady()
    }
    return this
  }

  async login({ open = true } = {}) {
    await this.open('/login')
    await this.page.evaluate(() => {
      localStorage.setItem('home_active_tab', 'plugins')
      localStorage.setItem('plugins:open', 'thu-chi')
    })
    await this.page.reload({ waitUntil: 'domcontentloaded' })
    await this.fill('#email', this.email)
    await this.fill('#password', this.password)
    await this.click({ text: 'Đăng nhập', selector: 'button[type="submit"]' })
    await this.waitUrl('/', 20000)
    this.owner ??= await value`select id from users where email = ${this.email}`
    await this.page.waitForFunction((name) => !!document.querySelector('[data-thuchi]') || [...document.querySelectorAll('[aria-label]')].some((e) => e.getAttribute('aria-label') === `Bật ${name}`), { timeout: 20000, polling: 100 }, PLUGIN_NAME)
    if (open) await this.waitReady()
    return this
  }

  async enablePlugin() {
    if (await this.page.evaluate(() => !!document.querySelector('[data-thuchi]'))) return
    await this.click({ label: `Bật ${PLUGIN_NAME}`, within: 'page' })
    await this.toast(`Đã bật ${PLUGIN_NAME}`)
    const opened = await this.page.waitForSelector('[data-thuchi]', { timeout: 3000 }).catch(() => null)
    if (!opened) {
      await this.click({ text: PLUGIN_NAME, selector: 'button', within: 'page' })
      await this.page.waitForSelector('[data-thuchi]', { timeout: 20000 })
    }
  }

  async waitReady() {
    await this.find({ text: 'Tổng quan', within: 'nav', exact: false }, { timeout: 20000 })
    await this.waitSynced()
  }

  async syncTitle() {
    return this.page.evaluate(() => document.querySelector('[data-thuchi] header button[title]')?.title ?? '')
  }

  async waitSynced(timeout = 20000) {
    await this.page.waitForFunction(() => document.querySelector('[data-thuchi] header button[title]')?.title.startsWith('Đã đồng bộ'), { timeout, polling: 100 })
  }

  async sync() {
    await this.closeSheet()
    const pulled = this.page.waitForResponse((r) => r.url().includes('/sync/pull') && r.request().method() === 'GET' && r.status() === 200, { timeout: 20000 })
    const btn = await this.find({ selector: '[data-thuchi] header button[title]', within: 'page' })
    await btn.click()
    await pulled
    await this.waitSynced()
    await sleep(150)
  }

  async find(spec, { timeout = 15000 } = {}) {
    try {
      const h = await this.page.waitForFunction((s) => window.__e2e.find(s), { timeout, polling: 100 }, spec)
      return h.asElement()
    } catch (err) {
      throw new Error(`Không thấy phần tử ${JSON.stringify(spec)} (${err.message})`)
    }
  }

  async exists(spec) {
    return this.page.evaluate((s) => !!window.__e2e.find(s), spec)
  }

  async settle(el) {
    await el.evaluate((e) => e.scrollIntoView({ block: 'center', inline: 'nearest' }))
    let prev = null
    for (let i = 0; i < 40; i++) {
      const box = await el.boundingBox()
      if (box && prev && Math.abs(box.x - prev.x) < 0.5 && Math.abs(box.y - prev.y) < 0.5 && Math.abs(box.height - prev.height) < 0.5) break
      prev = box
      await sleep(50)
    }
    await this.page.waitForFunction((e) => !e.disabled && e.getAttribute('aria-disabled') !== 'true' && !e.hasAttribute('data-disabled'), { timeout: 10000 }, el)
  }

  async click(textOrSpec, opts = {}) {
    const spec = typeof textOrSpec === 'string' ? { text: textOrSpec, ...opts } : { ...textOrSpec, ...opts }
    const el = await this.find(spec, opts)
    await this.settle(el)
    await el.click()
    return el
  }

  async fill(selector, text) {
    const el = await this.page.waitForSelector(selector, { visible: true })
    await this.settle(el)
    await el.click()
    await this.page.keyboard.down('Control')
    await this.page.keyboard.press('a')
    await this.page.keyboard.up('Control')
    await this.page.keyboard.press('Backspace')
    if (text !== '') await el.type(String(text))
    return el
  }

  async blur() {
    await this.page.evaluate(() => document.activeElement?.blur())
  }

  async setValue(selector, v) {
    await this.page.waitForSelector(selector, { visible: true })
    await this.page.$eval(
      selector,
      (el, val) => {
        const proto = el instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype
        Object.getOwnPropertyDescriptor(proto, 'value').set.call(el, val)
        el.dispatchEvent(new Event('input', { bubbles: true }))
        el.dispatchEvent(new Event('change', { bubbles: true }))
      },
      v,
    )
  }

  async inputValue(selector) {
    return this.page.$eval(selector, (el) => el.value)
  }

  async choose(trigger, option) {
    const h = await this.page.waitForFunction((s) => window.__e2e.findTrigger(s), { polling: 100 }, trigger)
    const el = h.asElement()
    await this.settle(el)
    await el.click()
    await this.click({ text: option, selector: '[data-slot="select-item"]', within: '[data-slot="select-content"]' })
    await this.page.waitForFunction(() => ![...document.querySelectorAll('[data-slot="select-content"]')].some((e) => e.getBoundingClientRect().height > 0 && !e.hasAttribute('data-closed')), { polling: 100 })
    await sleep(150)
  }

  async toggle(text, { kind = 'switch', within } = {}) {
    const h = await this.page.waitForFunction((s) => window.__e2e.findSwitch(s), { polling: 100 }, { text, kind, within })
    const el = h.asElement()
    await this.settle(el)
    await el.click()
  }

  async isChecked(text, { kind = 'switch', within } = {}) {
    return this.page.evaluate((s) => {
      const el = window.__e2e.findSwitch(s)
      return el?.getAttribute('aria-checked') === 'true' || el?.hasAttribute('data-checked')
    }, { text, kind, within })
  }

  async toast(text, timeout = 10000) {
    await this.page.waitForFunction((t) => [...document.querySelectorAll('[data-sonner-toast]')].some((e) => window.__e2e.norm(e.innerText).includes(t)), { timeout, polling: 100 }, text)
  }

  async text(within) {
    return this.page.evaluate((w) => window.__e2e.text(w), within)
  }

  async section(title) {
    return this.page.evaluate((t) => window.__e2e.section(t), title)
  }

  async expectText(text, { within, timeout = 15000 } = {}) {
    try {
      await this.page.waitForFunction((t, w) => window.__e2e.text(w).includes(window.__e2e.norm(t)), { timeout, polling: 100 }, text, within)
    } catch {
      throw new Error(`Không thấy chữ "${text}"${within ? ` trong ${within}` : ''} trên màn hình. Nội dung: ${(await this.text(within)).slice(0, 1500)}`)
    }
  }

  async expectNoText(text, { within, timeout = 15000 } = {}) {
    try {
      await this.page.waitForFunction((t, w) => !window.__e2e.text(w).includes(window.__e2e.norm(t)), { timeout, polling: 100 }, text, within)
    } catch {
      throw new Error(`Chữ "${text}" vẫn còn trên màn hình${within ? ` (${within})` : ''}`)
    }
  }

  async expectSection(title, parts, { timeout = 15000 } = {}) {
    const want = [].concat(parts)
    try {
      await this.page.waitForFunction(
        (t, ps) => {
          const s = window.__e2e.section(t)
          return s !== null && ps.every((p) => s.includes(window.__e2e.norm(p)))
        },
        { timeout, polling: 100 },
        title,
        want,
      )
    } catch {
      throw new Error(`Khối "${title}" không chứa ${JSON.stringify(want)}. Nội dung: ${await this.section(title)}`)
    }
  }

  async closeSheet() {
    for (let i = 0; i < 3; i++) {
      const open = await this.page.evaluate(() => window.__e2e?.layers().length ?? 0)
      if (!open) return
      await this.page.keyboard.press('Escape')
      await sleep(200)
    }
    await this.page.waitForFunction(() => window.__e2e.layers().length === 0, { timeout: 5000 })
  }

  async waitSheet(title) {
    await this.page.waitForFunction((t) => window.__e2e.layers().some((l) => window.__e2e.norm(l.innerText).includes(t)), { polling: 100 }, title)
    await sleep(250)
  }

  async waitNoSheet() {
    await this.page.waitForFunction(() => window.__e2e.layers().length === 0, { polling: 100 })
    await sleep(150)
  }

  async tab(name) {
    await this.closeSheet()
    await this.click({ text: name, within: 'nav', exact: false })
    await sleep(100)
  }

  async settings(item) {
    await this.tab('Cài đặt')
    if (await this.exists({ text: 'Cài đặt', exact: false, selector: 'main > div > button', within: 'page' })) await this.click({ text: 'Cài đặt', exact: false, selector: 'main > div > button', within: 'page' })
    if (item) {
      await this.click({ text: item, exact: false, selector: 'main section button', within: 'page' })
      await this.find({ text: item, selector: 'main h2', within: 'page' })
    }
  }

  async openTx(text) {
    await this.closeSheet()
    await this.click({ text, exact: false, selector: 'main section button', within: 'page' })
    await this.page.waitForFunction(() => window.__e2e.layers().length > 0, { polling: 100 })
    await sleep(300)
  }

  async addTx({ kind = 'Chi', amount, title, category, account, to, fee, when, debtKind, party }) {
    await this.closeSheet()
    if (!(await this.exists({ label: 'Thêm giao dịch', within: 'page', selector: 'button' }))) await this.tab('Giao dịch')
    await this.click({ label: 'Thêm giao dịch', within: 'page', selector: 'button' })
    await this.waitSheet('Thêm giao dịch')
    if (kind !== 'Chi') await this.click(kind)
    if (debtKind) await this.click(debtKind)
    if (party !== undefined) await this.fill('#qa-party', party)
    await this.fill('#qa-amount', amount)
    if (title !== undefined) await this.fill('#qa-title', title)
    if (category) await this.click({ text: category, selector: 'button', exact: true })
    if (account) await this.choose({ label: kind === 'Chuyển tiền' ? 'Từ tài khoản' : 'Tài khoản' }, account)
    if (to) await this.choose({ label: 'Đến tài khoản' }, to)
    if (fee !== undefined) await this.fill('#qa-fee', fee)
    if (when) await this.setValue('#qa-when', when)
    await this.click({ text: 'Lưu', selector: 'button[type="submit"]' })
    await this.toast('Đã lưu giao dịch')
    await this.waitNoSheet()
  }

  async localRows(store) {
    return this.page.evaluate(
      async (name, st) => {
        const db = await new Promise((res, rej) => {
          const r = indexedDB.open(name)
          r.onsuccess = () => res(r.result)
          r.onerror = () => rej(r.error)
        })
        try {
          return await new Promise((res, rej) => {
            const q = db.transaction(st).objectStore(st).getAll()
            q.onsuccess = () => res(q.result)
            q.onerror = () => rej(q.error)
          })
        } finally {
          db.close()
        }
      },
      `thuchi_${this.owner}`,
      store,
    )
  }

  async databases() {
    return this.page.evaluate(async () => (await indexedDB.databases()).map((d) => d.name))
  }

  async api(method, url, body) {
    if (!this.accessToken) throw new Error('chưa bắt được access token từ request của app')
    return this.page.evaluate(
      async ({ method, url, token, body }) => {
        const idb = await new Promise((res, rej) => {
          const r = indexedDB.open('keyval-store')
          r.onsuccess = () => res(r.result)
          r.onerror = () => rej(r.error)
        })
        const get = (k) =>
          new Promise((res, rej) => {
            const q = idb.transaction('keyval').objectStore('keyval').get(k)
            q.onsuccess = () => res(q.result)
            q.onerror = () => rej(q.error)
          })
        const priv = await get('dpop_private_key')
        const pub = await get('dpop_public_key')
        idb.close()
        const b64u = (buf) => btoa(String.fromCharCode(...new Uint8Array(buf))).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
        const enc = (o) => b64u(new TextEncoder().encode(JSON.stringify(o)))
        const jwk = await crypto.subtle.exportKey('jwk', pub)
        const full = new URL(`/api/v1${url}`, location.origin)
        const htu = new URL(full)
        htu.search = ''
        const ath = b64u(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(token)))
        const header = { alg: 'ES256', typ: 'dpop+jwt', jwk: { kty: jwk.kty, crv: jwk.crv, x: jwk.x, y: jwk.y } }
        const payload = { htm: method, htu: htu.toString(), jti: crypto.randomUUID(), iat: Math.floor(Date.now() / 1000), ath }
        const input = `${enc(header)}.${enc(payload)}`
        const sig = await crypto.subtle.sign({ name: 'ECDSA', hash: 'SHA-256' }, priv, new TextEncoder().encode(input))
        const res = await fetch(full, {
          method,
          credentials: 'include',
          headers: { 'Content-Type': 'application/json', Authorization: `DPoP ${token}`, DPoP: `${input}.${b64u(sig)}` },
          body: body === undefined ? undefined : JSON.stringify(body),
        })
        const text = await res.text()
        let json = null
        try {
          json = JSON.parse(text)
        } catch {
          /* không phải JSON */
        }
        return { status: res.status, body: json ?? text }
      },
      { method, url, token: this.accessToken, body },
    )
  }

  async push(mutations) {
    const res = await this.api('POST', '/plugins/thu-chi/sync/push', {
      mutations: mutations.map((m) => ({ mid: crypto.randomUUID(), ts: Date.now(), ...m })),
    })
    return res
  }

  async enableDownloads() {
    mkdirSync(DOWNLOADS_DIR, { recursive: true })
    const dir = path.join(DOWNLOADS_DIR, `${this.label}-${Date.now()}`)
    mkdirSync(dir, { recursive: true })
    const cdp = await this.page.createCDPSession()
    await cdp.send('Browser.setDownloadBehavior', { behavior: 'allow', downloadPath: dir, browserContextId: this.context.id, eventsEnabled: true })
    return dir
  }
}

export async function newSession(opts = {}) {
  const s = await Session.create({ email: opts.email ?? uniqueEmail(opts.prefix), password: opts.password ?? PASSWORD, label: opts.label })
  if (opts.register !== false) await s.register()
  return s
}

export const vnd = (n) => `${n < 0 ? '−' : ''}${new Intl.NumberFormat('vi-VN').format(Math.abs(Math.round(n)))} ₫`

export function compact(n) {
  const a = Math.abs(n)
  const sign = n < 0 ? '−' : ''
  const trim = (x) => (x >= 100 ? Math.round(x).toString() : x.toFixed(1).replace(/\.0$/, '').replace('.', ','))
  if (a >= 1_000_000_000) return `${sign}${trim(a / 1_000_000_000)}tỷ`
  if (a >= 1_000_000) return `${sign}${trim(a / 1_000_000)}tr`
  if (a >= 1_000) return `${sign}${Math.round(a / 1_000)}k`
  return `${sign}${a}`
}

export { sleep }
