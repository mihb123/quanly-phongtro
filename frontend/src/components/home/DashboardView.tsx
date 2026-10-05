import { useState, useEffect, useMemo } from 'react'
import { Plus, Users, Building2, TrendingUp, AlertCircle, AlertTriangle, Clock, CheckCircle2, Receipt, ArrowRight, DoorOpen } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { PageHeader } from '@/components/shared/PageHeader'
import { StatCard } from '@/components/shared/StatCard'
import { SectionCard } from '@/components/shared/SectionCard'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { EmptyState } from '@/components/shared/EmptyState'
import { formatCurrency } from '@/utils/format'
import { useHouseStore } from '@/data/houseData'
import { invoiceBreakdownKey, UNPAID_INVOICE_STATUSES } from '@/data/invoiceData'
import { getInvoiceBreakdown, getInvoicesPage } from '@/api/invoice'
import { getAvailableRooms, getRoomStats } from '@/api/room'
import { invalidateQueries, queryKey, useQuery } from '@/lib/queryCache'
import { useHouseCostStore } from '@/data/houseCostData'
import { useSelectedStore } from '@/data/selectedData'
import { CreateHouseModal } from './modals/CreateHouseModal'
import { cn } from '@/lib/utils'

const DASHBOARD_LIST_LIMIT = 20

