import { dayKey } from './format'
import type { Account, Budget, Category, Transaction } from './types'

export interface Summary {
  expense: number
  income: number
  expenseByCategory: Map<string | null, number>
  incomeByCategory: Map<string | null, number>
  expenseByDay: Map<string, number>
  countByDay: Map<string, number>
}

const NON_STATS_KINDS = new Set<Transaction['kind']>(['transfer', 'card_payment', 'adjustment', 'borrow', 'lend', 'repay', 'collect'])

export function inStats(t: Transaction, accounts: Map<string, Account>) {
  if (t.excluded || NON_STATS_KINDS.has(t.kind)) return false
  if (t.account_id) {
    const a = accounts.get(t.account_id)
    if (a && !a.include_in_stats) return false
  }
  return true
}

export function expenseValue(t: Transaction) {
  if (t.kind === 'refund') return -t.amount
  return t.direction === 'out' ? t.amount : 0
}

export function expenseCategory(t: Transaction) {
  return t.kind === 'refund' ? (t.refund_of_category_id ?? t.category_id) : t.category_id
}

export function summarize(txs: Transaction[], accounts: Map<string, Account>): Summary {
  const s: Summary = {
    expense: 0,
    income: 0,
    expenseByCategory: new Map(),
    incomeByCategory: new Map(),
    expenseByDay: new Map(),
    countByDay: new Map(),
  }
  for (const t of txs) {
    if (!inStats(t, accounts)) continue
    const day = dayKey(new Date(t.occurred_at))
    if (t.direction === 'out' || t.kind === 'refund') {
      const v = expenseValue(t)
      const cat = expenseCategory(t)
      s.expense += v
      s.expenseByCategory.set(cat, (s.expenseByCategory.get(cat) ?? 0) + v)
      s.expenseByDay.set(day, (s.expenseByDay.get(day) ?? 0) + v)
      s.countByDay.set(day, (s.countByDay.get(day) ?? 0) + 1)
    } else {
      s.income += t.amount
      s.incomeByCategory.set(t.category_id, (s.incomeByCategory.get(t.category_id) ?? 0) + t.amount)
    }
  }
  return s
}

export function needsReview(t: Transaction) {
  return t.needs_review || t.pair_status === 'suggested' || (t.direction === 'out' && t.kind === 'normal' && !t.category_id)
}

export function rootCategoryId(id: string | null, cats: Map<string, Category>) {
  if (!id) return null
  const parent = cats.get(id)?.parent_id
  return parent && cats.has(parent) ? parent : id
}

export function rollup(byCategory: Map<string | null, number>, cats: Map<string, Category>) {
  const out = new Map<string | null, number>()
  for (const [id, v] of byCategory) {
    const root = rootCategoryId(id, cats)
    out.set(root, (out.get(root) ?? 0) + v)
  }
  return out
}

export interface CategoryNode {
  category: Category
  children: Category[]
}

export function categoryTree(categories: Category[]): CategoryNode[] {
  const ids = new Set(categories.map((c) => c.id))
  const kids = new Map<string, Category[]>()
  for (const c of categories) {
    if (c.parent_id && ids.has(c.parent_id)) kids.set(c.parent_id, [...(kids.get(c.parent_id) ?? []), c])
  }
  return categories.filter((c) => !c.parent_id || !ids.has(c.parent_id)).map((c) => ({ category: c, children: kids.get(c.id) ?? [] }))
}

export function flatTree(categories: Category[]) {
  return categoryTree(categories).flatMap((n) => [{ category: n.category, child: false }, ...n.children.map((c) => ({ category: c, child: true }))])
}

export function categoryLabel(c: Category, child: boolean) {
  return child ? `↳ ${c.name}` : c.name
}

export function budgetKey(b: Budget) {
  return b.category_id ?? 'total'
}

export function effectiveBudgets(budgets: Budget[], month: string, strictlyBefore = false) {
  const out = new Map<string, Budget>()
  for (const b of budgets) {
    if (strictlyBefore ? b.month >= month : b.month > month) continue
    const k = budgetKey(b)
    const cur = out.get(k)
    if (!cur || b.month > cur.month || (b.month === cur.month && b.updated_at > cur.updated_at)) out.set(k, b)
  }
  return out
}

export function budgetSpent(key: string, sum: Summary, cats: Map<string, Category>) {
  if (key === 'total') return sum.expense
  let v = 0
  for (const [id, x] of sum.expenseByCategory) {
    if (id && (id === key || rootCategoryId(id, cats) === key)) v += x
  }
  return v
}
