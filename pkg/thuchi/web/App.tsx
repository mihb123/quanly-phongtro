import { useCallback, useState } from 'react'
import { Home as HomeIcon, ListIcon, PieChart, Settings as SettingsIcon, Plus } from '@/components/icons'
import { cn } from '@/lib/utils'
import { ThuChiProvider } from './context'
import { useReviewQueue } from './hooks'
import { monthKey } from './format'
import type { Transport } from './types'
import { MonthPicker, SyncBadge } from './ui/common'
import { Overview } from './ui/Overview'
import { Transactions } from './ui/Transactions'
import { Stats } from './ui/Stats'
import { Settings } from './ui/Settings'
import { QuickAddSheet, TransactionSheet } from './ui/TxSheets'

type Tab = 'overview' | 'transactions' | 'stats' | 'settings'
type Sub = Parameters<typeof Settings>[0]['sub']
const TAB_KEY = 'thuchi.tab'

function readTab(): Tab {
  try {
    const t = localStorage.getItem(TAB_KEY)
    if (t === 'overview' || t === 'transactions' || t === 'stats' || t === 'settings') return t
  } catch {
    /* ignore */
  }
  return 'overview'
}

export interface ThuChiAppProps {
  ownerId: string
  transport: Transport
  userEmail?: string
  offline?: boolean
  embedded?: boolean
  onLogout?: () => void
}

export default function ThuChiApp(props: ThuChiAppProps) {
  const [month, setMonthState] = useState(() => monthKey(new Date()))
  const setMonth = useCallback((m: string) => setMonthState(m), [])
  return (
    <ThuChiProvider ownerId={props.ownerId} transport={props.transport} month={month} setMonth={setMonth}>
      <Shell {...props} />
    </ThuChiProvider>
  )
}

function Shell({ onLogout, userEmail, embedded }: ThuChiAppProps) {
  const [tab, setTabState] = useState<Tab>(readTab)
  const [sub, setSub] = useState<Sub>(null)
  const [txView, setTxView] = useState<string | undefined>()
  const [openTx, setOpenTx] = useState<string | null>(null)
  const [adding, setAdding] = useState(false)
  const review = useReviewQueue()

  const setTab = (t: Tab) => {
    setTabState(t)
    try {
      localStorage.setItem(TAB_KEY, t)
    } catch {
      /* ignore */
    }
    if (!embedded) window.scrollTo({ top: 0 })
  }

  const goto = (t: 'transactions' | 'stats' | 'settings', s?: string) => {
    if (t === 'settings') setSub((s as Sub) ?? null)
    if (t === 'transactions') setTxView(s)
    setTab(t)
  }

  const nav: { key: Tab; label: string; icon: typeof HomeIcon; badge?: number }[] = [
    { key: 'overview', label: 'Tổng quan', icon: HomeIcon },
    { key: 'transactions', label: 'Giao dịch', icon: ListIcon, badge: review.length },
    { key: 'stats', label: 'Thống kê', icon: PieChart },
    { key: 'settings', label: 'Cài đặt', icon: SettingsIcon },
  ]

  const navButtons = nav.map((n) => (
    <button
      key={n.key}
      type="button"
      aria-current={tab === n.key ? 'page' : undefined}
      onClick={() => {
        if (n.key === 'settings' && tab === 'settings') setSub(null)
        if (n.key === 'transactions') setTxView(undefined)
        setTab(n.key)
      }}
      className={cn(
        'relative flex flex-col items-center justify-center gap-0.5 text-xs',
        embedded ? 'min-h-12 rounded-md' : 'min-h-14',
        tab === n.key ? (embedded ? 'bg-background text-primary shadow-sm' : 'text-primary') : 'text-muted-foreground',
      )}
    >
      <n.icon className="size-5" />
      {n.label}
      {!!n.badge && <span className="absolute top-1.5 left-1/2 ml-2 min-w-4 rounded-full bg-amber-500 px-1 text-[10px] leading-4 text-white">{n.badge > 99 ? '99+' : n.badge}</span>}
    </button>
  ))

  return (
    <div data-thuchi="" className={cn('mx-auto flex w-full max-w-xl flex-col bg-background', !embedded && 'min-h-dvh')}>
      {embedded && <nav className="grid grid-cols-4 gap-1 rounded-lg bg-muted p-1">{navButtons}</nav>}
      <header
        className={cn(
          'flex items-center justify-between gap-2 border-b bg-background/90 px-3 pb-2 backdrop-blur',
          embedded ? 'pt-2' : 'sticky top-0 z-30 pt-[max(0.5rem,env(safe-area-inset-top))]',
        )}
      >
        <span className="w-9 font-bold text-primary">₫</span>
        {tab !== 'settings' ? <MonthPicker /> : <span className="font-semibold">Cài đặt</span>}
        <SyncBadge />
      </header>
      <main className={cn('flex-1 pt-3', embedded ? 'pb-24 md:pb-20' : 'px-3 pb-[calc(5.5rem+env(safe-area-inset-bottom))]')}>
        {tab === 'overview' && <Overview onOpenTx={setOpenTx} onGoto={goto} />}
        {tab === 'transactions' && <Transactions key={txView ?? 'month'} onOpenTx={setOpenTx} initialView={txView} />}
        {tab === 'stats' && <Stats onOpenTx={setOpenTx} />}
        {tab === 'settings' && <Settings sub={sub} setSub={setSub} onLogout={onLogout} userEmail={userEmail} />}
      </main>
      {(tab === 'overview' || tab === 'transactions') && (
        <button
          type="button"
          aria-label="Thêm giao dịch"
          onClick={() => setAdding(true)}
          className={cn(
            'fixed z-30 flex size-14 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg active:scale-95',
            embedded ? 'right-4 bottom-[calc(5rem+env(safe-area-inset-bottom))] md:right-8 md:bottom-8' : 'right-[max(1rem,calc(50vw-17rem))] bottom-[calc(4.75rem+env(safe-area-inset-bottom))]',
          )}
        >
          <Plus className="size-6" />
        </button>
      )}
      {!embedded && (
        <nav className="fixed inset-x-0 bottom-0 z-30 border-t bg-background/95 pb-[env(safe-area-inset-bottom)] backdrop-blur">
          <div className="mx-auto grid max-w-xl grid-cols-4">{navButtons}</div>
        </nav>
      )}
      <TransactionSheet txId={openTx} onClose={() => setOpenTx(null)} />
      <QuickAddSheet open={adding} onClose={() => setAdding(false)} />
    </div>
  )
}
