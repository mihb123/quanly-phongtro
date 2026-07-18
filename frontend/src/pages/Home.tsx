import { useSelectedStore, type TabType } from '@/data/selectedData'
import { SidebarInset, SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'
import { Separator } from '@/components/ui/separator'

// Import extracted components
import { AppSidebar } from '@/components/home/sidebar/Sidebar'
import { MobileNav } from '@/components/home/sidebar/MobileNav'
import { MobileHeader } from '@/components/home/MobileHeader'
import { DashboardView } from '@/components/home/DashboardView'
import { HouseRoomsView } from '@/components/home/HouseRoomsView'
import { TenantsView } from '@/components/home/TenantView'
import { InvoicesView } from '@/components/home/InvoiceView'
import { RevenueView } from '@/components/home/RevenueView'
import { SettingsView } from '@/components/home/SettingsView'

// Tiêu đề hiển thị trên thanh header desktop theo tab đang mở.
const TAB_TITLES: Record<TabType, string> = {
  dashboard: 'Tổng quan',
  house_rooms: 'Nhà trọ',
  tenants: 'Khách thuê',
  invoices: 'Hóa đơn',
  revenue: 'Doanh thu',
  settings: 'Cài đặt',
}

export default function HomePage() {
  const { activeTab } = useSelectedStore()

  return (
    <SidebarProvider className="h-svh overflow-hidden">
      <AppSidebar />
      <MobileNav />
      <MobileHeader />

      <SidebarInset className="overflow-hidden">
        <header className="hidden h-14 shrink-0 items-center gap-2 border-b px-4 md:flex">
          <SidebarTrigger className="-ml-1" />
          <Separator orientation="vertical" className="h-4" />
          <h1 className="text-sm font-medium">{TAB_TITLES[activeTab]}</h1>
        </header>

        <div className="flex-1 overflow-x-hidden overflow-y-auto p-4 pt-20 pb-20 md:p-6 md:pt-6 md:pb-6">
          <div className="mx-auto flex max-w-6xl flex-col gap-6">
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

            {activeTab === 'revenue' && (
              <RevenueView />
            )}

            {activeTab === 'settings' && (
              <SettingsView />
            )}
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
