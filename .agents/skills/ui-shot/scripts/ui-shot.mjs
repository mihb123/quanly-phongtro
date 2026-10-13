#!/usr/bin/env node
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { captureCombo } from './lib/capture.mjs'
import { promptCredentials } from './lib/auth.mjs'
import { parseArgs, resolveDevice, resolveTarget, timestamp } from './lib/cli.mjs'
import { OUT_ROOT, FRONTEND_DIR, PROJECT, requireFromFrontend } from './lib/config.mjs'
import { ensureDevServer, launchBrowser, stopDevServer } from './lib/env.mjs'
import { login } from './lib/page.mjs'
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

function loadPuppeteer() {
  try {
    return requireFromFrontend('puppeteer')
  } catch {
    fail(`Không tìm thấy puppeteer trong ${FRONTEND_DIR}/node_modules — chạy: cd ${FRONTEND_DIR} && pnpm add -D puppeteer (hoặc npm i -D puppeteer)`)
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
  const puppeteer = loadPuppeteer()
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

async function captureAll(browser, opts, targets, devices, outDir) {
  const results = []
  for (const target of targets) {
    for (const device of devices) {
      for (const theme of opts.themes) {
        results.push(await captureCombo(browser, opts, target, device, theme, outDir))
      }
    }
  }
  for (const r of results) delete r.record
  return results
}

async function main() {
  let opts
  try {
    opts = parseArgs(process.argv.slice(2))
  } catch (err) {
    fail(`${err.message}\nXem: ui-shot --help`)
  }
  if (opts.command === 'clean') return cleanAll()
  if (opts.command === 'login') return loginCommand(opts)

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

  const puppeteer = loadPuppeteer()

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
    results: [], totals: { error: 0, warn: 0, info: 0 }, visualReview: 'pending',
  }
  try {
    const needsApp = targets.some((t) => t.url.startsWith(opts.base))
    if (needsApp) devServer = await ensureDevServer(opts)
    report.devServer = devServer ? 'started' : 'existing'
    ;({ browser, tempProfile } = await launchBrowser(puppeteer))
    if (opts.login && PROJECT.login && needsApp) report.auth = await login(browser, opts)
    report.results = await captureAll(browser, opts, targets, devices, outDir)
    for (const r of report.results) for (const i of r.issues) report.totals[i.level] += 1
    report.pdf = await writePdf(browser, report, outDir)
    fs.writeFileSync(path.join(outDir, 'report.json'), JSON.stringify(report, null, 2))
  } catch (err) {
    await cleanup()
    fail(`ui-shot lỗi: ${err.message}\nFile đã sinh (nếu có): ${outDir}`)
  }
  await cleanup()

  let uploadFailed = false
  if (opts.upload) {
    try {
      report.upload = await uploadResults(outDir, runName)
    } catch (err) {
      report.upload = { error: err.message }
      uploadFailed = true
    }
  }
  report.kept = keep || !opts.upload || uploadFailed
  if (!report.kept) fs.rmSync(outDir, { recursive: true, force: true })

  if (opts.json) console.log(JSON.stringify(report, null, 2))
  else printSummary(report)
  if (opts.open) {
    const target = report.upload?.pdfUrl || (report.kept ? report.pdf : null)
    if (target) spawn('xdg-open', [target], { detached: true, stdio: 'ignore' }).unref()
  }
  process.exit(uploadFailed ? 2 : report.totals.error ? 1 : 0)
}

main()
