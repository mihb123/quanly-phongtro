import { useState, useEffect, useMemo } from 'react'
import { Plus, Users, Building2, TrendingUp, AlertCircle, Clock, CheckCircle2, Receipt, ArrowRight, DoorOpen } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { PageHeader } from '@/components/shared/PageHeader'
import { StatCard } from '@/components/shared/StatCard'
import { SectionCard } from '@/components/shared/SectionCard'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { EmptyState } from '@/components/shared/EmptyState'
import { formatCurrency } from '@/utils/format'
import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { useInvoiceStore } from '@/data/invoiceData'
import { useTenantStore } from '@/data/tenantData'
import { useHouseCostStore } from '@/data/houseCostData'
import { useSelectedStore } from '@/data/selectedData'
import { CreateHouseModal } from './modals/CreateHouseModal'
import { type Room } from '@/api/room'
import { cn } from '@/lib/utils'

// View tổng quan: thống kê nhanh + danh sách hóa đơn cần thu/gần đây + phòng trống.
export function DashboardView() {
  const [showCreateHouse, setShowCreateHouse] = useState(false)
  const { houses } = useHouseStore()
  const { getRoomsByHouse } = useRoomStore()
  const { tenantsByHouse, fetchTenants } = useTenantStore()
  const { invoices, fetchInvoices } = useInvoiceStore()
  const { summaries, period, fetchSummaries } = useHouseCostStore()
  const { setActiveTab, selectHouse } = useSelectedStore()

  const [allRooms, setAllRooms] = useState<(Room & { houseName?: string })[]>([])
  const [isLoadingData, setIsLoadingData] = useState(true)

  // Fetch invoices and summaries on mount
  useEffect(() => {
    fetchInvoices()
    fetchSummaries()
  }, [fetchInvoices, fetchSummaries, period])

  // Fetch rooms for all houses to calculate occupancy and get available rooms
  useEffect(() => {
    let isMounted = true
    async function loadData() {
      if (houses.length === 0) {
        if (isMounted) {
          setAllRooms([])
          setIsLoadingData(false)
        }
        return
      }

      setIsLoadingData(true)
      try {
        const roomsList: (Room & { houseName?: string })[] = []

        await Promise.all(houses.map(async (house) => {
          const rooms = await getRoomsByHouse(house.id)
          rooms.forEach(r => roomsList.push({ ...r, houseName: house.name }))
          fetchTenants(house.id) // trigger tenant fetch to populate tenantsByHouse
        }))

        if (isMounted) {
          setAllRooms(roomsList)
        }
      } catch (err) {
        console.error("Error loading data:", err)
      } finally {
        if (isMounted) setIsLoadingData(false)
      }
    }

    loadData()
    return () => { isMounted = false }
  }, [houses, getRoomsByHouse, fetchTenants])

  // Calculate invoice stats and lists
  const { expectedRevenue, collectedRevenue, unpaidRevenue, pendingInvoicesCount, recentInvoices, unpaidInvoices } = useMemo(() => {
    let expected = 0;
    let collected = 0;
    let unpaid = 0;
    let pendingCount = 0;

    invoices.forEach(inv => {
      expected += inv.total_amount;
      if (inv.status === 'PAID') {
        collected += inv.total_amount;
      } else {
        unpaid += inv.total_amount;
        pendingCount += 1;
      }
    });

    const sortedInvoices = [...invoices].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
    const recent = sortedInvoices.slice(0, 5);
    const unpaidList = invoices.filter(inv => inv.status !== 'PAID');

    return {
      expectedRevenue: expected,
      collectedRevenue: collected,
      unpaidRevenue: unpaid,
      pendingInvoicesCount: pendingCount,
      recentInvoices: recent,
      unpaidInvoices: unpaidList
    };
  }, [invoices]);

  // Calculate total tenants across all houses
  const totalTenants = useMemo(() => {
    return Object.values(tenantsByHouse).reduce((acc, tenantsArray) => acc + (tenantsArray?.length || 0), 0);
  }, [tenantsByHouse]);

  const totalRooms = allRooms.length;
  const occupiedRooms = allRooms.filter(r => r.status === 'OCCUPIED').length;
  const availableRooms = allRooms.filter(r => r.status === 'AVAILABLE');
  const occupancyRate = totalRooms > 0 ? Math.round((occupiedRooms / totalRooms) * 100) : 0;

  // Calculate aggregated profit for current period
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
              <span className="truncate">Đã thu: {formatCurrency(collectedRevenue)}</span>
            </span>
          }
        />
        <StatCard
          label="Hóa đơn chưa thu"
          value={`${pendingInvoicesCount} phiếu`}
          icon={AlertCircle}
          tone="warning"
          sub={`Tổng nợ: ${formatCurrency(unpaidRevenue)}`}
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
            title={`Cần thu tiền (${unpaidInvoices.length})`}
            action={
              <Button variant="ghost" size="sm" onClick={() => setActiveTab('invoices')} className="text-muted-foreground">
                Xem tất cả
                <ArrowRight data-icon="inline-end" />
              </Button>
            }
            bodyClassName="divide-y divide-border/40 max-h-[300px] overflow-y-auto"
          >
            {isLoadingData ? (
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
            {recentInvoices.length === 0 ? (
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
            title={`Phòng đang trống (${availableRooms.length})`}
            className="flex flex-col lg:absolute lg:inset-0 max-h-[600px] lg:max-h-none"
            bodyClassName="divide-y divide-border/40 overflow-y-auto flex-1"
          >
            {isLoadingData ? (
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
                    <p className="text-xs text-muted-foreground bg-secondary px-2 py-0.5 rounded-sm line-clamp-1">{room.houseName}</p>
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
