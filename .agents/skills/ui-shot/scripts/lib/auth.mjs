import fs from 'node:fs'
import path from 'node:path'
import readline from 'node:readline'
import { PROFILE_DIR, PROJECT } from './config.mjs'
import { readEnvFile } from './env.mjs'

function ask(question, { hidden = false } = {}) {
  return new Promise((resolve) => {
    const rl = readline.createInterface({ input: process.stdin, output: process.stderr, terminal: true })
    let muted = false
    rl._writeToOutput = (s) => { if (!muted) process.stderr.write(s) }
    process.stderr.write(question)
    muted = hidden
    rl.question('', (answer) => {
      if (hidden) process.stderr.write('\n')
      rl.close()
      resolve(answer)
    })
  })
}

async function readPipedLines() {
  let data = ''
  for await (const chunk of process.stdin) data += chunk
  return data.split(/\r?\n/)
}

function quote(value) {
  return value.includes("'") ? value : `'${value}'`
}

export function writeCredentials(source, email, password) {
  const lines = fs.existsSync(source.file) ? fs.readFileSync(source.file, 'utf8').split('\n') : []
  const values = { [source.email]: email, [source.password]: password }
  const pending = new Set(Object.keys(values))
  const out = lines.map((line) => {
    const key = /^\s*([A-Z0-9_]+)\s*=/.exec(line)?.[1]
    if (!key || !pending.has(key)) return line
    pending.delete(key)
    return `${key}=${quote(values[key])}`
  })
  while (out.length && out[out.length - 1] === '') out.pop()
  for (const key of pending) out.push(`${key}=${quote(values[key])}`)
  fs.mkdirSync(path.dirname(source.file), { recursive: true, mode: 0o700 })
  const tmp = `${source.file}.tmp-${process.pid}`
  fs.writeFileSync(tmp, `${out.join('\n')}\n`, { mode: 0o600 })
  fs.renameSync(tmp, source.file)
  fs.chmodSync(source.file, 0o600)
}

export async function promptCredentials(opts) {
  if (!PROJECT.login) throw new Error('Project chưa khai báo "login" trong ui-shot.config.json nên không cần tài khoản')
  const source = PROJECT.credentials[0]
  const current = readEnvFile(source.file)
  let email = opts.email
  let password
  if (process.stdin.isTTY) {
    console.error(`Lưu tài khoản đăng nhập cho ${PROJECT.name} vào ${source.file} (chmod 600)`)
    if (!email) {
      const hint = current[source.email] ? ` [${current[source.email]}]` : ''
      email = (await ask(`Email${hint}: `)).trim() || current[source.email]
    }
    const keep = current[source.password] ? ' (Enter = giữ mật khẩu cũ)' : ''
    password = (await ask(`Mật khẩu${keep}: `, { hidden: true })) || current[source.password]
  } else {
    const lines = await readPipedLines()
    if (!email) email = (lines.shift() || '').trim()
    password = lines[0] || ''
  }
  if (!email || !password) throw new Error('Thiếu email hoặc mật khẩu')
  if (/[\r\n]/.test(email + password)) throw new Error('Email/mật khẩu không được chứa xuống dòng')
  writeCredentials(source, email, password)
  fs.rmSync(PROFILE_DIR, { recursive: true, force: true })
  return source
}
