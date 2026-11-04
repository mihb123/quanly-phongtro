#!/usr/bin/env node
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { captureCombo } from './lib/capture.mjs'
import { promptCredentials } from './lib/auth.mjs'
import { parseArgs, resolveDevice, resolveTarget, timestamp } from './lib/cli.mjs'
import { OUT_ROOT, PROJECT } from './lib/config.mjs'
import { ensureDevServer, launchBrowser, resolvePuppeteer, runDeploy, stopDevServer } from './lib/env.mjs'
import { findTabs, login, preparePage, setApiTimeout, tabSelectorFor, waitForStable } from './lib/page.mjs'
import { elapsed, progress, setQuiet } from './lib/progress.mjs'
import { printSummary, writePdf } from './lib/report.mjs'
import { uploadResults } from './lib/upload.mjs'

function fail(message) {
  console.error(message)
  process.exit(2)
}

function cleanAll() {
  const entries = fs.existsSync(OUT_ROOT) ? fs.readdirSync(OUT_ROOT) : []
  for (const entry of entries) fs.rmSync(path.join(OUT_ROOT, entry), { recursive: true, force: true })
  console.log(`Đã xoá ${entries.length} thư mục ảnh tạm trong ${OUT_ROOT}`)
}

async function loadPuppeteer() {
  try {
    return await resolvePuppeteer()
  } catch (err) {
    fail(err.message)
  }
}

async function loginCommand(opts) {
  let source
  try {
    source = await promptCredentials(opts)
  } catch (err) {
    fail(`ui-shot login: ${err.message}`)
  }
  console.log(`Đã lưu tài khoản vào ${source.file} (chmod 600), đã xoá phiên đăng nhập cũ.`)
  if (process.env.UI_SHOT_EMAIL && process.env.UI_SHOT_PASSWORD) {
    console.log('Lưu ý: đang đặt biến UI_SHOT_EMAIL/UI_SHOT_PASSWORD, biến này được ưu tiên hơn file.')
  }
  if (!opts.verify) return
  const puppeteer = await loadPuppeteer()
  let devServer = null
  let browser = null
  let tempProfile = null
  try {
    devServer = await ensureDevServer(opts)
    ;({ browser, tempProfile } = await launchBrowser(puppeteer))
    await login(browser, opts)
    console.log(`Đăng nhập thử OK tại ${opts.base}, phiên đã lưu cho các lần chụp sau.`)
  } catch (err) {
    console.error(`Đăng nhập thử thất bại: ${err.message}\nKiểm tra lại tài khoản rồi chạy lại: ui-shot login`)
    process.exitCode = 2
  } finally {
    await browser?.close().catch(() => {})
    if (tempProfile) fs.rmSync(tempProfile, { recursive: true, force: true })
    stopDevServer(devServer)
  }
}

async function deploy() {
  progress(`deploy: ${Array.isArray(PROJECT.deployCommand) ? PROJECT.deployCommand.join(' ') : PROJECT.deployCommand} (trong ${PROJECT.deployDir})...`)
  const result = await runDeploy()
  progress(`deploy: xong sau ${result.seconds}s (log: ${result.log})`)
  return result
}

async function deployCommand() {
  try {
    await deploy()
  } catch (err) {
    fail(`ui-shot deploy: ${err.message}`)
  }
}

// Mở màn hình (desktop, sáng), chờ API xong rồi dò thanh tab.
async function discoverTabs(browser, opts, target) {
  const record = { console: [], pageErrors: [], network: [], blockedWrites: [] }
  const page = await preparePage(browser, {
    device: resolveDevice('desktop'), theme: 'light', dpr: 1, storage: { ...target.storage, ...opts.storage }, allowWrites: opts.allowWrites, record,
  })
  try {
    progress(`${target.raw}: dò tab — tải trang + chờ API`)
    await page.goto(target.url, { waitUntil: 'domcontentloaded', timeout: 30_000 })
    await waitForStable(page, { log: (msg) => progress(`${target.raw}: ${msg}`) })
    const selector = tabSelectorFor(target.url, opts.tabSelector)
    const found = await findTabs(page, selector)
    progress(`${target.raw}: ${found.tabs.length} tab${found.how ? ` (nhận theo ${found.how === 'selector' ? selector : found.how})` : ''}${found.tabs.length ? ': ' + found.tabs.map((t) => `${t.index}. ${t.label}`).join(' | ') : ''}`)
    return found.tabs
  } finally {
    await page.close().catch(() => {})
  }
}

