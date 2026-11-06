import { useEffect, useMemo, useState } from 'react'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Search, X } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { useAccountMap, useAccounts, useCategories, useCategoryMap, useMonthTransactions, useRangeTransactions, useReviewQueue, useThuChi } from '../hooks'
import { normalize } from '../classify'
import { dayKey, dayLabel, money } from '../format'
import { categoryLabel, expenseValue, flatTree, inStats, rootCategoryId } from '../stats'
import type { Transaction } from '../types'
import { Empty, TxRow } from './common'

const ALL = '__all__'
const FILTER_KEY = 'thuchi.txFilter'
const MAX_ROWS = 300

function readFilter() {
  try {
    return JSON.parse(sessionStorage.getItem(FILTER_KEY) ?? '{}') as { account?: string; category?: string; allTime?: boolean }
  } catch {
    return {}
  }
}

export function Transactions({ onOpenTx, initialView }: { onOpenTx: (id: string) => void; initialView?: string }) {
  const { month } = useThuChi()
  const monthTxs = useMonthTransactions(month)
  const review = useReviewQueue()
  const accounts = useAccounts()
  const accountMap = useAccountMap()
  const categories = useCategories()
  const catMap = useCategoryMap()
  const [view, setView] = useState<'month' | 'review'>(initialView === 'review' ? 'review' : 'month')
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')
  const [account, setAccount] = useState(readFilter().account ?? ALL)
  const [category, setCategory] = useState(readFilter().category ?? ALL)
  const [allTime, setAllTime] = useState(readFilter().allTime ?? false)
  const allTxs = useRangeTransactions('', '\uffff', allTime && view === 'month')

  useEffect(() => {
    const t = setTimeout(() => setDebounced(query), 300)
    return () => clearTimeout(t)
  }, [query])

  useEffect(() => {
    try {
      sessionStorage.setItem(FILTER_KEY, JSON.stringify({ account, category, allTime }))
    } catch {
      /* ignore */
    }
  }, [account, category, allTime])

  const loading = view === 'month' && (allTime ? allTxs : monthTxs) === undefined
  const source = useMemo(() => (view === 'review' ? review : ((allTime ? allTxs : monthTxs) ?? [])), [view, review, allTime, allTxs, monthTxs])
  const filtered = useMemo(() => {
    const q = normalize(debounced)
    return source.filter((t) => {
      if (account !== ALL && t.account_id !== account) return false
      if (category !== ALL) {
        if (category === 'none' ? t.category_id !== null : ![t.category_id, t.refund_of_category_id].some((id) => id && (id === category || rootCategoryId(id, catMap) === category))) return false
      }
      if (q) {
        const hay = normalize(`${t.merchant} ${t.raw_content} ${t.note} ${t.counterparty_name} ${t.amount}`)
        if (!hay.includes(q)) return false
      }
      return true
    })
  }, [source, account, category, debounced, catMap])

  const groups = useMemo(() => {
    const map = new Map<string, Transaction[]>()
    for (const t of filtered.slice(0, MAX_ROWS)) {
      const k = dayKey(new Date(t.occurred_at))
      map.set(k, [...(map.get(k) ?? []), t])
    }
    return [...map.entries()]
  }, [filtered])

  const totals = useMemo(() => {
    let out = 0
    let inc = 0
    for (const t of filtered) {
      if (!inStats(t, accountMap)) continue
      if (t.direction === 'out' || t.kind === 'refund') out += expenseValue(t)
      else inc += t.amount
    }
    return { out, inc }
  }, [filtered, accountMap])

  const reviewCount = review.length
  const filtersOn = account !== ALL || category !== ALL || !!query

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-1 rounded-lg bg-muted p-1 text-sm">
        <button type="button" onClick={() => setView('month')} className={view === 'month' ? 'rounded-md bg-background py-1.5 font-medium shadow-sm' : 'py-1.5 text-muted-foreground'}>
          {allTime ? 'Mọi thời gian' : 'Trong tháng'}
        </button>
        <button type="button" onClick={() => setView('review')} className={view === 'review' ? 'rounded-md bg-background py-1.5 font-medium shadow-sm' : 'py-1.5 text-muted-foreground'}>
          Cần xem {reviewCount > 0 && <span className="ml-1 rounded-full bg-amber-500 px-1.5 text-xs text-white">{reviewCount}</span>}
        </button>
      </div>

      <div className="flex gap-2">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Tìm nội dung, số tiền…" className="pl-8" />
        </div>
        {view === 'month' && (
          <Button variant={allTime ? 'default' : 'outline'} aria-pressed={allTime} onClick={() => setAllTime(!allTime)}>
            Mọi lúc
          </Button>
        )}
        {filtersOn && (
          <Button
            variant="ghost"
            size="icon"
            aria-label="Xoá bộ lọc"
            onClick={() => {
              setAccount(ALL)
              setCategory(ALL)
              setQuery('')
            }}
          >
            <X />
          </Button>
        )}
      </div>
      <div className="grid grid-cols-2 gap-2">
        <Select value={account} onValueChange={(v) => setAccount(String(v))} items={[{ value: ALL, label: 'Mọi tài khoản' }, ...accounts.map((a) => ({ value: a.id, label: a.name }))]}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>Mọi tài khoản</SelectItem>
            {accounts.map((a) => (
              <SelectItem key={a.id} value={a.id}>
                {a.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={category}
          onValueChange={(v) => setCategory(String(v))}
          items={[{ value: ALL, label: 'Mọi danh mục' }, { value: 'none', label: 'Chưa phân loại' }, ...categories.map((c) => ({ value: c.id, label: `${c.name}${c.kind === 'income' ? ' (thu)' : ''}` }))]}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>Mọi danh mục</SelectItem>
            <SelectItem value="none">Chưa phân loại</SelectItem>
            {flatTree(categories).map(({ category: c, child }) => (
              <SelectItem key={c.id} value={c.id}>
                {categoryLabel(c, child)}
                {c.kind === 'income' ? ' (thu)' : ''}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex justify-between rounded-lg border px-3 py-2 text-sm">
        <span className="text-muted-foreground">{filtered.length} giao dịch</span>
        <span className="tabular-nums">
          Chi <b>{money(totals.out)}</b> · Thu <b className="text-emerald-600 dark:text-emerald-400">{money(totals.inc)}</b>
        </span>
      </div>

      {loading ? (
        <Empty>Đang tải…</Empty>
      ) : groups.length === 0 ? (
        <Empty>{view === 'review' ? 'Không còn giao dịch nào cần xem 🎉' : 'Không có giao dịch'}</Empty>
      ) : (
        groups.map(([day, list]) => {
          const dayOut = list.reduce((s, t) => s + (inStats(t, accountMap) ? expenseValue(t) : 0), 0)
          return (
            <section key={day} className="overflow-hidden rounded-xl border bg-card">
              <div className="flex justify-between bg-muted/50 px-3 py-1.5 text-xs font-medium text-muted-foreground">
                <span>{dayLabel(day)}</span>
                {dayOut !== 0 && <span className="tabular-nums">−{money(dayOut)}</span>}
              </div>
              <div className="divide-y">
                {list.map((t) => (
                  <TxRow key={t.id} tx={t} category={t.category_id ? catMap.get(t.category_id) : t.refund_of_category_id ? catMap.get(t.refund_of_category_id) : null} account={t.account_id ? accountMap.get(t.account_id) : undefined} onClick={() => onOpenTx(t.id)} />
                ))}
              </div>
            </section>
          )
        })
      )}
      {filtered.length > MAX_ROWS && <p className="py-2 text-center text-xs text-muted-foreground">Đang hiện {MAX_ROWS} giao dịch mới nhất, gõ thêm từ khoá để thu hẹp.</p>}
    </div>
  )
}
