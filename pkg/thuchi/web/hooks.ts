import { useContext, useMemo, useSyncExternalStore } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { Ctx } from './ctx'
import { monthRange } from './format'
import { effectiveBudgets } from './stats'
import { DEBT_KINDS, type Account, type Budget, type Category, type RawEvent, type Recurrence, type Rule, type Transaction } from './types'

export function useThuChi() {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useThuChi must be used inside ThuChiProvider')
  return ctx
}

export function useSyncStatus() {
  const { engine } = useThuChi()
  return useSyncExternalStore(engine.subscribe, engine.getStatus)
}

const EMPTY: never[] = []

export function useAccounts(): Account[] {
  const { db } = useThuChi()
  return useLiveQuery(() => db.accounts.orderBy('sort').toArray(), [db]) ?? EMPTY
}

export function useAccountMap() {
  const accounts = useAccounts()
  return useMemo(() => new Map(accounts.map((a) => [a.id, a])), [accounts])
}

export function useCategories(): Category[] {
  const { db } = useThuChi()
  return useLiveQuery(() => db.categories.orderBy('sort').toArray(), [db]) ?? EMPTY
}

export function useCategoryMap() {
  const cats = useCategories()
  return useMemo(() => new Map(cats.map((c) => [c.id, c])), [cats])
}

export function useRules(): Rule[] {
  const { db } = useThuChi()
  return useLiveQuery(() => db.rules.toArray(), [db]) ?? EMPTY
}

export function useAllBudgets(): Budget[] {
  const { db } = useThuChi()
  return useLiveQuery(() => db.budgets.toArray(), [db]) ?? EMPTY
}

export function useEffectiveBudgets(month: string) {
  const budgets = useAllBudgets()
  return useMemo(() => effectiveBudgets(budgets, month), [budgets, month])
}

export function useRangeTransactions(startISO: string, endISO: string, enabled = true): Transaction[] | undefined {
  const { db } = useThuChi()
  return useLiveQuery(
    () => (enabled ? db.transactions.where('occurred_at').between(startISO, endISO, true, false).reverse().sortBy('occurred_at') : EMPTY),
    [db, startISO, endISO, enabled],
  )
}

export function useMonthTransactions(month: string): Transaction[] | undefined {
  const { db } = useThuChi()
  return useLiveQuery(() => {
    const { startISO, endISO } = monthRange(month)
    return db.transactions.where('occurred_at').between(startISO, endISO, true, false).reverse().sortBy('occurred_at')
  }, [db, month])
}

export function useReviewQueue(): Transaction[] {
  const { db } = useThuChi()
  return (
    useLiveQuery(
      () =>
        db.transactions
          .filter((t) => t.needs_review || t.pair_status === 'suggested')
          .reverse()
          .sortBy('occurred_at'),
      [db],
    ) ?? EMPTY
  )
}

export function useRawEvents(statuses: RawEvent['status'][]): RawEvent[] {
  const { db } = useThuChi()
  const key = statuses.join(',')
  return (
    useLiveQuery(
      () => db.rawEvents.where('status').anyOf(key.split(',')).reverse().sortBy('received_at'),
      [db, key],
    ) ?? EMPTY
  )
}

export function useDebtTransactions(): Transaction[] | undefined {
  const { db } = useThuChi()
  return useLiveQuery(() => db.transactions.where('kind').anyOf(DEBT_KINDS).reverse().sortBy('occurred_at'), [db])
}

export function useParties(): string[] {
  const txs = useDebtTransactions()
  return useMemo(() => {
    const seen = new Map<string, string>()
    for (const t of txs ?? []) {
      const name = (t.party ?? '').trim()
      if (name && !seen.has(name.toLowerCase())) seen.set(name.toLowerCase(), name)
    }
    return [...seen.values()]
  }, [txs])
}

export function useRecurrences(): Recurrence[] {
  const { db } = useThuChi()
  return useLiveQuery(() => db.recurrences.orderBy('next_at').toArray(), [db]) ?? EMPTY
}