// --tabs all | tên | số thứ tự → danh sách tab cần chụp.
function pickTabs(found, wanted, target) {
  const list = found.map((t) => `${t.index}. ${t.label}`).join(' | ') || '(không thấy thanh tab — thử --tab-selector <css>)'
  if (!found.length) throw new Error(`${target.raw}: không tìm thấy tab nào. ${list}`)
  if (wanted.some((w) => w.toLowerCase() === 'all')) return found
  const norm = (s) => s.toLowerCase().replace(/\s+/g, ' ').trim()
  return wanted.map((w) => {
    const hit = /^\d+$/.test(w)
      ? found.find((t) => t.index === Number(w))
      : found.find((t) => norm(t.label) === norm(w)) || found.find((t) => norm(t.label).includes(norm(w)))
    if (!hit) throw new Error(`${target.raw}: không có tab "${w}". Các tab: ${list}`)
    return hit
  })
}

async function captureAll(browser, opts, targets, devices, outDir) {
  const results = []
  const plan = []
  for (const target of targets) {
    const tabs = opts.tabs ? pickTabs(await discoverTabs(browser, opts, target), opts.tabs, target) : [null]
    for (const device of devices) for (const theme of opts.themes) plan.push({ target, tabs, device, theme })
  }
  const total = plan.reduce((sum, p) => sum + p.tabs.length, 0)
  if (opts.tabs) progress(`kế hoạch: ${total} lần chụp (mỗi thiết bị × theme tải trang 1 lần rồi bấm lần lượt từng tab)`)
  let n = 0
  for (const { target, tabs, device, theme } of plan) {
    const tagOf = (tab, index) => `[${index}/${total}] ${target.raw}${tab ? ` › ${tab.label}` : ''} · ${device.name} ${device.width}x${device.height} · ${theme}`
    let started = Date.now()
    const out = await captureCombo(browser, opts, target, device, theme, outDir, {
      tabs,
      log: (msg, tab) => progress(`${tagOf(tab, Math.max(n, 1))}: ${msg}`),
      onStart: (tab) => {
        n += 1
        started = Date.now()
        progress(`${tagOf(tab, n)}: bắt đầu`)
      },
      onDone: (tab, result) => {
        const count = (level) => result.issues.filter((i) => i.level === level).length
        progress(`${tagOf(tab, n)}: xong sau ${elapsed(started)} — ${count('error')} lỗi, ${count('warn')} cảnh báo, ${result.shots.length} ảnh`)
      },
    })
    results.push(...out)
  }
  return results
}

