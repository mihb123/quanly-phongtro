import path from 'node:path'
import { PROJECT } from './config.mjs'
import { pageAudit } from './audit.mjs'
import { slugify } from './cli.mjs'
import { openTab, preparePage, runStep, scrollTo, settleFrame, tabSelectorFor, waitForStable } from './page.mjs'

export async function annotate(page, issues, file, viewport) {
  const marks = []
  for (const issue of issues) {
    if (issue.level === 'info') continue
    for (const item of issue.items) {
      const r = item.rect
      if (!r || r.y + r.h < 0 || r.y > viewport.height || r.x + r.w < 0 || r.x > viewport.width) continue
      item.mark = marks.length + 1
      marks.push({ rect: r, n: item.mark, level: issue.level })
    }
  }
  if (!marks.length) return null
  await page.evaluate((list) => {
    const root = document.createElement('div')
    root.setAttribute('data-ui-shot-overlay', '')
    root.style.cssText = 'position:fixed;inset:0;pointer-events:none;z-index:2147483647'
    for (const m of list) {
      const color = m.level === 'error' ? '#e11d48' : '#f59e0b'
      const box = document.createElement('div')
      box.style.cssText = `position:fixed;left:${m.rect.x - 2}px;top:${m.rect.y - 2}px;width:${m.rect.w + 4}px;height:${m.rect.h + 4}px;outline:2px dashed ${color}`
      const tag = document.createElement('div')
      tag.textContent = String(m.n)
      const above = m.rect.y >= 14
      tag.style.cssText = `position:fixed;left:${Math.max(0, m.rect.x - 3)}px;top:${above ? m.rect.y - 15 : m.rect.y + m.rect.h + 2}px;font:700 10px/12px monospace;color:#fff;background:${color};padding:0 3px;border-radius:2px`
      root.append(box, tag)
    }
    document.body.appendChild(root)
  }, marks)
  await page.screenshot({ path: file })
  await page.evaluate(() => document.querySelector('[data-ui-shot-overlay]')?.remove())
  return file
}

function pendingIssue(pending) {
  if (!pending.api.length && !pending.loaders.length) return null
  const parts = []
  if (pending.api.length) parts.push(`${pending.api.length} request API chưa xong`)
  if (pending.loaders.length) parts.push(`vẫn hiện vùng đang tải (${pending.loaders.join(', ')})`)
  return { level: 'warn', code: 'page/api-pending', message: `Hết thời gian chờ (--wait-api) mà còn ${parts.join(', ')} — ảnh có thể chưa đủ dữ liệu.`, items: [], samples: pending.api.slice(0, 5) }
}

// Gom lỗi console/network ghi được từ lúc bắt đầu chụp tab này vào kết quả, rồi xoá để tab sau ghi riêng.
function drainRecord(record, result) {
  for (const text of record.pageErrors) result.issues.push({ level: 'error', code: 'runtime/page-error', message: text, items: [] })
  if (record.console.length) result.issues.push({ level: 'warn', code: 'runtime/console-error', message: `${record.console.length} lỗi console`, items: [], samples: [...new Set(record.console)].slice(0, 5) })
  if (record.network.length) result.issues.push({ level: 'warn', code: 'network/failed', message: `${record.network.length} request API lỗi`, items: [], samples: [...new Set(record.network)].slice(0, 5) })
  if (record.blockedWrites.length) result.issues.push({ level: 'info', code: 'network/blocked-write', message: `Đã chặn ${record.blockedWrites.length} request ghi dữ liệu (dùng --allow-writes nếu thật sự cần)`, items: [], samples: [...new Set(record.blockedWrites)].slice(0, 5) })
  for (const list of Object.values(record)) list.length = 0
}

