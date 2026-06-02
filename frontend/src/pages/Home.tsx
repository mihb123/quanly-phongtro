import { useSelectedStore } from '@/data/selectedData'

// Import extracted components
import { Sidebar } from '@/components/home/sidebar/Sidebar'
import { MobileNav } from '@/components/home/sidebar/MobileNav'
import { DashboardView } from '@/components/home/DashboardView'
import { HouseRoomsView } from '@/components/home/HouseRoomsView'
import { TenantsView } from '@/components/home/TenantsView'
import { InvoicesView } from '@/components/home/InvoiceView'

export default function HomePage() {
  const { activeTab } = useSelectedStore()

  return (
    <div className="min-h-[100dvh] bg-background text-foreground selection:bg-primary/20 flex flex-col md:flex-row pb-16 md:pb-0">
      <Sidebar />
      <MobileNav />
      
      <main className="flex-1 w-full md:w-auto p-4 md:p-8 relative transition-all duration-300 overflow-x-hidden">
        <div className="max-w-6xl mx-auto space-y-8">
          {activeTab === 'dashboard' && (
            <DashboardView />
          )}

          {activeTab === 'house_rooms' && (
            <HouseRoomsView />
          )}

          {activeTab === 'tenants' && (
            <TenantsView />
          )}

          {activeTab === 'invoices' && (
            <InvoicesView />
          )}
        </div>
      </main>
    </div>
  )
}
