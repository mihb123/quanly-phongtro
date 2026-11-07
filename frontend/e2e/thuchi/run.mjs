import { execFileSync, spawn } from 'node:child_process'
import { randomBytes } from 'node:crypto'
import { createWriteStream, existsSync, mkdirSync, readdirSync, rmSync } from 'node:fs'
import path from 'node:path'
import { ARTIFACTS_DIR, BASE_URL, BINARY, DB_NAME, DB_URL, E2E_DIR, FRONTEND_DIR, PG_ADMIN_URL, PORT, ROOT_DIR, SERVER_LOG, TELEGRAM_FAKE_URL, TMP_DIR } from './lib/config.mjs'
import { psql } from './lib/db.mjs'
import { FAKE_ALLOWED_GROUP, FAKE_BOT_TOKEN, startFakeTelegram } from './lib/telegram-fake.mjs'

const args = process.argv.slice(2)
const filters = args.filter((a) => !a.startsWith('-'))
const passthrough = args.filter((a) => a.startsWith('-'))
const concurrency = process.env.E2E_CONCURRENCY || '3'

const log = (msg) => console.log(`\n\x1b[36m▶ ${msg}\x1b[0m`)
const hex = (n) => randomBytes(n).toString('hex')

async function health() {
  try {
    const res = await fetch(`${BASE_URL}/health`)
    return res.ok
  } catch {
    return false
  }
}

async function prepareDatabase() {
  if (!/e2e/i.test(DB_NAME)) throw new Error(`Tên DB "${DB_NAME}" phải chứa "e2e" (runner sẽ xoá và tạo lại DB này)`)
  try {
    await psql(PG_ADMIN_URL, 'select 1')
  } catch (err) {
    console.error(`Không kết nối được Postgres (${PG_ADMIN_URL}). Bật cluster dev theo AGENTS.md:\n  /usr/lib/postgresql/18/bin/pg_ctl -D ~/.local/share/thu-chi/pgdata -l ~/.local/share/thu-chi/pg.log -o "-p 5440 -k /tmp -c listen_addresses=127.0.0.1" start\n${err.message}`)
    process.exit(1)
  }
  log(`Tạo lại database ${DB_NAME} và chạy migration`)
  await psql(PG_ADMIN_URL, `DROP DATABASE IF EXISTS "${DB_NAME}" WITH (FORCE)`)
  await psql(PG_ADMIN_URL, `CREATE DATABASE "${DB_NAME}"`)
  const dir = path.join(ROOT_DIR, 'migrations')
  for (const f of readdirSync(dir).filter((f) => f.endsWith('.up.sql')).sort()) {
    execFileSync('psql', [DB_URL, '-X', '-q', '-v', 'ON_ERROR_STOP=1', '-f', path.join(dir, f)], { stdio: ['ignore', 'ignore', 'inherit'], env: { ...process.env, PGOPTIONS: '-c client_min_messages=warning' } })
  }
}

function build() {
  if (process.env.E2E_SKIP_BUILD && existsSync(BINARY)) {
    log(`Bỏ qua build, dùng ${BINARY}`)
    return
  }
  log('Build frontend + binary (make build)')
  execFileSync('make', ['build', `BINARY=${BINARY}`], { cwd: ROOT_DIR, stdio: 'inherit' })
}

async function startServer() {
  if (await health()) throw new Error(`Cổng ${PORT} đang có server khác chạy, đặt E2E_PORT khác`)
  rmSync(SERVER_LOG, { force: true })
  const out = createWriteStream(SERVER_LOG)
  const host = new URL(BASE_URL).host
  const env = {
    PATH: process.env.PATH,
    HOME: process.env.HOME,
    APP_ENV: 'dev',
    APP_PORT: String(PORT),
    APP_PORT_DEV: String(PORT),
    APP_URL: host,
    APP_URL_DEV: host,
    COOKIE_SECURE: 'false',
    TRUSTED_PROXIES: '127.0.0.1/32,::1/128',
    POSTGRES_DSN: DB_URL,
    ACCESS_TOKEN_JWT_SECRET: hex(32),
    REFRESH_TOKEN_JWT_SECRET: hex(32),
    APP_SECRET_ENCRYPTION_KEY: hex(32),
    ZALO_BOT_ENCRYPTION_KEY: hex(32),
    JWT_TTL_MINUTES: '60',
    OTP_EXPIRE_MINUTES: '10',
    SMTP_HOST: '127.0.0.1',
    SMTP_PORT: '1',
    SMTP_USERNAME: 'e2e',
    SMTP_PASSWORD: 'e2e',
    PLUGIN_DIR: path.join(TMP_DIR, 'plugins'),
    THUCHI_TELEGRAM_BOT_TOKEN: FAKE_BOT_TOKEN,
    THUCHI_TELEGRAM_API_BASE: `${TELEGRAM_FAKE_URL}/bot`,
    THUCHI_TELEGRAM_CHAT_ID: String(FAKE_ALLOWED_GROUP),
  }
  log(`Chạy server ${BASE_URL} (log: ${SERVER_LOG})`)
  const proc = spawn(BINARY, [], { cwd: TMP_DIR, env, stdio: ['ignore', 'pipe', 'pipe'] })
  proc.stdout.pipe(out)
  proc.stderr.pipe(out)
  let exited = null
  proc.on('exit', (code) => (exited = code))
  for (let i = 0; i < 100; i++) {
    if (exited !== null) throw new Error(`Server thoát sớm (code ${exited}), xem ${SERVER_LOG}`)
    if (await health()) return proc
    await new Promise((r) => setTimeout(r, 100))
  }
  proc.kill()
  throw new Error(`Server không lên sau 10 giây, xem ${SERVER_LOG}`)
}

function testFiles() {
  const dir = path.join(E2E_DIR, 'tests')
  const all = readdirSync(dir).filter((f) => f.endsWith('.test.mjs')).sort()
  const picked = filters.length ? all.filter((f) => filters.some((q) => f.includes(q))) : all
  if (!picked.length) throw new Error(`Không có file test nào khớp ${filters.join(', ')}`)
  return picked.map((f) => path.join(dir, f))
}

async function main() {
  mkdirSync(TMP_DIR, { recursive: true })
  rmSync(ARTIFACTS_DIR, { recursive: true, force: true })
  const files = testFiles()
  await prepareDatabase()
  build()
  const fakeTelegram = await startFakeTelegram(Number(new URL(TELEGRAM_FAKE_URL).port))
  const server = await startServer()
  const stop = () => server.exitCode === null && server.kill('SIGTERM')
  process.on('SIGINT', () => {
    stop()
    process.exit(130)
  })
  log(`Chạy ${files.length} file test (song song ${concurrency})`)
  const code = await new Promise((resolve) => {
    const child = spawn(process.execPath, ['--test', `--test-concurrency=${concurrency}`, '--test-timeout=240000', '--test-force-exit', ...passthrough, ...files], {
      cwd: FRONTEND_DIR,
      stdio: 'inherit',
      env: { ...process.env, E2E_TMP: TMP_DIR, E2E_SERVER_LOG: SERVER_LOG, E2E_DB_URL: DB_URL, E2E_BASE_URL: BASE_URL, E2E_TELEGRAM_FAKE_URL: TELEGRAM_FAKE_URL },
    })
    child.on('exit', (c) => resolve(c ?? 1))
  })
  stop()
  fakeTelegram.close()
  console.log(`\nDB giữ lại để soát: ${DB_URL}\nLog server: ${SERVER_LOG}${code ? `\nẢnh chụp/log lỗi: ${ARTIFACTS_DIR}` : ''}`)
  process.exit(code)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