async function main() {
  let opts
  try {
    opts = parseArgs(process.argv.slice(2))
  } catch (err) {
    fail(`${err.message}\nXem: ui-shot --help`)
  }
  setQuiet(opts.quiet)
  setApiTimeout(opts.waitApi)
  if (opts.command === 'clean') return cleanAll()
  if (opts.command === 'login') return loginCommand(opts)
  if (opts.command === 'deploy') return deployCommand()

  let devices
  let targets
  try {
    devices = opts.devices.map(resolveDevice)
    targets = opts.targets.map((t) => resolveTarget(t, opts.base))
  } catch (err) {
    fail(`${err.message}\nXem: ui-shot --help`)
  }
  const runName = `${timestamp()}-${targets.map((t) => t.slug).join('+').slice(0, 60)}`
  const keep = opts.keep || Boolean(opts.out)
  const outDir = opts.out || path.join(OUT_ROOT, runName)
  fs.mkdirSync(outDir, { recursive: true })
  const total = targets.length * devices.length * opts.themes.length
  const tabNote = opts.tabs ? ` × tab (${opts.tabs.join(', ')})` : ''
  progress(`ui-shot ${PROJECT.name}: ${targets.length} màn${tabNote} × ${devices.length} thiết bị × ${opts.themes.length} theme${opts.tabs ? '' : ` = ${total} lần chụp`} → ${outDir}`)

  progress('chuẩn bị puppeteer + Chrome')
  const puppeteer = await loadPuppeteer()

  let deployed = null
  if (opts.deploy) {
    try {
      deployed = await deploy()
    } catch (err) {
      fail(`ui-shot --deploy: ${err.message}`)
    }
  }

  let devServer = null
  let browser = null
  let tempProfile = null
  const cleanup = async () => {
    await browser?.close().catch(() => {})
    if (tempProfile) fs.rmSync(tempProfile, { recursive: true, force: true })
    stopDevServer(devServer)
  }
  process.on('SIGINT', () => { cleanup().finally(() => process.exit(130)) })

  const report = {
    tool: 'ui-shot', project: PROJECT.name, config: PROJECT.configFile, base: opts.base, startedAt: new Date().toISOString(), out: outDir,
    targets: opts.targets, devices: devices.map((d) => d.name), themes: opts.themes, steps: opts.steps,
    deploy: deployed, results: [], totals: { error: 0, warn: 0, info: 0 }, visualReview: 'pending',
  }
  try {
    const needsApp = targets.some((t) => t.url.startsWith(opts.base))
    if (needsApp) {
      progress(`kiểm tra app tại ${opts.base}`)
      devServer = await ensureDevServer(opts)
      progress(devServer ? 'dev server đã sẵn sàng' : 'app đang chạy sẵn')
    }
    report.devServer = devServer ? 'started' : 'existing'
    progress('mở Chrome headless')
    ;({ browser, tempProfile } = await launchBrowser(puppeteer))
    if (opts.login && PROJECT.login && needsApp) {
      progress('kiểm tra đăng nhập (dùng phiên đã lưu, hết phiên thì điền form)')
      report.auth = await login(browser, opts)
      progress(report.auth === 'session' ? 'đăng nhập: phiên cũ còn hiệu lực' : 'đăng nhập: đã đăng nhập lại bằng tài khoản đã lưu')
    }
    if (opts.listTabs) {
      for (const target of targets) {
        const tabs = await discoverTabs(browser, opts, target)
        console.log(`${target.raw}: ${tabs.length ? '' : 'không thấy thanh tab (thử --tab-selector <css>)'}`)
        for (const t of tabs) console.log(`  ${t.index}. ${t.label}${t.active ? '  (đang mở)' : ''}`)
      }
      await cleanup()
      fs.rmSync(outDir, { recursive: true, force: true })
      process.exit(0)
    }
    report.results = await captureAll(browser, opts, targets, devices, outDir)
    for (const r of report.results) for (const i of r.issues) report.totals[i.level] += 1
    progress('tạo report.pdf')
    report.pdf = await writePdf(browser, report, outDir)
    fs.writeFileSync(path.join(outDir, 'report.json'), JSON.stringify(report, null, 2))
  } catch (err) {
    await cleanup()
    fail(`ui-shot lỗi: ${err.message}\nFile đã sinh (nếu có): ${outDir}`)
  }
  await cleanup()

  let uploadFailed = false
  if (opts.upload) {
    progress('upload báo cáo')
    try {
      report.upload = await uploadResults(outDir, runName)
    } catch (err) {
      report.upload = { error: err.message }
      uploadFailed = true
    }
  }
  report.kept = keep || !opts.upload || uploadFailed
  if (!report.kept) fs.rmSync(outDir, { recursive: true, force: true })

  progress(`hoàn tất sau ${elapsed()} — ${report.totals.error} lỗi, ${report.totals.warn} cảnh báo`)
  if (opts.json) console.log(JSON.stringify(report, null, 2))
  else printSummary(report)
  if (opts.open) {
    const target = report.upload?.pdfUrl || (report.kept ? report.pdf : null)
    if (target) spawn('xdg-open', [target], { detached: true, stdio: 'ignore' }).unref()
  }
  process.exit(uploadFailed ? 2 : report.totals.error ? 1 : 0)
}

main()
