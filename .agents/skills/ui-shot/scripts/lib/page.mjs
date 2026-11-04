import { FREEZE_CSS, MOBILE_UA, PROJECT } from './config.mjs'
import { resolveDevice } from './cli.mjs'
import { loadCredentials } from './env.mjs'

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const shortUrl = (url) => url.replace(/^https?:\/\/[^/]+/, '').slice(0, 160)
// Request không phải dữ liệu của app (analytics, HMR, profiler...) — không chờ.
const IGNORED_REQUESTS = [
  /\/g\/collect|google-analytics|googletagmanager|doubleclick|facebook\.com\/tr|hotjar|clarity\.ms|sentry/,
  /\/_wdt\/|\/_profiler\/|hot-update|sockjs-node|\/__vite|\/@vite\//,
]
const LOADING_SELECTORS = [
  '[data-slot="skeleton"]', '.animate-pulse', '[aria-busy="true"]', '.p-progress-spinner', '.spinner-border',
  '.spinner-grow', '.general-loading-container', '.pt-loading', '.is-loading', '.loading-overlay',
]

// Thời gian tối đa chờ API (ms) — đặt từ --wait-api.
let apiTimeoutMs = 60_000
export function setApiTimeout(seconds) {
  apiTimeoutMs = Math.max(1, Number(seconds) || 60) * 1000
}

// Chờ tới khi không còn request XHR/fetch nào đang chạy trong `idle` ms. Trả danh sách request còn treo khi hết giờ.
async function waitForApiIdle(page, { until, idle = 500, log }) {
  const tracker = page.__uiShot
  if (!tracker) return []
  let lastLog = Date.now()
  for (;;) {
    if (!tracker.inflight.size && Date.now() - tracker.lastChange >= idle) return []
    if (Date.now() >= until) return [...tracker.inflight.values()]
    if (log && tracker.inflight.size && Date.now() - lastLog >= 3000) {
      lastLog = Date.now()
      const list = [...tracker.inflight.values()]
      log(`chờ API: còn ${list.length} request (${list.slice(0, 2).join(', ')}${list.length > 2 ? ', …' : ''})`)
    }
    await sleep(100)
  }
}

// Các selector "đang tải" đang hiện trong màn hình (bỏ qua spinner của panel offcanvas trượt ra ngoài,
// vd sidebar thông báo) — để chờ và báo đúng tên cho người dùng.
function shownLoaders(selectors) {
  const shown = (el) => {
    const r = el.getBoundingClientRect()
    return r.width > 0 && r.height > 0 && r.right > 0 && r.left < window.innerWidth && getComputedStyle(el).visibility !== 'hidden'
      && !el.closest('.offcanvas:not(.offcanvas-on):not(.show), [aria-hidden="true"]')
  }
  return selectors.filter((sel) => [...document.querySelectorAll(sel)].some(shown))
}

function visibleLoaders(page) {
  return page.evaluate(shownLoaders, [...LOADING_SELECTORS, ...PROJECT.loadingSelectors]).catch(() => [])
}

// Chờ trang ổn định: API xong hết + spinner/skeleton tắt (lặp vì API xong có thể kéo API khác), rồi ảnh/font.
// Trả { pendingApi: [...], loaders: [selector...] } — khác rỗng nghĩa là hết giờ mà vẫn còn đang tải.
export async function waitForStable(page, { log } = {}) {
  const until = Date.now() + apiTimeoutMs
  let pendingApi = []
  let loaders = []
  for (let round = 0; round < 5; round++) {
    pendingApi = await waitForApiIdle(page, { until, log })
    loaders = await visibleLoaders(page)
    if (!loaders.length || Date.now() >= until) break
    if (log) log(`chờ vùng đang tải biến mất: ${loaders.join(', ')}`)
    const deadline = Math.min(until, Date.now() + 5000)
    while (Date.now() < deadline && (await visibleLoaders(page)).length) await sleep(200)
  }
  if (!pendingApi.length && !loaders.length) pendingApi = await waitForApiIdle(page, { until: Date.now() + 5000 })
  await page.waitForNetworkIdle({ idleTime: 300, timeout: 3_000 }).catch(() => {})
  await page.evaluate(async () => {
    if (document.fonts?.ready) await document.fonts.ready.catch(() => {})
    await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)))
  })
  await sleep(250)
  return { pendingApi, loaders: pendingApi.length ? loaders : await visibleLoaders(page) }
}


