import { useMemo, useState } from 'react'
import { Bar, BarChart, CartesianGrid, Cell, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ChevronDown, ChevronRight, MapPin, Users } from '@/components/icons'
import { cn } from '@/lib/utils'
import { useAccountMap, useCategories, useCategoryMap, useDebtTransactions, useMonthTransactions, useRangeTransactions, useThuChi } from '../hooks'
import { compact, dayKey, dayLabel, money, monthKey, monthLabel, monthRange, shiftMonth } from '../format'
import { categoryLabel, expenseCategory, expenseValue, flatTree, inStats, rootCategoryId, summarize } from '../stats'
import type { Transaction } from '../types'
import { CategoryIcon, Empty, Section, TxRow } from './common'

const VIEWS = [
  ['category', 'Danh mục'],
  ['trend', 'Xu hướng'],
  ['calendar', 'Lịch'],
  ['map', 'Bản đồ'],
  ['debt', 'Vay nợ'],
] as const
type View = (typeof VIEWS)[number][0]
const VIEW_KEY = 'thuchi.statsView'
const PERIOD_KEY = 'thuchi.statsPeriod'
const TREND_KEY = 'thuchi.trendPeriod'
const ALL = '__all__'

function readStored<T>(key: string, parse: (raw: string) => T | undefined, fallback: T): T {
  try {
    const raw = localStorage.getItem(key)
    return (raw !== null && parse(raw)) || fallback
  } catch {
    return fallback
  }
}

function store(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* ignore */
  }
}

export function Stats({ onOpenTx }: { onOpenTx: (id: string) => void }) {
  const [view, setViewState] = useState<View>(() => readStored<View>(VIEW_KEY, (raw) => VIEWS.find(([k]) => k === raw)?.[0], 'category'))
  const setView = (v: View) => {
    setViewState(v)
    store(VIEW_KEY, v)
  }
  return (
    <div className="space-y-3">
      <div className="grid gap-1 rounded-lg bg-muted p-1 text-xs sm:text-sm" style={{ gridTemplateColumns: `repeat(${VIEWS.length}, minmax(0, 1fr))` }}>
        {VIEWS.map(([k, label]) => (
          <button key={k} type="button" onClick={() => setView(k)} className={cn('truncate whitespace-nowrap py-1.5', view === k ? 'rounded-md bg-background font-medium shadow-sm' : 'text-muted-foreground')}>
            {label}
          </button>
        ))}
      </div>
      {view === 'category' && <CategoryStats />}
      {view === 'trend' && <TrendStats />}
      {view === 'calendar' && <CalendarStats onOpenTx={onOpenTx} />}
      {view === 'map' && <MapStats />}
      {view === 'debt' && <DebtStats onOpenTx={onOpenTx} />}
    </div>
  )
}

type PeriodKind = 'month' | '3m' | 'year' | 'custom'
interface Period {
  kind: PeriodKind
  from: string
  to: string
}
const PERIODS: { value: PeriodKind; label: string }[] = [
  { value: 'month', label: 'Theo tháng' },
  { value: '3m', label: '3 tháng gần nhất' },
  { value: 'year', label: 'Năm nay' },
  { value: 'custom', label: 'Tuỳ chọn' },
]

function defaultPeriod(): Period {
  const now = new Date()
  return { kind: 'month', from: dayKey(new Date(now.getFullYear(), now.getMonth(), 1)), to: dayKey(now) }
}

function parsePeriod(raw: string): Period | undefined {
  const p = JSON.parse(raw) as Partial<Period>
  if (!PERIODS.some((o) => o.value === p.kind)) return undefined
  return { ...defaultPeriod(), ...p } as Period
}

function parseDay(key: string) {
  const [y, m, d] = key.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  return Number.isNaN(date.getTime()) || !y ? null : date
}

function shortMonth(month: string) {
  const [y, m] = month.split('-').map(Number)
  return `T${m}/${y}`
}

function dateText(d: Date) {
  return `${d.getDate()}/${d.getMonth() + 1}/${d.getFullYear()}`
}

