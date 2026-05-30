import { useSelectedStore } from '@/data/selectedData'

// Import extracted components
import { Sidebar } from '@/components/home/sidebar/Sidebar'
import { DashboardView } from '@/components/home/DashboardView'
import { HouseRoomsView } from '@/components/home/HouseRoomsView'
import { TenantsView } from '@/components/home/TenantsView'
import { InvoicesView } from '@/components/home/InvoicesView'

export default function HomePage() {
  const { activeTab } = useSelectedStore()

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 selection:bg-purple-200/50">
      {/* Sidebar background effect */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-0 -left-40 w-96 h-96 rounded-full bg-purple-200/50 blur-3xl" />
      </div>

      <div className="flex h-screen overflow-hidden relative">
        <Sidebar />
        
        <main className="flex-1 overflow-auto p-4 md:p-8 relative transition-all duration-300">
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
    </div>
  )
}
