import Dexie, { type Table } from 'dexie'
import type { Account, BalanceDelta, Budget, Category, OutboxItem, RawEvent, Recurrence, Rule, Transaction } from './types'

type MaybeTx = Transaction | null | undefined

export interface MetaRow {
  key: string
  value: unknown
}

export class ThuChiDB extends Dexie {
  accounts!: Table<Account, string>
  categories!: Table<Category, string>
  transactions!: Table<Transaction, string>
  rules!: Table<Rule, string>
  budgets!: Table<Budget, string>
  recurrences!: Table<Recurrence, string>
  rawEvents!: Table<RawEvent, string>
  outbox!: Table<OutboxItem, number>
  meta!: Table<MetaRow, string>

  constructor(name: string) {
    super(name)
    this.version(1).stores({
      accounts: 'id, sort',
      categories: 'id, kind, sort',
      transactions: 'id, occurred_at, account_id, category_id, pair_id',
      rules: 'id',
      budgets: 'id, month',
      rawEvents: 'id, status, received_at',
      outbox: '++seq, [entity+id]',
      meta: 'key',
    })
    this.version(2).stores({
      transactions: 'id, occurred_at, account_id, category_id, pair_id, kind',
      recurrences: 'id, next_at',
    })
  }

  async getMeta<T>(key: string): Promise<T | undefined> {
    const row = await this.meta.get(key)
    return row?.value as T | undefined
  }

  async setMeta(key: string, value: unknown) {
    await this.meta.put({ key, value })
  }
}

export function dbName(ownerId: string) {
  return `thuchi_${ownerId}`
}

export async function deleteLocalData(ownerId?: string) {
  if (ownerId) {
    await Dexie.delete(dbName(ownerId))
    return
  }
  const names = await Dexie.getDatabaseNames()
  await Promise.all(names.filter((n) => n.startsWith('thuchi_')).map((n) => Dexie.delete(n)))
}

export function isManual(t: Transaction) {
  return t.sources.every((s) => s === 'manual')
}

function balanceEffect(t: MaybeTx): [string, number] | null {
  if (!t || t.deleted_at || !t.account_id || !isManual(t)) return null
  return [t.account_id, t.direction === 'in' ? t.amount : -t.amount]
}

export async function applyBalanceDelta(db: ThuChiDB, delta: BalanceDelta, sign = 1): Promise<BalanceDelta> {
  const applied: BalanceDelta = {}
  for (const [id, d] of Object.entries(delta)) {
    if (!d) continue
    const acc = await db.accounts.get(id)
    if (!acc || acc.connector !== 'manual') continue
    await db.accounts.update(id, { balance: acc.balance + sign * d })
    applied[id] = d
  }
  return applied
}

export async function shiftBalances(db: ThuChiDB, changes: [MaybeTx, MaybeTx][]): Promise<BalanceDelta> {
  const delta: BalanceDelta = {}
  for (const [before, after] of changes) {
    const b = balanceEffect(before)
    const a = balanceEffect(after)
    if (b) delta[b[0]] = (delta[b[0]] ?? 0) - b[1]
    if (a) delta[a[0]] = (delta[a[0]] ?? 0) + a[1]
  }
  return applyBalanceDelta(db, delta)
}

type LocalTxFields = Pick<Transaction, 'id' | 'account_id' | 'direction' | 'amount' | 'occurred_at'> & Partial<Transaction>

export function localTx(fields: LocalTxFields): Transaction {
  const now = new Date().toISOString()
  return {
    raw_content: '',
    merchant: '',
    counterparty_name: '',
    counterparty_account: '',
    balance_after: null,
    category_id: null,
    note: '',
    lat: null,
    lng: null,
    place_name: '',
    kind: 'normal',
    pair_id: null,
    pair_status: '',
    refund_of_category_id: null,
    excluded: false,
    sources: ['manual'],
    reconciled: false,
    needs_review: false,
    party: '',
    recurrence_id: null,
    version: 0,
    change_seq: 0,
    created_at: now,
    updated_at: now,
    deleted_at: null,
    ...fields,
  }
}