export async function preparePage(browser, { device, theme, dpr, storage, allowWrites, record }) {
  const page = await browser.newPage()
  if (PROJECT.cookies?.length) {
    await page.setCookie(...PROJECT.cookies)
  }
  await page.setViewport({
    width: device.width, height: device.height, deviceScaleFactor: dpr,
    isMobile: device.mobile, hasTouch: device.mobile,
  })
  if (device.mobile) await page.setUserAgent(MOBILE_UA)
  await page.emulateMediaFeatures([
    { name: 'prefers-color-scheme', value: theme },
    { name: 'prefers-reduced-motion', value: 'reduce' },
  ])
  await page.evaluateOnNewDocument((entries, mode, css, themeKey) => {
    try {
      for (const [key, value] of Object.entries(entries)) localStorage.setItem(key, value)
      if (themeKey) localStorage.setItem(themeKey, mode)
    } catch { /* noop */ }
    const inject = () => {
      const style = document.createElement('style')
      style.setAttribute('data-ui-shot', '')
      style.textContent = css
      document.head.appendChild(style)
    }
    if (document.head) inject()
    else document.addEventListener('DOMContentLoaded', inject, { once: true })
  }, storage, theme, FREEZE_CSS, PROJECT.themeStorageKey)

  page.on('console', (msg) => {
    if (msg.type() === 'error') record.console.push(msg.text().slice(0, 300))
  })
  page.on('pageerror', (err) => record.pageErrors.push(String(err?.message || err).slice(0, 300)))
  page.on('response', (res) => {
    const url = res.url()
    if (res.status() >= 400 && url.includes(PROJECT.apiPrefix)) record.network.push(`${res.status()} ${res.request().method()} ${url.replace(/^https?:\/\/[^/]+/, '')}`)
  })
  page.on('requestfailed', (req) => {
    const reason = req.failure()?.errorText || 'failed'
    if (reason.includes('ERR_BLOCKED_BY_CLIENT') || reason.includes('ERR_FAILED')) return
    record.network.push(`FAILED ${req.method()} ${req.url().replace(/^https?:\/\/[^/]+/, '')} (${reason})`)
  })
  // Theo dõi request XHR/fetch đang chạy để waitForStable chờ API xong hết.
  const tracker = { inflight: new Map(), lastChange: Date.now() }
  page.__uiShot = tracker
  const untrack = (req) => {
    if (tracker.inflight.delete(req)) tracker.lastChange = Date.now()
  }
  page.on('requestfinished', untrack)
  page.on('requestfailed', untrack)
  await page.setRequestInterception(true)
  page.on('request', (req) => {
    const method = req.method()
    const url = req.url()
    const isWrite = !['GET', 'HEAD', 'OPTIONS'].includes(method)
    if (!allowWrites && isWrite && url.includes(PROJECT.apiPrefix) && !PROJECT.allowedWrites.some((re) => re.test(url.split('?')[0]))) {
      record.blockedWrites.push(`${method} ${url.replace(/^https?:\/\/[^/]+/, '')}`)
      req.abort('blockedbyclient').catch(() => {})
      return
    }
    if (['xhr', 'fetch'].includes(req.resourceType()) && ![...IGNORED_REQUESTS, ...PROJECT.ignoreRequests].some((re) => re.test(url))) {
      tracker.inflight.set(req, `${method} ${shortUrl(url)}`)
      tracker.lastChange = Date.now()
    }
    req.continue().catch(() => {})
  })
  return page
}

export async function login(browser, opts) {
  const record = { console: [], pageErrors: [], network: [], blockedWrites: [] }
  const page = await preparePage(browser, {
    device: resolveDevice('desktop'), theme: 'light', dpr: 1, storage: {}, allowWrites: false, record,
  })
  try {
    const cfg = PROJECT.login
    await page.goto(`${opts.base}/`, { waitUntil: 'domcontentloaded', timeout: 30_000 })
    await waitForStable(page)
    if (!new URL(page.url()).pathname.startsWith(cfg.path)) return 'session'
    const creds = loadCredentials()
    if (!creds) {
      const files = PROJECT.credentials.map((c) => c.file).join(', ') || 'chưa khai báo "credentials" trong ui-shot.config.json'
      throw new Error(`Cần đăng nhập nhưng thiếu tài khoản: đặt UI_SHOT_EMAIL/UI_SHOT_PASSWORD hoặc điền file (chmod 600): ${files}`)
    }
    await page.waitForSelector(cfg.email, { visible: true, timeout: 15_000 })
    await page.type(cfg.email, creds.email)
    await page.type(cfg.password, creds.password)
    await page.click(cfg.submit)
    await page.waitForFunction((p) => !location.pathname.startsWith(p), { timeout: 20_000 }, cfg.path).catch(() => {})
    if (new URL(page.url()).pathname.startsWith(cfg.path)) throw new Error(`Đăng nhập thất bại (vẫn ở ${cfg.path})`)
    return 'logged-in'
  } finally {
    await page.close().catch(() => {})
  }
}

