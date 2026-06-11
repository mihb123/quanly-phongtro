import { LayoutDashboard, Users, Receipt, Building } from 'lucide-react'
import { useSelectedStore } from '@/data/selectedData'

export function MobileNav() {
  const { activeTab, setActiveTab } = useSelectedStore()

  const tabs = [
    { id: 'dashboard', icon: LayoutDashboard, label: 'Tổng quan' },
    { id: 'house_rooms', icon: Building, label: 'Nhà trọ' },
    { id: 'tenants', icon: Users, label: 'Khách' },
    { id: 'invoices', icon: Receipt, label: 'Hóa đơn' },
  ] as const

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-50 bg-card border-t border-border shadow-[0_-4px_24px_rgba(0,0,0,0.02)] safe-bottom pb-[env(safe-area-inset-bottom)] md:hidden">
      <div className="flex items-center justify-around p-2">
        {tabs.map((tab) => {
          const isActive = activeTab === tab.id
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as 'dashboard' | 'house_rooms' | 'tenants' | 'invoices')}
              className={`flex flex-col items-center justify-center min-w-[64px] touch-target rounded-xl transition-all active:scale-95 cursor-pointer ${
                isActive ? 'text-primary' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              <div className={`p-1.5 rounded-lg mb-1 transition-colors ${isActive ? 'bg-primary/10' : ''}`}>
                <tab.icon className={`w-5 h-5 ${isActive ? 'fill-primary/20' : ''}`} />
              </div>
              <span className={`text-[10px] font-bold ${isActive ? 'text-primary' : ''}`}>
                {tab.label}
              </span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}
