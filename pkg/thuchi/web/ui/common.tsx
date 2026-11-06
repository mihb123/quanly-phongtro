import { AlertCircle, Bank, CheckCircle2, ChevronLeft, ChevronRight, CloudUpload, CreditCard, Cash, Loader2, Pencil, Tag, TransferIcon, Users, Wallet, WifiOff } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useSyncStatus, useThuChi } from '../hooks'
import { money, monthLabel, relativeTime, shiftMonth, timeLabel, monthKey } from '../format'
import { ICONS, KIND_LABEL, txTitle } from './icons'
import { isDebtKind, type Account, type Category, type Transaction } from '../types'

export function CategoryIcon({ category, kind, className }: { category?: Category | null; kind?: Transaction['kind']; className?: string }) {
  const debt = !!kind && isDebtKind(kind)
  const Icon = kind === 'transfer' ? TransferIcon : kind === 'adjustment' ? Pencil : debt ? Users : category ? (ICONS[category.icon] ?? Tag) : Tag
  const color = kind === 'transfer' || kind === 'adjustment' ? '#64748b' : debt ? '#8b5cf6' : (category?.color ?? '#94a3b8')
  return (
    <span
      className={cn('inline-flex size-9 shrink-0 items-center justify-center rounded-full', className)}
      style={{ backgroundColor: `${color}22`, color }}
    >
      <Icon className="size-4.5" />
    </span>
  )
}

export function AccountIcon({ account }: { account?: Account }) {
  const Icon = account?.kind === 'cash' ? Cash : account?.kind === 'ewallet' ? Wallet : account?.kind === 'credit' ? CreditCard : Bank
  return (
    <span className="inline-flex size-9 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
      <Icon className="size-4.5" />
    </span>
  )
}

export function Amount({ tx, className }: { tx: Transaction; className?: string }) {
  const positive = tx.direction === 'in'
  const muted = tx.kind === 'transfer' || tx.kind === 'card_payment' || tx.kind === 'adjustment' || tx.excluded
  return (
    <span
      className={cn(
        'tabular-nums font-semibold whitespace-nowrap',
        muted ? 'text-muted-foreground' : positive ? 'text-emerald-600 dark:text-emerald-400' : 'text-foreground',
        className,
      )}
    >
      {positive ? '+' : '−'}
      {money(tx.amount)}
    </span>
  )
}

export function TxRow({ tx, category, account, onClick }: { tx: Transaction; category?: Category | null; account?: Account; onClick?: () => void }) {
  const badges: string[] = []
  if (tx.kind === 'transfer') badges.push(tx.pair_status === 'suggested' ? 'Nghi nội bộ' : 'Nội bộ')
  else if (tx.pair_status === 'suggested') badges.push('Nghi nội bộ')
  if (tx.kind === 'refund') badges.push('Hoàn tiền')
  if (tx.kind === 'card_payment') badges.push('Trả thẻ')
  if (tx.recurrence_id) badges.push('Định kỳ')
  if (tx.excluded) badges.push('Không tính')
  if (!tx.change_seq) badges.push('Chưa đồng bộ')
  return (
    <button type="button" onClick={onClick} className="flex w-full items-center gap-3 px-3 py-2.5 text-left hover:bg-muted/60 active:bg-muted min-h-14">
      <CategoryIcon category={category} kind={tx.kind} />
      <span className="min-w-0 flex-1">
        <span className="block truncate font-medium">{txTitle(tx, category)}</span>
        <span className="flex items-center gap-1.5 text-xs text-muted-foreground min-w-0">
          <span className="shrink-0">{timeLabel(tx.occurred_at)}</span>
          <span className="truncate">· {isDebtKind(tx.kind) ? [KIND_LABEL[tx.kind], tx.party].filter(Boolean).join(' ') : (category?.name ?? (tx.kind === 'refund' || tx.kind === 'card_payment' ? undefined : KIND_LABEL[tx.kind]) ?? 'Chưa phân loại')}</span>
          {account && <span className="hidden truncate sm:inline">· {account.name}</span>}
        </span>
        {badges.length > 0 && (
          <span className="mt-0.5 flex flex-wrap gap-1">
            {badges.map((b) => (
              <span key={b} className="rounded bg-muted px-1.5 py-px text-[11px] text-muted-foreground">
                {b}
              </span>
            ))}
          </span>
        )}
      </span>
      <Amount tx={tx} />
    </button>
  )
}

export function MonthPicker() {
  const { month, setMonth } = useThuChi()
  const current = monthKey(new Date())
  return (
    <div className="flex items-center gap-1">
      <Button variant="ghost" size="icon-sm" aria-label="Tháng trước" onClick={() => setMonth(shiftMonth(month, -1))}>
        <ChevronLeft />
      </Button>
      <button type="button" className="min-w-28 text-center font-semibold" onClick={() => setMonth(current)}>
        {monthLabel(month)}
      </button>
      <Button variant="ghost" size="icon-sm" aria-label="Tháng sau" disabled={month >= current} onClick={() => setMonth(shiftMonth(month, 1))}>
        <ChevronRight />
      </Button>
    </div>
  )
}

export function SyncBadge() {
  const status = useSyncStatus()
  const { engine } = useThuChi()
  let icon = <CheckCircle2 className="size-4 text-emerald-600" />
  let label = `Đã đồng bộ ${relativeTime(status.lastSyncAt)}`
  if (status.state === 'syncing') {
    icon = <Loader2 className="size-4 animate-spin" />
    label = 'Đang đồng bộ'
  } else if (status.state === 'offline') {
    icon = <WifiOff className="size-4 text-amber-600" />
    label = status.pending ? `Ngoại tuyến · ${status.pending} thay đổi chờ gửi` : 'Ngoại tuyến'
  } else if (status.state === 'error') {
    icon = <AlertCircle className="size-4 text-destructive" />
    label = `Lỗi đồng bộ: ${status.error ?? ''}`
  } else if (status.pending) {
    icon = <CloudUpload className="size-4 text-amber-600" />
    label = `${status.pending} thay đổi chờ gửi`
  }
  return (
    <Button variant="ghost" size="icon-sm" aria-label={label} title={label} onClick={() => engine.sync()}>
      {icon}
    </Button>
  )
}

export function Empty({ children }: { children: React.ReactNode }) {
  return <div className="px-4 py-10 text-center text-sm text-muted-foreground">{children}</div>
}

export function Section({ title, action, children, className }: { title?: React.ReactNode; action?: React.ReactNode; children: React.ReactNode; className?: string }) {
  return (
    <section className={cn('rounded-xl border bg-card', className)}>
      {(title || action) && (
        <div className="flex items-center justify-between gap-2 px-3 pt-3 pb-1">
          <h2 className="text-sm font-semibold text-muted-foreground">{title}</h2>
          {action}
        </div>
      )}
      {children}
    </section>
  )
}
