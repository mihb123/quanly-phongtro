import path from 'node:path'
import { PROJECT } from './config.mjs'
import { pageAudit } from './audit.mjs'
import { preparePage, runStep, scrollTo, settleFrame, waitForStable } from './page.mjs'

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

export async function captureCombo(browser, opts, target, device, theme, outDir) {
  const record = { console: [], pageErrors: [], network: [], blockedWrites: [] }
  const result = {
    target: target.raw, device: device.name, viewport: `${device.width}x${device.height}`, mobile: device.mobile, theme,
    shots: [], annotated: [], issues: [], stepErrors: [], record,
  }
  const page = await preparePage(browser, {
    device, theme, dpr: opts.dpr, storage: { ...target.storage, ...opts.storage }, allowWrites: opts.allowWrites, record,
  })
  const stem = `${target.slug}.${device.name}.${theme}`
  try {
    await page.goto(target.url, { waitUntil: 'domcontentloaded', timeout: 30_000 })
    await waitForStable(page)
    await page.evaluate((mode) => document.documentElement.classList.toggle('dark', mode === 'dark'), theme)
    for (const step of opts.steps) {
      try {
        await runStep(page, step)
      } catch (err) {
        result.stepErrors.push(`${step}: ${err.message}`)
        break
      }
    }
    await page.evaluate((mode) => document.documentElement.classList.toggle('dark', mode === 'dark'), theme)
    await settleFrame(page)
    result.finalPath = new URL(page.url()).pathname
    const maxPages = opts.pages ?? (device.mobile ? 4 : 2)

    await scrollTo(page, 0)
    await settleFrame(page)
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

    result.focus = []
    for (const [i, selector] of opts.focus.entries()) {
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
  } finally {
    await page.close().catch(() => {})
  }
  for (const text of record.pageErrors) result.issues.push({ level: 'error', code: 'runtime/page-error', message: text, items: [] })
  if (record.console.length) result.issues.push({ level: 'warn', code: 'runtime/console-error', message: `${record.console.length} lỗi console`, items: [], samples: [...new Set(record.console)].slice(0, 5) })
  if (record.network.length) result.issues.push({ level: 'warn', code: 'network/failed', message: `${record.network.length} request API lỗi`, items: [], samples: [...new Set(record.network)].slice(0, 5) })
  if (record.blockedWrites.length) result.issues.push({ level: 'info', code: 'network/blocked-write', message: `Đã chặn ${record.blockedWrites.length} request ghi dữ liệu (dùng --allow-writes nếu thật sự cần)`, items: [], samples: [...new Set(record.blockedWrites)].slice(0, 5) })
  for (const err of result.stepErrors) result.issues.push({ level: 'error', code: 'tool/step-failed', message: err, items: [] })
  return result
}