export async function runStep(page, step, { log } = {}) {
  const sep = step.indexOf(':')
  const kind = sep > 0 ? step.slice(0, sep) : step
  const arg = sep > 0 ? step.slice(sep + 1) : ''
  switch (kind) {
    case 'click':
      if (arg.startsWith('text=')) {
        const label = arg.slice(5).trim()
        const handle = await page.evaluateHandle((text) => {
          const norm = (s) => (s || '').replace(/\s+/g, ' ').trim()
          const candidates = [...document.querySelectorAll('button, a, [role="button"], [role="tab"], [role="menuitem"], [role="option"], [role="radio"], [role="checkbox"], [role="switch"], label, summary, [data-slot$="trigger"]')]
            .filter((el) => el.getClientRects().length && el.checkVisibility?.({ opacityProperty: true, visibilityProperty: true }) !== false)
          return candidates.find((el) => norm(el.innerText) === text || norm(el.getAttribute('aria-label')) === text)
            || candidates.filter((el) => norm(el.innerText).includes(text)).sort((a, b) => norm(a.innerText).length - norm(b.innerText).length)[0]
            || null
        }, label)
        const el = handle.asElement()
        if (!el) throw new Error(`Không thấy phần tử bấm được có chữ "${label}"`)
        await el.click()
      } else {
        await page.waitForSelector(arg, { visible: true, timeout: 8_000 })
        await page.click(arg)
      }
      break
    case 'type': {
      const eq = arg.indexOf('=')
      if (eq < 1) throw new Error(`type cần dạng type:<css>=<giá trị>`)
      const selector = arg.slice(0, eq)
      await page.waitForSelector(selector, { visible: true, timeout: 8_000 })
      await page.type(selector, arg.slice(eq + 1))
      break
    }
    case 'press': await page.keyboard.press(arg); break
    case 'hover': await page.hover(arg); break
    case 'scroll': await page.$eval(arg, (el) => el.scrollIntoView({ block: 'center' })); break
    case 'waitfor': await page.waitForSelector(arg, { visible: true, timeout: 10_000 }); break
    case 'wait': await new Promise((r) => setTimeout(r, Number(arg) || 500)); break
    default: throw new Error(`Step không hỗ trợ: ${step}`)
  }
  return waitForStable(page, { log })
}

