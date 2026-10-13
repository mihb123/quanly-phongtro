import { FREEZE_CSS, MOBILE_UA, PROJECT } from './config.mjs'
import { resolveDevice } from './cli.mjs'
import { loadCredentials } from './env.mjs'

export async function waitForStable(page) {
  await page.waitForNetworkIdle({ idleTime: 400, timeout: 10_000 }).catch(() => {})
  await page.waitForFunction(() => {
    const busy = document.querySelectorAll('[data-slot="skeleton"], .animate-pulse, [aria-busy="true"]')
    return [...busy].every((el) => !el.getClientRects().length)
  }, { timeout: 8_000, polling: 200 }).catch(() => {})
  await page.evaluate(async () => {
    if (document.fonts?.ready) await document.fonts.ready.catch(() => {})
    await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)))
  })
  await new Promise((r) => setTimeout(r, 250))
}


export async function preparePage(browser, { device, theme, dpr, storage, allowWrites, record }) {
  const page = await browser.newPage()
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

export async function runStep(page, step) {
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
  await waitForStable(page)
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

