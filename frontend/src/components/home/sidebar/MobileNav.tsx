import { LayoutDashboard, Building2, Users, Receipt, Wallet } from '@/components/icons'
import { useSelectedStore, type TabType } from '@/data/selectedData'
import { cn } from '@/lib/utils'

export function MobileNav() {
  const { activeTab, setActiveTab } = useSelectedStore()

  const tabs: { id: TabType; icon: typeof LayoutDashboard; label: string }[] = [
    { id: 'dashboard', icon: LayoutDashboard, label: 'Tổng quan' },
    { id: 'house_rooms', icon: Building2, label: 'Nhà trọ' },
    { id: 'tenants', icon: Users, label: 'Khách thuê' },
    { id: 'invoices', icon: Receipt, label: 'Hóa đơn' },
    { id: 'revenue', icon: Wallet, label: 'Doanh thu' },
  ]

  return (
    <nav
      aria-label="Điều hướng chính"
      className="fixed bottom-0 left-0 right-0 z-50 border-t border-border bg-background pb-[env(safe-area-inset-bottom)] md:hidden"
    >
      <div className="grid grid-cols-5 px-1 py-1">
        {tabs.map((tab) => {
          const isActive = activeTab === tab.id
          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              aria-current={isActive ? 'page' : undefined}
              className={cn(
                'flex min-h-14 min-w-0 cursor-pointer touch-manipulation flex-col items-center justify-center gap-1 rounded-md px-1 transition-colors active:scale-95 focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none',
                isActive ? 'text-primary' : 'text-muted-foreground hover:text-foreground',
              )}
            >
              <tab.icon className="size-5" aria-hidden />
              <span className={cn('max-w-full truncate text-[11px]', isActive ? 'font-semibold' : 'font-medium')}>{tab.label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}
