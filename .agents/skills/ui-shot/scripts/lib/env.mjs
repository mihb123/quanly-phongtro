import { spawn, spawnSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { FRONTEND_DIR, PROFILE_DIR, PROJECT, SHARED_MODULES_DIR, requireFromFrontend, requireShared } from './config.mjs'
import { progress } from './progress.mjs'

export function readEnvFile(file) {
  try {
    const out = {}
    for (const line of fs.readFileSync(file, 'utf8').split('\n')) {
      const match = /^\s*([A-Z0-9_]+)\s*=\s*(.*)\s*$/.exec(line)
      if (match) out[match[1]] = match[2].replace(/^['"]|['"]$/g, '')
    }
    return out
  } catch {
    return {}
  }
}

export function loadCredentials() {
  if (process.env.UI_SHOT_EMAIL && process.env.UI_SHOT_PASSWORD) {
    return { email: process.env.UI_SHOT_EMAIL, password: process.env.UI_SHOT_PASSWORD }
  }
  for (const source of PROJECT.credentials) {
    const env = readEnvFile(source.file)
    if (env[source.email] && env[source.password]) return { email: env[source.email], password: env[source.password] }
  }
  return null
}

export async function reachable(url) {
  try {
    const res = await fetch(url, { signal: AbortSignal.timeout(10_000) })
    return res.status < 500
  } catch {
    return false
  }
}

export async function ensureDevServer(opts) {
  if (await reachable(opts.base)) return null
  if (opts.baseExplicit) throw new Error(`Không truy cập được ${opts.base}`)
  const port = new URL(opts.base).port || '5173'
  const [cmd, ...args] = PROJECT.devCommand.map((part) => part.replaceAll('{port}', port))
  progress(`app chưa chạy — bật dev server: ${[cmd, ...args].join(' ')} (chờ tối đa 45s)`)
  const child = spawn(cmd, args, {
    cwd: FRONTEND_DIR, detached: true, stdio: 'ignore',
  })
  child.unref()
  const deadline = Date.now() + 45_000
  while (Date.now() < deadline) {
    if (await reachable(opts.base)) return child
    if (child.exitCode !== null) break
    await new Promise((r) => setTimeout(r, 500))
  }
  stopDevServer(child)
  throw new Error(`Không khởi động được dev server (${PROJECT.devCommand.join(' ')}) tại ${opts.base}`)
}

export function stopDevServer(child) {
  if (!child) return
  try {
    process.kill(-child.pid, 'SIGTERM')
  } catch {
    try { child.kill('SIGTERM') } catch { /* noop */ }
  }
}

function tryRequire(load) {
  try {
    return load('puppeteer')
  } catch {
    return null
  }
}

function run(cmd, args) {
  console.error(`ui-shot: ${cmd} ${args.join(' ')}`)
  const res = spawnSync(cmd, args, { cwd: SHARED_MODULES_DIR, stdio: ['ignore', 2, 2] })
  if (res.error) throw new Error(`Không chạy được ${cmd}: ${res.error.message}`)
  if (res.status !== 0) throw new Error(`${cmd} ${args.join(' ')} lỗi (exit ${res.status})`)
}

function installSharedPuppeteer() {
  console.error(`ui-shot: chưa có puppeteer dùng chung — cài vào ${SHARED_MODULES_DIR} (một lần cho mọi project)`)
  fs.mkdirSync(SHARED_MODULES_DIR, { recursive: true })
  const pkg = path.join(SHARED_MODULES_DIR, 'package.json')
  if (!fs.existsSync(pkg)) fs.writeFileSync(pkg, '{"private":true}\n')
  run('npm', ['install', '--no-audit', '--no-fund', 'puppeteer'])
}

async function ensureBrowser(puppeteer, dir) {
  let executable = ''
  try {
    executable = await puppeteer.executablePath()
  } catch { /* chưa có Chrome */ }
  if (typeof executable === 'string' && fs.existsSync(executable)) return
  const installer = path.join(dir, 'node_modules', 'puppeteer', 'install.mjs')
  if (!fs.existsSync(installer)) throw new Error(`puppeteer chưa có Chrome và không thấy ${installer} để tải`)
  console.error('ui-shot: tải Chrome cho puppeteer (một lần)')
  const res = spawnSync(process.execPath, [installer], { cwd: dir, stdio: ['ignore', 2, 2], env: { ...process.env, PUPPETEER_SKIP_DOWNLOAD: '', PUPPETEER_CHROME_HEADLESS_SHELL_SKIP_DOWNLOAD: 'true', PUPPETEER_FIREFOX_SKIP_DOWNLOAD: 'true' } })
  if (res.status !== 0) throw new Error(`Tải Chrome cho puppeteer lỗi (exit ${res.status})`)
}

export async function resolvePuppeteer() {
  let puppeteer = tryRequire(requireShared)
  let dir = SHARED_MODULES_DIR
  let installError = null
  if (!puppeteer) {
    try {
      installSharedPuppeteer()
      puppeteer = tryRequire(requireShared)
    } catch (err) {
      installError = err
    }
  }
  if (!puppeteer) {
    puppeteer = tryRequire(requireFromFrontend)
    dir = FRONTEND_DIR
  }
  if (!puppeteer) {
    throw new Error(`Không có puppeteer: cài bản dùng chung vào ${SHARED_MODULES_DIR} thất bại (${installError?.message || 'không rõ lỗi'}).\nCài tay: cd ${SHARED_MODULES_DIR} && npm install puppeteer`)
  }
  await ensureBrowser(puppeteer, dir)
  return puppeteer
}

export function sandboxBlocked() {
  if (process.env.UI_SHOT_NO_SANDBOX === '1' || process.getuid?.() === 0) return true
  try {
    return fs.readFileSync('/proc/sys/kernel/apparmor_restrict_unprivileged_userns', 'utf8').trim() === '1'
  } catch {
    return false
  }
}

export async function launchBrowser(puppeteer) {
  const profile = PROFILE_DIR
  fs.mkdirSync(profile, { recursive: true })
  const args = ['--hide-scrollbars', '--font-render-hinting=none', '--disable-gpu']
  if (sandboxBlocked()) args.push('--no-sandbox')
  try {
    return { browser: await puppeteer.launch({ headless: true, userDataDir: profile, args }), tempProfile: null }
  } catch {
    const tempProfile = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-shot-profile-'))
    return { browser: await puppeteer.launch({ headless: true, userDataDir: tempProfile, args }), tempProfile }
  }
}


export function runDeploy() {
  const cmd = PROJECT.deployCommand
  if (!cmd) return Promise.reject(new Error('Project chưa khai báo "deployCommand" trong ui-shot.config.json'))
  const [bin, args] = Array.isArray(cmd) ? [cmd[0], cmd.slice(1)] : [process.env.SHELL || '/bin/bash', ['-ic', cmd]]
  const label = Array.isArray(cmd) ? cmd.join(' ') : cmd
  const logFile = path.join(os.tmpdir(), `ui-shot-deploy-${PROJECT.slug}.log`)
  const started = Date.now()
  return new Promise((resolve, reject) => {
    const log = fs.openSync(logFile, 'w')
    const child = spawn(bin, args, { cwd: PROJECT.deployDir, stdio: ['ignore', log, log] })
    child.on('error', (err) => {
      fs.closeSync(log)
      reject(new Error(`Không chạy được deployCommand (${label}): ${err.message}`))
    })
    child.on('exit', (code, signal) => {
      fs.closeSync(log)
      const seconds = Math.round((Date.now() - started) / 1000)
      if (code === 0) return resolve({ command: label, dir: PROJECT.deployDir, seconds, log: logFile })
      const tail = fs.readFileSync(logFile, 'utf8').trimEnd().split('\n').slice(-20).join('\n')
      reject(new Error(`deployCommand lỗi (${signal || `exit ${code}`}) sau ${seconds}s: ${label}\n${tail}\nlog đầy đủ: ${logFile}`))
    })
  })
}