// Chụp + soát lỗi trạng thái hiện tại của trang (đầu trang, cuộn từng màn, chụp cận).
async function shootState(page, opts, target, device, theme, outDir, stem, result, log) {
  await page.evaluate((mode) => document.documentElement.classList.toggle('dark', mode === 'dark'), theme)
  await settleFrame(page)
  result.finalPath = new URL(page.url()).pathname
  const maxPages = opts.pages ?? (device.mobile ? 4 : 2)

  await scrollTo(page, 0)
  await settleFrame(page)
  log('soát lỗi + chụp màn 1 (đầu trang)')
  const top = await page.evaluate(pageAudit, { mobile: device.mobile, phase: 'top' })
  result.resolvedTheme = top.theme
  result.scroller = top.scroller
  const first = path.join(outDir, `${stem}.p1.png`)
  await page.screenshot({ path: first })
  result.shots.push(first)
  result.issues.push(...top.issues.map((i) => ({ ...i, at: 'p1' })))
  const ann1 = await annotate(page, top.issues, path.join(outDir, `${stem}.p1.annotated.png`), device)
  if (ann1) result.annotated.push(ann1)

  let index = 1
  let reachedBottom = !top.scroller
  while (!reachedBottom && index < maxPages) {
    const isLast = index === maxPages - 1
    const pos = await scrollTo(page, isLast ? 'bottom' : index)
    await settleFrame(page)
    index += 1
    log(`cuộn + chụp màn ${index}${isLast ? ' (cuối trang)' : ''}`)
    const file = path.join(outDir, `${stem}.p${index}.png`)
    await page.screenshot({ path: file })
    result.shots.push(file)
    reachedBottom = pos.done
    if (pos.done) {
      const bottom = await page.evaluate(pageAudit, { mobile: device.mobile, phase: 'bottom' })
      result.issues.push(...bottom.issues.map((i) => ({ ...i, at: `p${index}` })))
      const ann = await annotate(page, bottom.issues, path.join(outDir, `${stem}.p${index}.annotated.png`), device)
      if (ann) result.annotated.push(ann)
    }
  }
  if (!reachedBottom) result.notes = [`Còn nội dung chưa chụp (giới hạn --pages ${maxPages}).`]
  await scrollTo(page, 0)

  result.focus = []
  for (const [i, selector] of opts.focus.entries()) {
    log(`chụp cận ${selector}`)
    const el = await page.$(selector)
    if (!el) {
      result.stepErrors.push(`focus ${selector}: không tìm thấy phần tử`)
      continue
    }
    await el.evaluate((node) => node.scrollIntoView({ block: 'center' }))
    await settleFrame(page)
    const file = path.join(outDir, `${stem}.focus${i + 1}.png`)
    await el.screenshot({ path: file })
    result.focus.push(file)
  }

  if (result.resolvedTheme !== theme) {
    result.issues.push({ level: 'error', code: 'page/theme-state', message: `Yêu cầu theme ${theme} nhưng trang đang ở ${result.resolvedTheme}.`, items: [] })
  }
  if (PROJECT.login && !new URL(target.url).pathname.startsWith(PROJECT.login.path) && result.finalPath.startsWith(PROJECT.login.path)) {
    result.issues.push({ level: 'error', code: 'page/redirected-login', message: `Bị chuyển về ${PROJECT.login.path} — chưa đăng nhập được.`, items: [] })
  }
}

// Tải 1 màn hình trên 1 thiết bị × theme rồi chụp. Có `tabs` thì trên cùng trang đó bấm lần lượt từng tab
// (giống người dùng thật, tận dụng dữ liệu đã tải) và chụp mỗi tab thành 1 kết quả riêng.
// log(msg, tab) báo tiến độ; onStart(tab)/onDone(tab, result) để bên gọi đếm số lần chụp.
export async function captureCombo(browser, opts, target, device, theme, outDir, { log = () => {}, tabs = [null], onStart = () => {}, onDone = () => {} } = {}) {
  const record = { console: [], pageErrors: [], network: [], blockedWrites: [] }
  const page = await preparePage(browser, {
    device, theme, dpr: opts.dpr, storage: { ...target.storage, ...opts.storage }, allowWrites: opts.allowWrites, record,
  })
  const selector = tabSelectorFor(target.url, opts.tabSelector)
  const results = []
  try {
    log(`tải trang ${target.url}`, tabs[0])
    await page.goto(target.url, { waitUntil: 'domcontentloaded', timeout: 30_000 })
    log('chờ API tải xong + spinner/skeleton tắt', tabs[0])
    let loadState = await waitForStable(page, { log: (m) => log(m, tabs[0]) })
    for (const tab of tabs) {
      onStart(tab)
      const tlog = (m) => log(m, tab)
      const result = {
        target: tab ? `${target.raw} › ${tab.label}` : target.raw, tab: tab?.label || null, device: device.name,
        viewport: `${device.width}x${device.height}`, mobile: device.mobile, theme,
        shots: [], annotated: [], issues: [], stepErrors: [],
      }
      const stem = `${target.slug}${tab ? `.tab${tab.index}-${slugify(tab.label).slice(0, 30)}` : ''}.${device.name}.${theme}`
      const pending = { api: loadState?.pendingApi || [], loaders: loadState?.loaders || [] }
      const note = (state) => {
        if (state) Object.assign(pending, { api: state.pendingApi, loaders: state.loaders })
      }
      loadState = null
      if (tab) {
        tlog(`mở tab ${tab.index} rồi chờ API của tab`)
        try {
          note(await openTab(page, tab, selector, { log: tlog }))
        } catch (err) {
          result.stepErrors.push(`tab "${tab.label}": ${err.message}`)
          tlog(`không mở được tab: ${err.message}`)
        }
      }
      for (const [i, step] of opts.steps.entries()) {
        tlog(`step ${i + 1}/${opts.steps.length}: ${step}`)
        try {
          note(await runStep(page, step, { log: tlog }))
        } catch (err) {
          result.stepErrors.push(`${step}: ${err.message}`)
          tlog(`step lỗi: ${err.message}`)
          break
        }
      }
      await shootState(page, opts, target, device, theme, outDir, stem, result, tlog)
      drainRecord(record, result)
      for (const err of result.stepErrors) result.issues.push({ level: 'error', code: 'tool/step-failed', message: err, items: [] })
      const waitIssue = pendingIssue(pending)
      if (waitIssue) result.issues.push(waitIssue)
      results.push(result)
      onDone(tab, result)
      // Step mở dialog/menu: đóng lại trước khi sang tab sau.
      if (opts.steps.length && tab !== tabs[tabs.length - 1]) await page.keyboard.press('Escape').catch(() => {})
    }
  } finally {
    await page.close().catch(() => {})
  }
  return results
}