// View tổng quan: thống kê nhanh + danh sách hóa đơn cần thu/gần đây + phòng trống.
export function DashboardView() {
  const [showCreateHouse, setShowCreateHouse] = useState(false)
  const { houses } = useHouseStore()
  const { summaries, period, fetchSummaries } = useHouseCostStore()
  const { setActiveTab, selectHouse } = useSelectedStore()

  useEffect(() => {
    fetchSummaries()
  }, [fetchSummaries, period])

  const breakdownQuery = useQuery(invoiceBreakdownKey({ period }), () => getInvoiceBreakdown({ period }))
  const unpaidQuery = useQuery(queryKey('invoices:unpaid', { period, limit: DASHBOARD_LIST_LIMIT }), () =>
    getInvoicesPage({ status: UNPAID_INVOICE_STATUSES, period, page: 1, limit: DASHBOARD_LIST_LIMIT }),
  )
  const recentQuery = useQuery(queryKey('invoices:recent', { limit: 5 }), () => getInvoicesPage({ page: 1, limit: 5 }))
  const roomStatsQuery = useQuery(queryKey('rooms:stats', {}), () => getRoomStats())
  const availableQuery = useQuery(queryKey('rooms:available', { limit: DASHBOARD_LIST_LIMIT }), () => getAvailableRooms(DASHBOARD_LIST_LIMIT))

  const breakdown = breakdownQuery.data
  const expectedRevenue = breakdown?.expected.total_amount ?? 0
  const collectedRevenue = breakdown?.collected.total_amount ?? 0
  const unpaidRevenue = expectedRevenue - collectedRevenue
  const pendingInvoicesCount = breakdown ? breakdown.expected.invoice_count - breakdown.collected.invoice_count : 0
  const unpaidInvoices = unpaidQuery.data?.items ?? []
  const unpaidTotal = unpaidQuery.data?.total ?? 0
  const recentInvoices = recentQuery.data?.items ?? []
  const availableRooms = availableQuery.data?.items ?? []
  const availableTotal = availableQuery.data?.total ?? 0

  const roomStats = roomStatsQuery.data
  const totalRooms = roomStats?.total ?? 0
  const occupiedRooms = roomStats?.occupied ?? 0
  const totalTenants = roomStats?.tenants ?? 0
  const occupancyRate = totalRooms > 0 ? Math.round((occupiedRooms / totalRooms) * 100) : 0
  const loadFailed = [breakdownQuery, unpaidQuery, recentQuery, roomStatsQuery, availableQuery].some(q => q.error !== undefined)

  const totalProfit = useMemo(() => {
    return (summaries || []).reduce((sum, s) => sum + s.profit, 0);
  }, [summaries]);

  const navigateToHouse = (houseId: string) => {
    const house = houses.find(h => h.id === houseId) || null;
    selectHouse(house);
    setActiveTab('house_rooms');
  };

  return (
    <div className="space-y-8 safe-fade-in pb-12">
      {showCreateHouse && (
        <CreateHouseModal onClose={() => setShowCreateHouse(false)} />
      )}

      <PageHeader
        title="Tổng quan"
        description={`Quản lý ${houses.length} nhà trọ với tổng cộng ${totalRooms} phòng.`}
        action={
          <Button onClick={() => setShowCreateHouse(true)}>
            <Plus data-icon="inline-start" /> Tạo nhà trọ
          </Button>
        }
      />

      {loadFailed && (
        <div role="alert" className="flex items-center justify-between gap-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          <span className="flex items-center gap-2"><AlertTriangle className="size-4 shrink-0" /> Một phần số liệu chưa tải được.</span>
          <Button variant="outline" size="sm" onClick={() => { invalidateQueries('invoices:'); invalidateQueries('rooms:') }}>Thử lại</Button>
        </div>
      )}

      {/* Balanced 4-Column Stats Grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
        <StatCard
          label="Lợi nhuận ròng"
          value={formatCurrency(totalProfit)}
          icon={TrendingUp}
          tone={totalProfit >= 0 ? 'positive' : 'negative'}
          sub={`Kỳ: ${period}`}
          onClick={() => setActiveTab('revenue')}
        />
        <StatCard
          label="Doanh thu dự kiến"
          value={formatCurrency(expectedRevenue)}
          icon={TrendingUp}
          tone="info"
          sub={
            <span className="flex items-center gap-1">
              <CheckCircle2 className="size-3 shrink-0 text-success" />
              <span className="truncate">Đã thu: {formatCurrency(collectedRevenue)} · Kỳ {period}</span>
            </span>
          }
        />
        <StatCard
          label="Hóa đơn chưa thu"
          value={`${pendingInvoicesCount} phiếu`}
          icon={AlertCircle}
          tone="warning"
          sub={`Tổng nợ: ${formatCurrency(unpaidRevenue)} · Kỳ ${period}`}
        />
        <StatCard
          label="Tỷ lệ lấp đầy"
          value={`${occupancyRate}%`}
          icon={Building2}
          tone="positive"
          sub={`Đang thuê: ${occupiedRooms} / ${totalRooms} phòng`}
        />
        <StatCard
          label="Khách lưu trú"
          value={`${totalTenants} người`}
          icon={Users}
          tone="info"
          sub="Đang hoạt động trên hệ thống"
        />
      </div>

      {/* Main Content Grid: Lists */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

        {/* Left Column: Unpaid Invoices & Recent Activities (Span 2) */}
        <div className="lg:col-span-2 flex flex-col gap-6">

          {/* Unpaid Invoices */}
          <SectionCard
            icon={AlertCircle}
            title={`Cần thu tiền kỳ ${period} (${unpaidTotal})`}
            action={
              <Button variant="ghost" size="sm" onClick={() => setActiveTab('invoices')} className="text-muted-foreground">
                Xem tất cả
                <ArrowRight data-icon="inline-end" />
              </Button>
            }
            bodyClassName="divide-y divide-border/40 max-h-[300px] overflow-y-auto"
          >
            {unpaidQuery.isLoading ? (
              <div className="p-5 text-center text-sm text-muted-foreground">Đang tải dữ liệu...</div>
            ) : unpaidInvoices.length === 0 ? (
              <EmptyState title="Không có hóa đơn nợ." />
            ) : (
              unpaidInvoices.map(inv => (
                <div key={inv.id} className="p-4 hover:bg-secondary/30 transition-colors flex justify-between items-center cursor-pointer" onClick={() => setActiveTab('invoices')}>
                  <div>
                    <p className="text-sm font-medium text-foreground">{inv.room_name}</p>
                    <p className="text-xs text-muted-foreground mt-0.5">Kỳ: {inv.period}</p>
                  </div>
                  <div className="text-right space-y-1">
                    <p className="text-sm font-semibold tabular-nums text-foreground">{formatCurrency(inv.total_amount)}</p>
                    <StatusBadge status={inv.status} />
                  </div>
                </div>
              ))
            )}
          </SectionCard>

          {/* Recent Invoices */}
          <SectionCard
            icon={Receipt}
            title="Hóa đơn gần đây"
            bodyClassName="divide-y divide-border/40"
          >
            {recentQuery.isLoading ? (
              <div className="p-5 text-center text-sm text-muted-foreground">Đang tải dữ liệu...</div>
            ) : recentInvoices.length === 0 ? (
              <EmptyState title="Chưa có hóa đơn nào được tạo." />
            ) : (
              recentInvoices.map(inv => (
                <div key={inv.id} className="p-4 flex justify-between items-center">
                  <div className="flex items-center gap-3">
                    <div className={cn(
                      'flex size-8 shrink-0 items-center justify-center rounded-full',
                      inv.status === 'PAID' ? 'bg-success/15 text-success' : 'bg-secondary text-muted-foreground',
                    )}>
                      {inv.status === 'PAID' ? <CheckCircle2 className="size-4" /> : <Clock className="size-4" />}
                    </div>
                    <div>
                      <p className="text-sm font-medium text-foreground">{inv.room_name} <span className="text-muted-foreground font-normal">· Kỳ {inv.period}</span></p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {new Date(inv.created_at).toLocaleDateString('vi-VN')}
                      </p>
                    </div>
                  </div>
                  <p className="text-sm font-semibold tabular-nums text-foreground">{formatCurrency(inv.total_amount)}</p>
                </div>
              ))
            )}
          </SectionCard>

        </div>

        {/* Right Column: Available Rooms (Span 1) */}
        <div className="lg:col-span-1 relative min-h-[400px]">
          <SectionCard
            icon={DoorOpen}
            title={`Phòng đang trống (${availableTotal})`}
            className="flex flex-col lg:absolute lg:inset-0 max-h-[600px] lg:max-h-none"
            bodyClassName="divide-y divide-border/40 overflow-y-auto flex-1"
          >
            {availableQuery.isLoading ? (
              <div className="p-5 text-center text-sm text-muted-foreground">Đang tải...</div>
            ) : availableRooms.length === 0 ? (
              <EmptyState title="Không có phòng trống." />
            ) : (
              availableRooms.map(room => (
                <div
                  key={room.id}
                  className="p-4 hover:bg-secondary/30 transition-colors cursor-pointer group"
                  onClick={() => navigateToHouse(room.house_id)}
                >
                  <div className="flex justify-between items-start mb-1">
                    <p className="text-sm font-medium text-foreground">{room.name}</p>
                    <p className="text-sm font-semibold tabular-nums text-foreground">{room.price ? formatCurrency(room.price) : 'Chưa đặt giá'}</p>
                  </div>
                  <div className="flex items-center justify-between mt-2">
                    <p className="text-xs text-muted-foreground bg-secondary px-2 py-0.5 rounded-sm line-clamp-1">{room.house_name}</p>
                    <ArrowRight className="size-3.5 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
                  </div>
                </div>
              ))
            )}
          </SectionCard>
        </div>

      </div>
    </div>
  )
}
