import { useLayoutEffect, useRef } from 'react'
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

const SCROLL_STORAGE_KEY = 'home:tabScroll'

function readTabScroll(): Partial<Record<TabType, number>> {
  try {
    const raw = sessionStorage.getItem(SCROLL_STORAGE_KEY)
    const parsed: unknown = raw ? JSON.parse(raw) : null
    return parsed && typeof parsed === 'object' ? (parsed as Partial<Record<TabType, number>>) : {}
  } catch {
    return {}
  }
}

function writeTabScroll(value: Partial<Record<TabType, number>>) {
  try {
    sessionStorage.setItem(SCROLL_STORAGE_KEY, JSON.stringify(value))
  } catch {
    // Bỏ qua khi trình duyệt chặn sessionStorage.
  }
}

// Mỗi tab giữ vị trí cuộn riêng: rời tab thì lưu, quay lại thì cuộn về chỗ cũ khi nội dung đã đủ cao.
function useTabScrollRestoration(activeTab: TabType) {
  const containerRef = useRef<HTMLDivElement>(null)
  const positionsRef = useRef<Partial<Record<TabType, number>>>(readTabScroll())

  useLayoutEffect(() => {
    const container = containerRef.current
    if (!container) return
    const target = positionsRef.current[activeTab] ?? 0
    container.scrollTop = target

    let restoring = target > 0 && container.scrollTop < target
    let observer: ResizeObserver | null = null
    let timeoutId: number | undefined
    const stopRestoring = () => {
      restoring = false
      observer?.disconnect()
      window.clearTimeout(timeoutId)
    }

    if (restoring && typeof ResizeObserver !== 'undefined' && container.firstElementChild) {
      observer = new ResizeObserver(() => {
        if (!restoring) return
        container.scrollTop = target
        if (container.scrollTop >= target - 1) stopRestoring()
      })
      observer.observe(container.firstElementChild)
      timeoutId = window.setTimeout(stopRestoring, 3000)
    }

    const handleScroll = () => {
      if (restoring) return
      positionsRef.current = { ...positionsRef.current, [activeTab]: container.scrollTop }
      writeTabScroll(positionsRef.current)
    }
    const handleUserScroll = () => {
      if (restoring) stopRestoring()
    }

    container.addEventListener('scroll', handleScroll, { passive: true })
    container.addEventListener('wheel', handleUserScroll, { passive: true })
    container.addEventListener('touchstart', handleUserScroll, { passive: true })
    return () => {
      stopRestoring()
      container.removeEventListener('scroll', handleScroll)
      container.removeEventListener('wheel', handleUserScroll)
      container.removeEventListener('touchstart', handleUserScroll)
    }
  }, [activeTab])

  return containerRef
}

export default function HomePage() {
  const { activeTab } = useSelectedStore()
  const scrollContainerRef = useTabScrollRestoration(activeTab)

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

        <div
          ref={scrollContainerRef}
          className="flex-1 overflow-x-hidden overflow-y-auto overscroll-contain p-4 pt-[calc(5rem+env(safe-area-inset-top))] pb-[calc(5rem+env(safe-area-inset-bottom))] md:p-6 md:pt-6 md:pb-6"
        >
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
