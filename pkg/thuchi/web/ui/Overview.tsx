import { useMemo } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { TrendingDown, TrendingUp, Inbox, AlertTriangle, ChevronRight } from '@/components/icons'
import { useAccounts, useAccountMap, useCategoryMap, useEffectiveBudgets, useMonthTransactions, useRawEvents, useReviewQueue, useThuChi } from '../hooks'
import { compact, money, monthLabel, shiftMonth } from '../format'
import { budgetSpent, rollup, summarize } from '../stats'
import { AccountIcon, CategoryIcon, Empty, Section, TxRow } from './common'

export function Overview({ onOpenTx, onGoto }: { onOpenTx: (id: string) => void; onGoto: (tab: 'transactions' | 'stats' | 'settings', sub?: string) => void }) {
  const { month, db } = useThuChi()
  const accounts = useAccounts()
  const accountMap = useAccountMap()
  const catMap = useCategoryMap()
  const txs = useMonthTransactions(month)
  const prevMonth = shiftMonth(month, -1)
  const prevTxs = useLiveQuery(async () => {
    const [y, m] = prevMonth.split('-').map(Number)
    const start = new Date(y, m - 1, 1).toISOString()
    const now = new Date()
    const sameDay = new Date(y, m - 1, Math.min(now.getDate() + 1, new Date(y, m, 0).getDate() + 1))
    const end = month === `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}` ? sameDay.toISOString() : new Date(y, m, 1).toISOString()
    return db.transactions.where('occurred_at').between(start, end, true, false).toArray()
  }, [db, prevMonth, month])
  const review = useReviewQueue()
  const unparsed = useRawEvents(['unparsed', 'error'])
  const budgets = useEffectiveBudgets(month)

  const sum = useMemo(() => summarize(txs ?? [], accountMap), [txs, accountMap])
  const prev = useMemo(() => summarize(prevTxs ?? [], accountMap), [prevTxs, accountMap])
  const diff = prev.expense > 0 ? Math.round(((sum.expense - prev.expense) / prev.expense) * 100) : null
  const topCats = [...rollup(sum.expenseByCategory, catMap).entries()].filter(([, v]) => v > 0).sort((a, b) => b[1] - a[1]).slice(0, 5)
  const totalBalance = accounts.filter((a) => a.include_in_stats).reduce((s, a) => s + a.balance, 0)
  const budgetAlerts = [...budgets.entries()]
    .filter(([key, b]) => b.limit_amount > 0 && (key === 'total' || catMap.has(key)))
    .map(([key, b]) => ({ b, spent: budgetSpent(key, sum, catMap) }))
    .filter(({ b, spent }) => spent >= b.limit_amount * 0.8)

  return (
    <div className="space-y-3">
      <section className="rounded-xl bg-primary p-4 text-primary-foreground">
        <p className="text-sm opacity-80">Chi {monthLabel(month).toLowerCase()}</p>
        <p className="mt-1 text-3xl font-bold tabular-nums">{money(sum.expense)}</p>
        <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm opacity-90">
          {diff !== null && (
            <span className="inline-flex items-center gap-1">
              {diff > 0 ? <TrendingUp className="size-4" /> : <TrendingDown className="size-4" />}
              {diff > 0 ? '+' : ''}
              {diff}% so với cùng kỳ tháng trước
            </span>
          )}
          <span>Thu {compact(sum.income)}</span>
        </div>
      </section>

      {(review.length > 0 || unparsed.length > 0) && (
        <Section
          title={
            <span className="inline-flex items-center gap-1.5">
              <Inbox className="size-4" /> Cần xem ({review.length})
            </span>
          }
          action={
            review.length > 4 ? (
              <button type="button" className="text-xs text-primary" onClick={() => onGoto('transactions', 'review')}>
                Xem tất cả
              </button>
            ) : undefined
          }
        >
          <div className="divide-y">
            {review.slice(0, 4).map((t) => (
              <TxRow key={t.id} tx={t} category={t.category_id ? catMap.get(t.category_id) : null} account={t.account_id ? accountMap.get(t.account_id) : undefined} onClick={() => onOpenTx(t.id)} />
            ))}
            {unparsed.length > 0 && (
              <button type="button" onClick={() => onGoto('settings', 'unparsed')} className="flex w-full items-center gap-3 px-3 py-3 text-left text-sm hover:bg-muted/60">
                <AlertTriangle className="size-5 text-amber-600" />
                <span className="flex-1">{unparsed.length} thông báo ngân hàng chưa đọc được</span>
                <ChevronRight className="size-4 text-muted-foreground" />
              </button>
            )}
          </div>
        </Section>
      )}

      {budgetAlerts.length > 0 && (
        <Section title="Hạn mức sắp vượt">
          <div className="space-y-2 px-3 pb-3">
            {budgetAlerts.map(({ b, spent }) => {
              const pct = Math.round((spent / b.limit_amount) * 100)
              const cat = b.category_id ? catMap.get(b.category_id) : null
              return (
                <div key={b.id}>
                  <div className="flex justify-between text-sm">
                    <span>{cat?.name ?? 'Tổng chi'}</span>
                    <span className={pct >= 100 ? 'text-destructive' : 'text-amber-600'}>
                      {compact(spent)} / {compact(b.limit_amount)} ({pct}%)
                    </span>
                  </div>
                  <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-muted">
                    <div className={pct >= 100 ? 'h-full bg-destructive' : 'h-full bg-amber-500'} style={{ width: `${Math.min(pct, 100)}%` }} />
                  </div>
                </div>
              )
            })}
          </div>
        </Section>
      )}

      <Section
        title="Tài khoản"
        action={<span className="text-sm font-semibold tabular-nums">{money(totalBalance)}</span>}
      >
        <div className="divide-y">
          {accounts.map((a) => (
            <div key={a.id} className="flex items-center gap-3 px-3 py-2.5">
              <AccountIcon account={a} />
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{a.name}</p>
                <p className="text-xs text-muted-foreground">
                  {a.connector === 'manual' ? 'Nhập tay' : a.connector === 'sepay' ? 'SePay' : a.connector === 'email' ? 'Email' : 'Thông báo'}
                  {!a.include_in_stats && ' · không tính thống kê'}
                </p>
              </div>
              <span className="tabular-nums font-medium">{money(a.balance)}</span>
            </div>
          ))}
          {accounts.length === 0 && <Empty>Chưa có tài khoản</Empty>}
        </div>
      </Section>

      <Section
        title="Chi nhiều nhất"
        action={
          <button type="button" className="text-xs text-primary" onClick={() => onGoto('stats')}>
            Thống kê
          </button>
        }
      >
        <div className="divide-y">
          {topCats.map(([id, v]) => {
            const c = id ? catMap.get(id) : null
            return (
              <div key={id ?? 'none'} className="flex items-center gap-3 px-3 py-2.5">
                <CategoryIcon category={c} />
                <span className="flex-1 truncate">{c?.name ?? 'Chưa phân loại'}</span>
                <span className="text-xs text-muted-foreground">{sum.expense > 0 ? Math.round((v / sum.expense) * 100) : 0}%</span>
                <span className="w-24 text-right tabular-nums font-medium">{money(v)}</span>
              </div>
            )
          })}
          {topCats.length === 0 && <Empty>Chưa có khoản chi nào trong tháng</Empty>}
        </div>
      </Section>
    </div>
  )
}
