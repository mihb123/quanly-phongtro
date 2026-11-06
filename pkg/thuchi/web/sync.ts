import type { Table } from 'dexie'
import { applyBalanceDelta, type ThuChiDB } from './db'
import type { BalanceDelta, Entity, OutboxItem, PullResult, PushResult, Transport } from './types'

export type SyncState = 'idle' | 'syncing' | 'offline' | 'error'

export interface SyncStatus {
  state: SyncState
  pending: number
  lastSyncAt: number | null
  error: string | null
  rejected: { mid: string; error: string; at: number }[]
}

const API = '/plugins/thu-chi'
const PUSH_BATCH = 200
const PULL_LIMIT = 1000
const DATE_FIELDS = ['occurred_at', 'created_at', 'updated_at', 'deleted_at', 'balance_at', 'received_at', 'start_at', 'end_at', 'next_at']

export function uuidv7(): string {
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  const ms = Date.now()
  bytes[0] = (ms / 2 ** 40) & 0xff
  bytes[1] = (ms / 2 ** 32) & 0xff
  bytes[2] = (ms / 2 ** 24) & 0xff
  bytes[3] = (ms / 2 ** 16) & 0xff
  bytes[4] = (ms / 2 ** 8) & 0xff
  bytes[5] = ms & 0xff
  bytes[6] = (bytes[6] & 0x0f) | 0x70
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const h = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`
}

function normalizeDates<T extends object>(row: T): T {
  const r = row as Record<string, unknown>
  for (const f of DATE_FIELDS) {
    const v = r[f]
    if (typeof v === 'string' && v) {
      const d = new Date(v)
      if (!Number.isNaN(d.getTime())) r[f] = d.toISOString()
    }
  }
  return row
}

function isNetworkError(err: unknown) {
  const e = err as { response?: unknown; code?: string; message?: string }
  return !e?.response && (e?.code === 'ERR_NETWORK' || e?.code === 'ECONNABORTED' || /network/i.test(e?.message ?? ''))
}

export class SyncEngine {
  private listeners = new Set<() => void>()
  private running: Promise<void> | null = null
  private again = false
  private timer: ReturnType<typeof setTimeout> | null = null
  private interval: ReturnType<typeof setInterval> | null = null
  private cleanup: (() => void)[] = []
  status: SyncStatus = { state: 'idle', pending: 0, lastSyncAt: null, error: null, rejected: [] }

  readonly db: ThuChiDB
  private transport: Transport

  constructor(db: ThuChiDB, transport: Transport) {
    this.db = db
    this.transport = transport
  }

  subscribe = (fn: () => void) => {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  getStatus = () => this.status

  private setStatus(patch: Partial<SyncStatus>) {
    this.status = { ...this.status, ...patch }
    this.listeners.forEach((fn) => fn())
  }

  start() {
    const onOnline = () => this.sync()
    const onVisible = () => {
      if (document.visibilityState === 'visible') this.sync()
    }
    window.addEventListener('online', onOnline)
    window.addEventListener('offline', () => this.setStatus({ state: 'offline' }))
    document.addEventListener('visibilitychange', onVisible)
    this.interval = setInterval(() => this.sync(), 60_000)
    this.cleanup.push(() => {
      window.removeEventListener('online', onOnline)
      document.removeEventListener('visibilitychange', onVisible)
      if (this.interval) clearInterval(this.interval)
      if (this.timer) clearTimeout(this.timer)
    })
    void this.db.getMeta<number>('lastSyncAt').then((at) => at && this.setStatus({ lastSyncAt: at }))
    void this.refreshPending()
    void this.sync()
  }

  stop() {
    this.cleanup.forEach((fn) => fn())
    this.cleanup = []
  }

  schedule(delay = 800) {
    if (this.timer) clearTimeout(this.timer)
    this.timer = setTimeout(() => void this.sync(), delay)
  }

  async refreshPending() {
    this.setStatus({ pending: await this.db.outbox.count() })
  }

  sync(): Promise<void> {
    if (this.running) {
      this.again = true
      return this.running
    }
    const run = async () => {
      do {
        this.again = false
        await this.runOnce()
      } while (this.again)
    }
    const locks = typeof navigator !== 'undefined' ? navigator.locks : undefined
    const p = ((locks ? locks.request('thuchi_sync', run) : run()) as Promise<void>).finally(() => {
      this.running = null
    })
    this.running = p
    return p
  }

  private async runOnce() {
    if (typeof navigator !== 'undefined' && !navigator.onLine) {
      this.setStatus({ state: 'offline' })
      return
    }
    this.setStatus({ state: 'syncing', error: null })
    try {
      await this.push()
      await this.pull()
      const now = Date.now()
      await this.db.setMeta('lastSyncAt', now)
      this.setStatus({ state: 'idle', lastSyncAt: now })
    } catch (err) {
      if (isNetworkError(err)) {
        this.setStatus({ state: 'offline' })
      } else {
        this.setStatus({ state: 'error', error: err instanceof Error ? err.message : String(err) })
      }
    } finally {
      await this.refreshPending()
    }
  }

  private async push() {
    for (;;) {
      const batch = await this.db.outbox.orderBy('seq').limit(PUSH_BATCH).toArray()
      if (batch.length === 0) return
      const mutations = batch.map((b) => {
        const m = { ...b }
        delete m.seq
        delete m.drop
        delete m.balance
        return m
      })
      const res = await this.transport<{ data: PushResult }>('POST', `${API}/sync/push`, { mutations })
      const results = new Map(res.data.results.map((r) => [r.mid, r]))
      const rejected: SyncStatus['rejected'] = []
      const dropLocal: { entity: Entity; id: string }[] = []
      const revert: BalanceDelta[] = []
      for (const item of batch) {
        const r = results.get(item.mid)
        if (r && !r.ok) {
          rejected.push({ mid: item.mid, error: r.error ?? 'rejected', at: Date.now() })
          if (item.op === 'upsert' && item.entity && item.id) dropLocal.push({ entity: item.entity, id: item.id })
          if (item.drop) dropLocal.push(...item.drop)
          if (item.balance) revert.push(item.balance)
        }
      }
      await this.db.transaction('rw', [this.db.outbox, this.db.accounts, this.db.categories, this.db.transactions, this.db.rules, this.db.budgets, this.db.recurrences], async () => {
        await this.db.outbox.bulkDelete(batch.map((b) => b.seq!))
        for (const d of dropLocal) {
          const table = this.table(d.entity)
          const row = (await table.get(d.id)) as { change_seq?: number } | undefined
          if (row && !row.change_seq) await table.delete(d.id)
        }
        for (const delta of revert) await applyBalanceDelta(this.db, delta, -1)
      })
      if (rejected.length) this.setStatus({ rejected: [...rejected, ...this.status.rejected].slice(0, 20) })
    }
  }

  table(entity: Entity): Table<object, string> {
    return this.db[entity] as unknown as Table<object, string>
  }

  private async pull() {
    let since = (await this.db.getMeta<number>('cursor')) ?? 0
    for (let i = 0; i < 100; i++) {
      const res = await this.transport<{ data: PullResult }>('GET', `${API}/sync/pull?since=${since}&limit=${PULL_LIMIT}`)
      const data = res.data
      await this.apply(data)
      since = data.seq
      if (!data.has_more) return
    }
  }

  private async apply(data: PullResult) {
    const db = this.db
    await db.transaction('rw', [db.accounts, db.categories, db.transactions, db.rules, db.budgets, db.recurrences, db.rawEvents, db.outbox, db.meta], async () => {
      if (data.reset) {
        await Promise.all([db.accounts.clear(), db.categories.clear(), db.transactions.clear(), db.rules.clear(), db.budgets.clear(), db.recurrences.clear(), db.rawEvents.clear()])
      }
      const pending = await db.outbox.toArray()
      const overlay = new Map<string, OutboxItem[]>()
      const pendingBalance = new Map<string, number>()
      for (const p of pending) {
        for (const [id, d] of Object.entries(p.balance ?? {})) pendingBalance.set(id, (pendingBalance.get(id) ?? 0) + d)
        if (!p.entity || !p.id) continue
        const key = `${p.entity}:${p.id}`
        overlay.set(key, [...(overlay.get(key) ?? []), p])
      }
      const entities: [Entity, object[]][] = [
        ['accounts', data.changes.accounts],
        ['categories', data.changes.categories],
        ['transactions', data.changes.transactions],
        ['rules', data.changes.rules],
        ['budgets', data.changes.budgets],
        ['recurrences', data.changes.recurrences ?? []],
      ]
      for (const [entity, rows] of entities) {
        const table = this.table(entity)
        for (const raw of rows ?? []) {
          const row = normalizeDates({ ...(raw as Record<string, unknown>) }) as Record<string, unknown>
          const id = row.id as string
          const pend = overlay.get(`${entity}:${id}`)
          if (pend?.some((p) => p.op === 'delete')) continue
          if (row.deleted_at) {
            await table.delete(id)
            continue
          }
          for (const p of pend ?? []) Object.assign(row, p.changes ?? {})
          if (entity === 'accounts' && pendingBalance.has(id) && typeof row.balance === 'number') row.balance += pendingBalance.get(id)!
          await table.put(row)
        }
      }
      for (const raw of data.changes.raw_events ?? []) {
        const row = normalizeDates({ ...raw })
        if (row.deleted_at) await db.rawEvents.delete(row.id)
        else await db.rawEvents.put(row)
      }
      await db.meta.put({ key: 'cursor', value: data.seq })
    })
  }

  private async enqueue(items: OutboxItem[], local: () => Promise<BalanceDelta | void>) {
    const db = this.db
    await db.transaction('rw', [db.outbox, db.accounts, db.categories, db.transactions, db.rules, db.budgets, db.recurrences, db.rawEvents], async () => {
      const delta = await local()
      if (delta && Object.values(delta).some(Boolean)) items[0].balance = delta
      await db.outbox.bulkAdd(items)
    })
    await this.refreshPending()
    this.schedule()
  }

  async create(entity: Entity, row: Record<string, unknown> & { id?: string }, defaults: Record<string, unknown> = {}, optimistic?: () => Promise<BalanceDelta | void>): Promise<string> {
    const id = row.id ?? uuidv7()
    const now = new Date().toISOString()
    const changes: Record<string, unknown> = { ...row }
    delete changes.id
    const local = { ...defaults, ...changes, id, version: 0, change_seq: 0, created_at: now, updated_at: now, deleted_at: null }
    await this.enqueue([{ mid: uuidv7(), entity, op: 'upsert', id, changes, ts: Date.now() }], async () => {
      await this.table(entity).put(local)
      if (optimistic) return optimistic()
    })
    return id
  }

  async patch(entity: Entity, id: string, changes: Record<string, unknown>, optimistic?: () => Promise<BalanceDelta | void>) {
    await this.enqueue([{ mid: uuidv7(), entity, op: 'patch', id, changes, ts: Date.now() }], async () => {
      await this.table(entity).update(id, changes)
      if (optimistic) return optimistic()
    })
  }

  async remove(entity: Entity, id: string, optimistic?: () => Promise<BalanceDelta | void>) {
    await this.enqueue([{ mid: uuidv7(), entity, op: 'delete', id, ts: Date.now() }], async () => {
      await this.table(entity).delete(id)
      if (optimistic) return optimistic()
    })
  }

  async action(action: string, args: Record<string, unknown>, optimistic?: () => Promise<BalanceDelta | void>, drop?: { entity: Entity; id: string }[]) {
    await this.enqueue([{ mid: uuidv7(), op: 'action', action, args, ts: Date.now(), ...(drop?.length ? { drop } : {}) }], async () => {
      if (optimistic) return optimistic()
    })
  }
}