// ---- Tab trong trang ----
// Dò thanh tab: selector chỉ định → [role=tab] → nav-tabs Bootstrap → nhóm nút anh em có class "...-tab".
// Đánh dấu từng tab bằng data-ui-shot-tab="<số thứ tự>" để bấm lại. Trả { how, tabs: [{ index, label, active }] }.
function findTabsInPage(selector) {
  // Hiển thị thật: có kích thước, không ẩn, nằm trong bề ngang màn hình (loại panel offcanvas trượt ra ngoài).
  const visible = (el) => {
    const r = el.getBoundingClientRect()
    return r.width > 0 && r.height > 0 && r.right > 0 && r.left < window.innerWidth && getComputedStyle(el).visibility !== 'hidden'
  }
  const label = (el) => (el.innerText || el.getAttribute('aria-label') || el.title || '').replace(/\s+/g, ' ').trim()
  const inModal = (el) => Boolean(el.closest('[role="dialog"], .modal, .p-dialog, .offcanvas, [class*="quick-panel"], [class*="quick_panel"]'))
  const groups = []
  const add = (how, els, specificity = 0) => {
    const list = [...new Set(els)].filter((el) => visible(el) && !inModal(el) && label(el))
    if (list.length >= (how === 'selector' ? 1 : 2)) groups.push({ how, els: list, specificity })
  }
  if (selector) {
    add('selector', [...document.querySelectorAll(selector)])
  } else {
    for (const list of document.querySelectorAll('[role="tablist"]')) add('role=tab', [...list.querySelectorAll('[role="tab"]')])
    for (const nav of document.querySelectorAll('.nav-tabs, .nav-pills, .nav-tabs-line')) add('nav-tabs', [...nav.querySelectorAll(':scope > li > a, :scope > li > button, :scope > a, :scope > button, :scope > .nav-link')])
    const tabToken = /(^|[-_])tabs?$/i
    // Nhóm theo cha + class tab; class gốc ("ddr-tab") ưu tiên hơn biến thể ("ddr-status-tab" — thường là bộ lọc).
    const byKey = new Map()
    for (const el of document.querySelectorAll('button, a, [role="button"], li')) {
      const token = [...el.classList].find((c) => tabToken.test(c) && !/content|pane|body|panel/i.test(c))
      if (!token) continue
      if (!byKey.has(el.parentElement)) byKey.set(el.parentElement, new Map())
      const byToken = byKey.get(el.parentElement)
      if (!byToken.has(token)) byToken.set(token, [])
      byToken.get(token).push(el)
    }
    for (const byToken of byKey.values()) {
      for (const [token, els] of byToken) add('class *-tab', els, token.split(/[-_]/).length)
    }
  }
  if (!groups.length) return { how: null, tabs: [] }
  const rank = { selector: 0, 'role=tab': 1, 'nav-tabs': 2, 'class *-tab': 2 }
  const top = (g) => g.els[0].getBoundingClientRect().top + window.scrollY
  groups.sort((a, b) => rank[a.how] - rank[b.how] || a.specificity - b.specificity || top(a) - top(b))
  const { how, els } = groups[0]
  document.querySelectorAll('[data-ui-shot-tab]').forEach((el) => el.removeAttribute('data-ui-shot-tab'))
  return {
    how,
    tabs: els.map((el, i) => {
      el.setAttribute('data-ui-shot-tab', String(i + 1))
      const active = el.getAttribute('aria-selected') === 'true' || /(^|\s)(active|selected|current)(\s|$)/.test(el.className) || el.classList.contains('is-active')
      return { index: i + 1, label: label(el), active }
    }),
  }
}

export function tabSelectorFor(url, override) {
  if (override) return override
  const map = PROJECT.tabSelectors || {}
  const pathname = new URL(url).pathname
  const key = Object.keys(map).filter((k) => k !== '*' && pathname.startsWith(k)).sort((a, b) => b.length - a.length)[0]
  return map[key] || map['*'] || null
}

export function findTabs(page, selector) {
  return page.evaluate(findTabsInPage, selector || null)
}

// Bấm tab (khớp theo nhãn, rồi theo số thứ tự) và chờ API của tab tải xong.
export async function openTab(page, tab, selector, { log } = {}) {
  const found = await findTabs(page, selector)
  const norm = (s) => s.toLowerCase().replace(/\s+/g, ' ').trim()
  const match = found.tabs.find((t) => norm(t.label) === norm(tab.label)) || found.tabs.find((t) => t.index === tab.index)
  if (!match) throw new Error(`không thấy tab "${tab.label}" trên màn hình này (thấy: ${found.tabs.map((t) => t.label).join(' | ') || 'không có thanh tab'})`)
  await page.$eval(`[data-ui-shot-tab="${match.index}"]`, (el) => { el.scrollIntoView({ block: 'nearest', inline: 'center' }); el.click() })
  await page.evaluate(() => window.scrollTo(0, 0))
  return waitForStable(page, { log })
}


export async function scrollTo(page, position) {
  return page.evaluate((pos) => {
    const vh = window.innerHeight
    const docEl = document.scrollingElement || document.documentElement
    let best = docEl.scrollHeight > vh + 1 ? { el: docEl, area: window.innerWidth * vh } : null
    for (const el of document.body.querySelectorAll('*')) {
      const s = getComputedStyle(el)
      if (!['auto', 'scroll', 'overlay'].includes(s.overflowY)) continue
      if (el.scrollHeight <= el.clientHeight + 1 || el.clientHeight < vh * 0.4) continue
      if (!el.getClientRects().length) continue
      const area = el.clientWidth * el.clientHeight
      if (!best || area > best.area) best = { el, area }
    }
    if (!best) return { pages: 1, done: true }
    const el = best.el
    const step = Math.round(el.clientHeight * 0.85)
    const max = el.scrollHeight - el.clientHeight
    const target = pos === 'bottom' ? max : Math.min(max, pos * step)
    el.scrollTop = target
    return { pages: Math.ceil(max / step) + 1, done: el.scrollTop >= max - 2, scrollTop: el.scrollTop, max }
  }, position)
}

export async function settleFrame(page) {
  await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))))
  await new Promise((r) => setTimeout(r, 200))
}

