import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'

export const LIB_DIR = path.dirname(fileURLToPath(import.meta.url))
export const CONFIG_NAMES = ['ui-shot.config.json', '.agents/ui-shot.config.json']
export const USER_CONFIG_FILE = path.join(os.homedir(), '.config', 'ui-shot', 'config.json')

function expandHome(p) {
  return p.startsWith('~/') ? path.join(os.homedir(), p.slice(2)) : p
}

function findUp(start, test) {
  let dir = path.resolve(start)
  for (;;) {
    if (test(dir)) return dir
    const parent = path.dirname(dir)
    if (parent === dir) return null
    dir = parent
  }
}

function readJson(file) {
  try {
    return JSON.parse(fs.readFileSync(file, 'utf8'))
  } catch (err) {
    console.error(`Không đọc được ${file}: ${err.message}`)
    process.exit(2)
  }
}

const USER_CONFIG = fs.existsSync(USER_CONFIG_FILE) ? readJson(USER_CONFIG_FILE) : {}

function loadProject() {
  const start = expandHome(process.env.UI_SHOT_PROJECT || process.cwd())
  const configRoot = findUp(start, (d) => CONFIG_NAMES.some((n) => fs.existsSync(path.join(d, n))))
  const root = configRoot || findUp(start, (d) => fs.existsSync(path.join(d, '.git'))) || path.resolve(start)
  const configFile = configRoot ? CONFIG_NAMES.map((n) => path.join(root, n)).find((f) => fs.existsSync(f)) : null
  const config = configFile ? readJson(configFile) : {}
  const frontendGuess = fs.existsSync(path.join(root, 'frontend', 'package.json')) ? 'frontend' : '.'
  const name = config.name || path.basename(root)
  return {
    name,
    slug: name.replace(/[^a-zA-Z0-9_-]+/g, '-'),
    root,
    configFile,
    frontendDir: path.resolve(root, config.frontendDir || frontendGuess),
    baseUrl: (config.baseUrl || 'http://127.0.0.1:5173').replace(/\/$/, ''),
    devCommand: config.devCommand || ['pnpm', 'exec', 'vite', '--host', '127.0.0.1', '--port', '{port}', '--strictPort'],
    defaultTargets: config.defaultTargets || ['/'],
    deployCommand: config.deployCommand || null,
    deployDir: path.resolve(root, expandHome(config.deployDir || '.')),
    tabs: config.tabs || null,
    tabSelectors: config.tabSelectors || {},
    loadingSelectors: config.loadingSelectors || [],
    ignoreRequests: (config.ignoreRequests || []).map((s) => new RegExp(s)),
    themeStorageKey: config.themeStorageKey || null,
    apiPrefix: config.apiPrefix || '/api/',
    allowedWrites: (config.allowedWrites || []).map((s) => new RegExp(s)),
    login: config.login
      ? { path: '/login', email: '#email', password: '#password', submit: 'button[type="submit"]', ...config.login }
      : null,
    credentials: (config.credentials || [{ file: `~/.config/ui-shot/${name.replace(/[^a-zA-Z0-9_-]+/g, '-')}.env` }]).map((c) => ({
      file: expandHome(c.file), email: c.email || 'UI_SHOT_EMAIL', password: c.password || 'UI_SHOT_PASSWORD',
    })),
    cookies: config.cookies || [],
    uploadUrl: config.uploadUrl || null,
  }
}

export const PROJECT = loadProject()
export const FRONTEND_DIR = PROJECT.frontendDir
export const requireFromFrontend = createRequire(path.join(FRONTEND_DIR, 'package.json'))
export const SHARED_MODULES_DIR = path.join(os.homedir(), '.cache', 'ui-shot')
export const requireShared = createRequire(path.join(SHARED_MODULES_DIR, 'package.json'))

export const DEVICES = {
  'mobile-s': { width: 360, height: 740, mobile: true },
  mobile: { width: 390, height: 844, mobile: true },
  tablet: { width: 768, height: 1024, mobile: true },
  desktop: { width: 1440, height: 900, mobile: false },
}
export const DEFAULT_DEVICES = ['mobile-s', 'desktop']
export const DEFAULT_BASE = PROJECT.baseUrl
export const MOBILE_UA = 'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Mobile Safari/537.36'
export const FREEZE_CSS = `*,*::before,*::after{transition-duration:0s!important;transition-delay:0s!important;animation-duration:0s!important;animation-delay:0s!important;animation-iteration-count:1!important;caret-color:transparent!important;scroll-behavior:auto!important}`
export const OUT_ROOT = path.join(os.tmpdir(), 'ui-shot')
export const PROFILE_DIR = path.join(os.homedir(), '.cache', 'ui-shot', PROJECT.slug, 'profile')
export const UPLOAD_BASE = (process.env.UI_SHOT_UPLOAD_URL || PROJECT.uploadUrl || USER_CONFIG.uploadUrl || '').replace(/\/+$/, '')