function periodRange(p: Period, month: string) {
  const y = Number(month.slice(0, 4))
  if (p.kind === '3m') {
    const first = shiftMonth(month, -2)
    return { start: monthRange(first).start, end: monthRange(month).end, label: `${shortMonth(first)} – ${shortMonth(month)}` }
  }
  if (p.kind === 'year') return { start: new Date(y, 0, 1), end: new Date(y + 1, 0, 1), label: `Năm ${y}` }
  if (p.kind === 'custom') {
    let a = parseDay(p.from)
    let b = parseDay(p.to)
    if (a && b) {
      if (a > b) [a, b] = [b, a]
      return { start: a, end: new Date(b.getFullYear(), b.getMonth(), b.getDate() + 1), label: `${dateText(a)} – ${dateText(b)}` }
    }
  }
  const r = monthRange(month)
  return { start: r.start, end: r.end, label: monthLabel(month) }
}

function CategoryStats() {
  const { month } = useThuChi()
  const [period, setPeriodState] = useState<Period>(() => readStored(PERIOD_KEY, parsePeriod, defaultPeriod()))
  const setPeriod = (p: Period) => {
    setPeriodState(p)
    store(PERIOD_KEY, JSON.stringify(p))
  }
  const range = periodRange(period, month)
  const txs = useRangeTransactions(range.start.toISOString(), range.end.toISOString())
  const accountMap = useAccountMap()
  const catMap = useCategoryMap()
  const [side, setSide] = useState<'expense' | 'income'>('expense')
  const [open, setOpen] = useState<string | null>(null)
  const sum = useMemo(() => summarize(txs ?? [], accountMap), [txs, accountMap])
  const total = side === 'expense' ? sum.expense : sum.income
  const rows = useMemo(() => {
    const groups = new Map<string | null, { value: number; parts: Map<string | null, number> }>()
    for (const [id, v] of side === 'expense' ? sum.expenseByCategory : sum.incomeByCategory) {
      const root = rootCategoryId(id, catMap)
      const g = groups.get(root) ?? { value: 0, parts: new Map() }
      g.value += v
      g.parts.set(id, (g.parts.get(id) ?? 0) + v)
      groups.set(root, g)
    }
    return [...groups.entries()]
      .filter(([, g]) => g.value > 0)
      .sort((a, b) => b[1].value - a[1].value)
      .map(([id, g]) => {
        const c = id ? catMap.get(id) : undefined
        const parts = [...g.parts.entries()]
          .filter(([, v]) => v !== 0)
          .sort((a, b) => b[1] - a[1])
          .map(([cid, value]) => {
            const cc = cid ? catMap.get(cid) : undefined
            return { id: cid ?? 'none', name: cid === id ? `${c?.name ?? ''} (chung)` : (cc?.name ?? 'Chưa phân loại'), value, category: cc }
          })
        const hasSub = [...g.parts.keys()].some((cid) => cid !== id)
        return { id: id ?? 'none', name: c?.name ?? 'Chưa phân loại', color: c?.color ?? '#94a3b8', value: g.value, category: c, parts: hasSub ? parts : [] }
      })
  }, [sum, side, catMap])

  return (
    <>
      <div className="flex gap-2">
        <div className="grid flex-1 grid-cols-2 gap-1 rounded-lg border p-1 text-sm">
          {(['expense', 'income'] as const).map((s) => (
            <button key={s} type="button" onClick={() => setSide(s)} className={side === s ? 'rounded-md bg-muted py-1 font-medium' : 'py-1 text-muted-foreground'}>
              {s === 'expense' ? 'Chi' : 'Thu'}
            </button>
          ))}
        </div>
        <Select value={period.kind} onValueChange={(v) => setPeriod({ ...period, kind: v as PeriodKind })} items={PERIODS}>
          <SelectTrigger className="w-40 shrink-0">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {PERIODS.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {period.kind === 'custom' && (
        <div className="grid grid-cols-2 gap-2">
          <Input type="date" aria-label="Từ ngày" value={period.from} max={period.to || undefined} onChange={(e) => setPeriod({ ...period, from: e.target.value })} />
          <Input type="date" aria-label="Đến ngày" value={period.to} min={period.from || undefined} onChange={(e) => setPeriod({ ...period, to: e.target.value })} />
        </div>
      )}
      <Section title={range.label}>
        <div className="relative h-52">
          {rows.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={rows} dataKey="value" nameKey="name" innerRadius="62%" outerRadius="90%" paddingAngle={1.5} stroke="none" isAnimationActive={false}>
                  {rows.map((r) => (
                    <Cell key={r.id} fill={r.color} />
                  ))}
                </Pie>
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <Empty>{txs === undefined ? 'Đang tải…' : 'Chưa có dữ liệu'}</Empty>
          )}
          {rows.length > 0 && (
            <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
              <span className="text-xs text-muted-foreground">Tổng {side === 'expense' ? 'chi' : 'thu'}</span>
              <span className="text-xl font-bold tabular-nums">{compact(total)}</span>
            </div>
          )}
        </div>
        <div className="divide-y border-t">
          {rows.map((r) => {
            const expanded = open === r.id && r.parts.length > 0
            return (
              <div key={r.id}>
                <button type="button" aria-expanded={r.parts.length > 0 ? expanded : undefined} onClick={() => r.parts.length > 0 && setOpen(expanded ? null : r.id)} className={cn('flex w-full items-center gap-2 px-3 py-2.5 text-left', r.parts.length > 0 ? 'hover:bg-muted/60' : 'cursor-default')}>
                  <CategoryIcon category={r.category} />
                  <span className="min-w-0 flex-1 truncate">{r.name}</span>
                  <span className="w-10 text-right text-xs text-muted-foreground">{total > 0 ? Math.round((r.value / total) * 100) : 0}%</span>
                  <span className="w-28 text-right tabular-nums font-medium">{money(r.value)}</span>
                  {r.parts.length > 0 ? expanded ? <ChevronDown className="size-4 shrink-0 text-muted-foreground" /> : <ChevronRight className="size-4 shrink-0 text-muted-foreground" /> : <span className="size-4 shrink-0" />}
                </button>
                {expanded && (
                  <div className="divide-y bg-muted/30">
                    {r.parts.map((c) => (
                      <div key={c.id} className="flex items-center gap-2 py-2 pr-9 pl-12 text-sm">
                        <span className="min-w-0 flex-1 truncate">{c.name}</span>
                        <span className="w-10 text-right text-xs text-muted-foreground">{r.value > 0 ? Math.round((c.value / r.value) * 100) : 0}%</span>
                        <span className="w-24 text-right tabular-nums">{money(c.value)}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </Section>
    </>
  )
}

type TrendPeriod = '6' | '12' | 'year'
const TREND_PERIODS: [TrendPeriod, string][] = [
  ['6', '6 tháng'],
  ['12', '12 tháng'],
  ['year', 'Năm nay'],
]
const EXPENSE_COLOR = '#ef4444'
const INCOME_COLOR = '#10b981'

function TrendStats() {
  const { month } = useThuChi()
  const [period, setPeriodState] = useState<TrendPeriod>(() => readStored<TrendPeriod>(TREND_KEY, (raw) => TREND_PERIODS.find(([k]) => k === raw)?.[0], '6'))
  const setPeriod = (p: TrendPeriod) => {
    setPeriodState(p)
    store(TREND_KEY, p)
  }
  const months = useMemo(() => {
    const n = period === 'year' ? Number(month.slice(5)) : Number(period)
    return Array.from({ length: n }, (_, i) => shiftMonth(month, i - n + 1))
  }, [period, month])
  const txs = useRangeTransactions(monthRange(months[0]).startISO, monthRange(month).endISO)
  const accountMap = useAccountMap()
  const data = useMemo(() => {
    const byMonth = new Map(months.map((m) => [m, { month: m, label: `T${Number(m.slice(5))}`, expense: 0, income: 0 }]))
    for (const t of txs ?? []) {
      if (!inStats(t, accountMap)) continue
      const row = byMonth.get(monthKey(new Date(t.occurred_at)))
      if (!row) continue
      if (t.direction === 'out' || t.kind === 'refund') row.expense += expenseValue(t)
      else row.income += t.amount
    }
    return [...byMonth.values()]
  }, [months, txs, accountMap])
  const totalExpense = data.reduce((s, d) => s + d.expense, 0)
  const totalIncome = data.reduce((s, d) => s + d.income, 0)
  const n = Math.max(1, data.length)

  return (
    <>
      <div className="grid grid-cols-3 gap-1 rounded-lg border p-1 text-sm">
        {TREND_PERIODS.map(([k, label]) => (
          <button key={k} type="button" onClick={() => setPeriod(k)} className={period === k ? 'rounded-md bg-muted py-1 font-medium' : 'py-1 text-muted-foreground'}>
            {label}
          </button>
        ))}
      </div>
      <Section title={`${shortMonth(months[0])} – ${shortMonth(month)}`}>
        <div className="h-56 px-1 pb-2">
          {txs === undefined ? (
            <Empty>Đang tải…</Empty>
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barGap={1} barCategoryGap="20%">
                <CartesianGrid vertical={false} strokeDasharray="3 3" stroke="var(--border)" />
                <XAxis dataKey="label" tickLine={false} axisLine={false} interval={0} tick={{ fontSize: 10, fill: 'var(--muted-foreground)' }} />
                <YAxis tickFormatter={(v: number) => compact(v)} tickLine={false} axisLine={false} width={40} tick={{ fontSize: 10, fill: 'var(--muted-foreground)' }} />
                <Tooltip
                  cursor={{ fill: 'var(--muted)', opacity: 0.6 }}
                  contentStyle={{ background: 'var(--popover)', border: '1px solid var(--border)', borderRadius: 8, fontSize: 12 }}
                  labelFormatter={(_, payload) => {
                    const m = (payload?.[0]?.payload as { month?: string } | undefined)?.month
                    return m ? monthLabel(m) : ''
                  }}
                  formatter={(v) => money(Number(v))}
                />
                <Bar dataKey="expense" name="Chi" fill={EXPENSE_COLOR} radius={[3, 3, 0, 0]} isAnimationActive={false} />
                <Bar dataKey="income" name="Thu" fill={INCOME_COLOR} radius={[3, 3, 0, 0]} isAnimationActive={false} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>
      </Section>
      <div className="grid grid-cols-2 gap-2">
        <div className="rounded-xl border bg-card p-3">
          <p className="text-xs text-muted-foreground">Tổng chi</p>
          <p className="text-lg font-bold tabular-nums" style={{ color: EXPENSE_COLOR }}>{money(totalExpense)}</p>
          <p className="text-xs text-muted-foreground tabular-nums">TB {compact(Math.round(totalExpense / n))}/tháng</p>
        </div>
        <div className="rounded-xl border bg-card p-3">
          <p className="text-xs text-muted-foreground">Tổng thu</p>
          <p className="text-lg font-bold tabular-nums" style={{ color: INCOME_COLOR }}>{money(totalIncome)}</p>
          <p className="text-xs text-muted-foreground tabular-nums">TB {compact(Math.round(totalIncome / n))}/tháng</p>
        </div>
      </div>
      <Section>
        <div className="flex gap-2 border-b px-3 py-2 text-xs text-muted-foreground">
          <span className="flex-1">Tháng</span>
          <span className="w-24 text-right">Chi</span>
          <span className="w-24 text-right">Thu</span>
        </div>
        <div className="divide-y">
          {[...data].reverse().map((d) => (
            <div key={d.month} className="flex gap-2 px-3 py-2 text-sm tabular-nums">
              <span className="flex-1 truncate">{shortMonth(d.month)}</span>
              <span className="w-24 text-right">{compact(d.expense)}</span>
              <span className="w-24 text-right text-emerald-600 dark:text-emerald-400">{compact(d.income)}</span>
            </div>
          ))}
          <div className="flex gap-2 bg-muted/40 px-3 py-2 text-sm font-semibold tabular-nums">
            <span className="flex-1">Tổng · chênh {compact(totalIncome - totalExpense)}</span>
            <span className="w-24 text-right">{compact(totalExpense)}</span>
            <span className="w-24 text-right text-emerald-600 dark:text-emerald-400">{compact(totalIncome)}</span>
          </div>
        </div>
      </Section>
    </>
  )
}

function CalendarStats({ onOpenTx }: { onOpenTx: (id: string) => void }) {
  const { month } = useThuChi()
  const txs = useMonthTransactions(month)
  const accountMap = useAccountMap()
  const catMap = useCategoryMap()
  const categories = useCategories()
  const expenseCats = useMemo(() => flatTree(categories.filter((c) => c.kind === 'expense')), [categories])
  const [category, setCategory] = useState(ALL)
  const [selected, setSelected] = useState<string | null>(null)

  const filtered = useMemo(
    () =>
      (txs ?? []).filter((t) => {
        if (!inStats(t, accountMap)) return false
        if (category === ALL) return true
        const id = expenseCategory(t)
        return id === category || rootCategoryId(id, catMap) === category
      }),
    [txs, accountMap, category, catMap],
  )
  const byDay = useMemo(() => {
    const m = new Map<string, number>()
    for (const t of filtered) {
      const v = expenseValue(t)
      if (!v) continue
      const k = dayKey(new Date(t.occurred_at))
      m.set(k, (m.get(k) ?? 0) + v)
    }
    return m
  }, [filtered])
  const max = Math.max(1, ...byDay.values())
  const { start, end } = monthRange(month)
  const days = Math.round((end.getTime() - start.getTime()) / 86400000)
  const lead = (start.getDay() + 6) % 7
  const cells: (Date | null)[] = [...Array(lead).fill(null), ...Array.from({ length: days }, (_, i) => new Date(start.getFullYear(), start.getMonth(), i + 1))]
  const today = dayKey(new Date())
  const selectedTxs = selected ? filtered.filter((t) => dayKey(new Date(t.occurred_at)) === selected) : []

  return (
    <>
      <Select value={category} onValueChange={(v) => setCategory(String(v))} items={[{ value: ALL, label: 'Tất cả danh mục' }, ...expenseCats.map(({ category: c }) => ({ value: c.id, label: c.name }))]}>
        <SelectTrigger className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>Tất cả danh mục</SelectItem>
          {expenseCats.map(({ category: c, child }) => (
            <SelectItem key={c.id} value={c.id}>
              {categoryLabel(c, child)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Section>
        <div className="grid grid-cols-7 gap-1 p-2 text-center text-[11px] text-muted-foreground">
          {['T2', 'T3', 'T4', 'T5', 'T6', 'T7', 'CN'].map((d) => (
            <span key={d}>{d}</span>
          ))}
          {cells.map((d, i) => {
            if (!d) return <span key={`e${i}`} />
            const k = dayKey(d)
            const v = byDay.get(k) ?? 0
            const alpha = v > 0 ? 0.15 + 0.75 * (v / max) : 0
            return (
              <button
                key={k}
                type="button"
                onClick={() => setSelected(selected === k ? null : k)}
                className={cn('flex aspect-square min-h-11 flex-col items-center justify-center rounded-md border text-foreground', selected === k ? 'border-primary ring-2 ring-primary/30' : 'border-transparent', k === today && 'font-bold')}
                style={{ backgroundColor: v > 0 ? `color-mix(in oklch, var(--primary) ${Math.round(alpha * 100)}%, transparent)` : undefined }}
              >
                <span className={cn('text-xs', alpha > 0.55 && 'text-primary-foreground')}>{d.getDate()}</span>
                {v > 0 && <span className={cn('text-[10px] leading-none tabular-nums', alpha > 0.55 ? 'text-primary-foreground' : 'text-muted-foreground')}>{compact(v)}</span>}
              </button>
            )
          })}
        </div>
      </Section>
      {selected && (
        <Section title={`${dayLabel(selected)} · ${money(byDay.get(selected) ?? 0)}`}>
          <div className="divide-y">
            {selectedTxs.map((t) => (
              <TxRow key={t.id} tx={t} category={t.category_id ? catMap.get(t.category_id) : null} account={t.account_id ? accountMap.get(t.account_id) : undefined} onClick={() => onOpenTx(t.id)} />
            ))}
            {selectedTxs.length === 0 && <Empty>Không có giao dịch</Empty>}
          </div>
        </Section>
      )}
    </>
  )
}

function MapStats() {
  const { month } = useThuChi()
  const txs = useMonthTransactions(month)
  const withLoc = (txs ?? []).filter((t) => t.lat != null && t.lng != null)
  return (
    <Section>
      <div className="flex flex-col items-center gap-2 px-4 py-10 text-center text-sm text-muted-foreground">
        <MapPin className="size-8" />
        <p>Bản đồ chi tiêu sẽ có ở giai đoạn 3.</p>
        <p>Tháng này có {withLoc.length} giao dịch đã gắn vị trí.</p>
      </div>
    </Section>
  )
}

interface DebtPerson {
  key: string
  name: string
  lent: number
  collected: number
  borrowed: number
  repaid: number
  txs: Transaction[]
}

function DebtStats({ onOpenTx }: { onOpenTx: (id: string) => void }) {
  const txs = useDebtTransactions()
  const accountMap = useAccountMap()
  const catMap = useCategoryMap()
  const [open, setOpen] = useState<string | null>(null)
  const people = useMemo(() => {
    const map = new Map<string, DebtPerson>()
    for (const t of txs ?? []) {
      const name = (t.party ?? '').trim() || 'Không rõ'
      const key = name.toLowerCase()
      const p = map.get(key) ?? { key, name, lent: 0, collected: 0, borrowed: 0, repaid: 0, txs: [] }
      if (t.kind === 'lend') p.lent += t.amount
      else if (t.kind === 'collect') p.collected += t.amount
      else if (t.kind === 'borrow') p.borrowed += t.amount
      else if (t.kind === 'repay') p.repaid += t.amount
      p.txs.push(t)
      map.set(key, p)
    }
    const outstanding = (p: DebtPerson) => Math.max(0, p.lent - p.collected) + Math.max(0, p.borrowed - p.repaid)
    return [...map.values()].sort((a, b) => outstanding(b) - outstanding(a) || b.txs[0].occurred_at.localeCompare(a.txs[0].occurred_at))
  }, [txs])
  const receivable = people.reduce((s, p) => s + Math.max(0, p.lent - p.collected), 0)
  const payable = people.reduce((s, p) => s + Math.max(0, p.borrowed - p.repaid), 0)

  return (
    <>
      <div className="grid grid-cols-2 gap-2">
        <div className="rounded-xl border bg-card p-3">
          <p className="text-xs text-muted-foreground">Cần thu</p>
          <p className="truncate text-lg font-bold tabular-nums" style={{ color: INCOME_COLOR }}>{money(receivable)}</p>
          <p className="text-xs text-muted-foreground">Cho vay chưa thu lại</p>
        </div>
        <div className="rounded-xl border bg-card p-3">
          <p className="text-xs text-muted-foreground">Cần trả</p>
          <p className="truncate text-lg font-bold tabular-nums" style={{ color: EXPENSE_COLOR }}>{money(payable)}</p>
          <p className="text-xs text-muted-foreground">Đi vay chưa trả hết</p>
        </div>
      </div>
      <Section title="Theo người">
        {people.length === 0 ? (
          <Empty>{txs === undefined ? 'Đang tải…' : 'Chưa có khoản vay nợ nào. Thêm giao dịch loại “Vay nợ” để theo dõi.'}</Empty>
        ) : (
          <div className="divide-y border-t">
            {people.map((p) => {
              const recv = p.lent - p.collected
              const pay = p.borrowed - p.repaid
              const expanded = open === p.key
              const parts = [p.lent && `cho vay ${compact(p.lent)}`, p.collected && `đã thu ${compact(p.collected)}`, p.borrowed && `đi vay ${compact(p.borrowed)}`, p.repaid && `đã trả ${compact(p.repaid)}`].filter(Boolean)
              return (
                <div key={p.key}>
                  <button type="button" aria-expanded={expanded} onClick={() => setOpen(expanded ? null : p.key)} className="flex min-h-14 w-full items-center gap-3 px-3 py-2.5 text-left hover:bg-muted/60">
                    <span className="inline-flex size-9 shrink-0 items-center justify-center rounded-full bg-violet-500/15 text-violet-600 dark:text-violet-400">
                      <Users className="size-4.5" />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium">{p.name}</span>
                      <span className="block truncate text-xs text-muted-foreground">{parts.join(' · ')}</span>
                    </span>
                    <span className="shrink-0 text-right text-sm tabular-nums">
                      {recv > 0 && <span className="block font-semibold text-emerald-600 dark:text-emerald-400">Thu {money(recv)}</span>}
                      {pay > 0 && <span className="block font-semibold text-red-600 dark:text-red-400">Trả {money(pay)}</span>}
                      {recv <= 0 && pay <= 0 && <span className="block text-xs text-muted-foreground">{recv < 0 || pay < 0 ? 'Dư' : 'Đã xong'}</span>}
                    </span>
                    {expanded ? <ChevronDown className="size-4 shrink-0 text-muted-foreground" /> : <ChevronRight className="size-4 shrink-0 text-muted-foreground" />}
                  </button>
                  {expanded && (
                    <div className="divide-y bg-muted/30">
                      {p.txs.map((t, i) => {
                        const day = dayKey(new Date(t.occurred_at))
                        const showDay = i === 0 || dayKey(new Date(p.txs[i - 1].occurred_at)) !== day
                        return (
                          <div key={t.id}>
                            {showDay && <p className="px-3 pt-2 text-xs font-medium text-muted-foreground">{dayLabel(day)}</p>}
                            <TxRow tx={t} category={t.category_id ? catMap.get(t.category_id) : null} account={t.account_id ? accountMap.get(t.account_id) : undefined} onClick={() => onOpenTx(t.id)} />
                          </div>
                        )
                      })}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </Section>
    </>
  )
}
