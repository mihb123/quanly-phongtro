import { execFile } from 'node:child_process'
import { createHash } from 'node:crypto'
import { promisify } from 'node:util'
import { DB_URL } from './config.mjs'

const exec = promisify(execFile)

export function lit(v) {
  if (v === null || v === undefined) return 'NULL'
  if (typeof v === 'number' || typeof v === 'bigint') return String(v)
  if (typeof v === 'boolean') return v ? 'TRUE' : 'FALSE'
  if (v instanceof Date) return `'${v.toISOString()}'`
  return `'${String(v).replace(/'/g, "''")}'`
}

export function sql(strings, ...values) {
  return strings.reduce((acc, s, i) => acc + s + (i < values.length ? lit(values[i]) : ''), '')
}

export async function psql(url, query) {
  const { stdout } = await exec('psql', [url, '-X', '-q', '-At', '-v', 'ON_ERROR_STOP=1', '-c', query], { maxBuffer: 64 << 20 })
  return stdout
}

export async function queryRows(query) {
  const out = await psql(DB_URL, `select coalesce(json_agg(t), '[]'::json) from (${query}) t`)
  return JSON.parse(out)
}

export const rows = (strings, ...values) => queryRows(sql(strings, ...values))
export const row = async (strings, ...values) => (await queryRows(sql(strings, ...values)))[0]
export const value = async (strings, ...values) => {
  const r = (await queryRows(sql(strings, ...values)))[0]
  return r ? Object.values(r)[0] : undefined
}
export const execSql = (strings, ...values) => psql(DB_URL, sql(strings, ...values))

export async function waitFor(fn, { timeout = 15000, interval = 200, message = 'điều kiện' } = {}) {
  const deadline = Date.now() + timeout
  let last
  let lastErr
  for (;;) {
    try {
      last = await fn()
      if (last) return last
    } catch (err) {
      lastErr = err
    }
    if (Date.now() > deadline) {
      const detail = lastErr ? ` (lỗi cuối: ${lastErr.message})` : ` (giá trị cuối: ${JSON.stringify(last)})`
      throw new Error(`Hết thời gian chờ ${message}${detail}`)
    }
    await new Promise((r) => setTimeout(r, interval))
  }
}

export function accountHash(owner, accountNumber) {
  const digits = accountNumber.replace(/\D/g, '')
  return createHash('sha256').update(`${owner}:${digits}`).digest('hex')
}

export const tx = {
  live: (owner) => rows`select * from thuchi_transactions where owner_id = ${owner} and deleted_at is null order by occurred_at`,
  get: (id) => row`select * from thuchi_transactions where id = ${id}`,
  byMerchant: (owner, merchant) => row`select * from thuchi_transactions where owner_id = ${owner} and merchant = ${merchant} and deleted_at is null order by created_at desc limit 1`,
}

export const account = {
  byName: (owner, name) => row`select * from thuchi_accounts where owner_id = ${owner} and name = ${name} and deleted_at is null`,
  get: (id) => row`select * from thuchi_accounts where id = ${id}`,
}

export const category = {
  byKey: (owner, key, kind = 'expense') => row`select * from thuchi_categories where owner_id = ${owner} and key = ${key} and kind = ${kind} and deleted_at is null`,
  byName: (owner, name) => row`select * from thuchi_categories where owner_id = ${owner} and name = ${name} and deleted_at is null`,
}

export function rangeTotals(owner, start, end) {
  return row`
    select
      coalesce(sum(case when t.kind = 'refund' then -t.amount when t.direction = 'out' then t.amount else 0 end), 0)::bigint as expense,
      coalesce(sum(case when t.kind <> 'refund' and t.direction = 'in' then t.amount else 0 end), 0)::bigint as income
    from thuchi_transactions t
    left join thuchi_accounts a on a.id = t.account_id
    where t.owner_id = ${owner} and t.deleted_at is null and not t.excluded
      and t.kind not in ('transfer', 'card_payment', 'adjustment', 'borrow', 'lend', 'repay', 'collect')
      and coalesce(case when a.deleted_at is null then a.include_in_stats end, true)
      and t.occurred_at >= ${start} and t.occurred_at < ${end}`
}

export function monthTotals(owner, { y, m }) {
  return rangeTotals(owner, new Date(Date.UTC(y, m - 1, 1, -7)), new Date(Date.UTC(y, m, 1, -7)))
}
