export type ID = string

export interface Meta {
  version: number
  change_seq: number
  created_at: string
  updated_at: string
  deleted_at: string | null
}

export type AccountKind = 'bank' | 'ewallet' | 'credit' | 'cash'

export interface Account extends Meta {
  id: ID
  name: string
  kind: AccountKind
  bank_code: string
  account_no_last4: string
  holder_name: string
  connector: string
  balance: number
  balance_at: string | null
  include_in_stats: boolean
  sort: number
}

export type CategoryKind = 'expense' | 'income'

export interface Category extends Meta {
  id: ID
  key: string
  name: string
  icon: string
  color: string
  kind: CategoryKind
  parent_id: ID | null
  sort: number
}

export type Direction = 'in' | 'out'
export type TxKind = 'normal' | 'transfer' | 'refund' | 'card_payment' | 'adjustment' | 'borrow' | 'lend' | 'repay' | 'collect'
export type DebtKind = 'borrow' | 'lend' | 'repay' | 'collect'

export const DEBT_KINDS: DebtKind[] = ['borrow', 'lend', 'repay', 'collect']

export function isDebtKind(kind: TxKind): kind is DebtKind {
  return (DEBT_KINDS as TxKind[]).includes(kind)
}
export type PairStatus = '' | 'auto' | 'suggested' | 'confirmed' | 'rejected'

export interface Transaction extends Meta {
  id: ID
  account_id: ID | null
  direction: Direction
  amount: number
  occurred_at: string
  raw_content: string
  merchant: string
  counterparty_name: string
  counterparty_account: string
  balance_after: number | null
  category_id: ID | null
  note: string
  lat: number | null
  lng: number | null
  place_name: string
  kind: TxKind
  pair_id: ID | null
  pair_status: PairStatus
  refund_of_category_id: ID | null
  excluded: boolean
  sources: string[]
  reconciled: boolean
  needs_review: boolean
  party: string
  recurrence_id: ID | null
}

export type RuleMatch = 'contains' | 'regex' | 'merchant' | 'own_account'

export interface Rule extends Meta {
  id: ID
  pattern: string
  match: RuleMatch
  category_id: ID | null
  merchant_name: string
  origin: 'manual' | 'learned'
  hits: number
}

export interface Budget extends Meta {
  id: ID
  category_id: ID | null
  month: string
  limit_amount: number
}

export type Freq = 'daily' | 'weekly' | 'monthly' | 'yearly'

export interface Recurrence extends Meta {
  id: ID
  account_id: ID | null
  direction: Direction
  amount: number
  category_id: ID | null
  merchant: string
  note: string
  freq: Freq
  interval: number
  start_at: string
  end_at: string | null
  next_at: string
  active: boolean
}

export interface RawEvent {
  id: ID
  source: 'sepay' | 'notify' | 'email'
  source_key: string
  payload?: string
  received_at: string
  status: 'pending' | 'parsed' | 'duplicate' | 'unparsed' | 'ignored' | 'error' | 'dismissed'
  error: string
  transaction_id: ID | null
  version: number
  change_seq: number
  updated_at: string
  deleted_at: string | null
}

export type Entity = 'accounts' | 'categories' | 'transactions' | 'rules' | 'budgets' | 'recurrences'

export interface OutboxItem {
  seq?: number
  mid: string
  entity?: Entity
  op: 'upsert' | 'patch' | 'delete' | 'action'
  id?: ID
  changes?: Record<string, unknown>
  action?: string
  args?: Record<string, unknown>
  ts: number
  drop?: { entity: Entity; id: ID }[]
  balance?: BalanceDelta
}

export type BalanceDelta = Record<ID, number>

export interface Changes {
  accounts: Account[]
  categories: Category[]
  transactions: Transaction[]
  rules: Rule[]
  budgets: Budget[]
  recurrences?: Recurrence[]
  raw_events: RawEvent[]
}

export interface PullResult {
  reset?: boolean
  seq: number
  has_more: boolean
  changes: Changes
}

export interface PushResult {
  results: { mid: string; ok: boolean; error?: string }[]
  seq: number
}

export interface IngestToken {
  id: ID
  kind: 'notify' | 'email' | 'sepay'
  name: string
  config?: Record<string, string>
  last_used_at: string | null
  created_at: string
}

export interface IngestInfo {
  owner_id: ID
  urls: Record<'sepay' | 'notify' | 'email', string>
  tokens: IngestToken[]
  sepay: IngestToken | null
  packages: Record<string, string>
  parsers: string[]
}

export interface TelegramLink {
  id: ID
  telegram_user_id: number
  telegram_username: string
  telegram_name: string
  default_account_id: ID | null
  linked_at: string | null
}

export interface TelegramInfo {
  enabled: boolean
  bot_username: string
  mode: 'webhook' | 'polling'
  reads_group_messages: boolean
  links: TelegramLink[]
  examples?: { text: string; desc: string }[]
}

export interface TelegramCode {
  code: string
  command: string
  url: string
  expires_at: string
}

export type Transport = <T>(method: 'GET' | 'POST' | 'PUT' | 'DELETE', url: string, data?: unknown) => Promise<T>
