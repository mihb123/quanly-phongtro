import { spawn } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { FRONTEND_DIR, PROFILE_DIR, PROJECT } from './config.mjs'

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
    const res = await fetch(url, { signal: AbortSignal.timeout(3000) })
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

