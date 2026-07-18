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
    <nav className="fixed bottom-0 left-0 right-0 z-50 border-t border-border bg-background safe-bottom pb-[env(safe-area-inset-bottom)] md:hidden">
      <div className="flex items-center justify-around px-2 py-1">
        {tabs.map((tab) => {
          const isActive = activeTab === tab.id
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'flex min-w-16 touch-target cursor-pointer flex-col items-center justify-center gap-1 rounded-md transition-colors active:scale-95',
                isActive ? 'text-primary' : 'text-muted-foreground hover:text-foreground',
              )}
            >
              <tab.icon className="size-5" />
              <span className="text-[11px] font-medium">{tab.label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}
