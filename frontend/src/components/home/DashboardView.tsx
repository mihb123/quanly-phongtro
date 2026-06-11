import { useState, useEffect, useMemo } from 'react'
import { Plus, Users, Building2, TrendingUp, AlertCircle, Clock, CheckCircle2, Receipt, ArrowRight, DoorOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { useInvoiceStore } from '@/data/invoiceData'
import { useTenantStore } from '@/data/tenantData'
import { useSelectedStore } from '@/data/selectedData'
import { CreateHouseModal } from './modals/CreateHouseModal'
import { type Room } from '@/api/room'

const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
}

export function DashboardView() {
  const [showCreateHouse, setShowCreateHouse] = useState(false)
  const { houses } = useHouseStore()
  const { getRoomsByHouse } = useRoomStore()
  const { tenantsByHouse, fetchTenants } = useTenantStore()
  const { invoices, fetchInvoices } = useInvoiceStore()
  const { setActiveTab, selectHouse } = useSelectedStore()

  const [allRooms, setAllRooms] = useState<(Room & { houseName?: string })[]>([])
  const [isLoadingData, setIsLoadingData] = useState(true)

  // Fetch invoices on mount
  useEffect(() => {
    fetchInvoices()
  }, [fetchInvoices])

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
      
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 border-b border-border/40 pb-6">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-foreground">
            Tổng quan
          </h1>
          <p className="text-muted-foreground mt-2 font-medium">
            Quản lý {houses.length} nhà trọ với tổng cộng {totalRooms} phòng.
          </p>
        </div>
        <div className="shrink-0 flex gap-3">
          <Button 
            onClick={() => setShowCreateHouse(true)} 
            className="rounded-xl px-5 font-bold shadow-sm transition-all active:scale-95 flex items-center gap-2 touch-target"
          >
            <Plus className="w-4 h-4" /> Tạo nhà trọ
          </Button>
        </div>
      </div>

      {/* Balanced 4-Column Stats Grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
        <div className="bg-card border border-border/60 rounded-2xl p-4 sm:p-5 shadow-sm">
          <div className="flex items-center gap-2 sm:gap-3 mb-2 sm:mb-3">
            <div className="w-6 h-6 sm:w-8 sm:h-8 rounded-full bg-blue-100 flex items-center justify-center shrink-0">
              <TrendingUp className="w-3 h-3 sm:w-4 sm:h-4 text-blue-600" />
            </div>
            <h3 className="text-[10px] sm:text-xs font-bold text-muted-foreground uppercase tracking-wider line-clamp-1">Doanh thu dự kiến</h3>
          </div>
          <p className="text-lg sm:text-2xl font-extrabold text-foreground truncate">{formatCurrency(expectedRevenue)}</p>
          <div className="flex items-center gap-1 sm:gap-1.5 mt-1.5 sm:mt-2 text-[10px] sm:text-xs font-medium text-emerald-600">
            <CheckCircle2 className="w-3 h-3 shrink-0" />
            <span className="truncate">Đã thu: {formatCurrency(collectedRevenue)}</span>
          </div>
        </div>

        <div className="bg-card border border-border/60 rounded-2xl p-4 sm:p-5 shadow-sm">
          <div className="flex items-center gap-2 sm:gap-3 mb-2 sm:mb-3">
            <div className="w-6 h-6 sm:w-8 sm:h-8 rounded-full bg-orange-100 flex items-center justify-center shrink-0">
              <AlertCircle className="w-3 h-3 sm:w-4 sm:h-4 text-orange-600" />
            </div>
            <h3 className="text-[10px] sm:text-xs font-bold text-muted-foreground uppercase tracking-wider line-clamp-1">Hóa đơn chưa thu</h3>
          </div>
          <p className="text-lg sm:text-2xl font-extrabold text-foreground truncate">{pendingInvoicesCount} <span className="text-xs sm:text-base text-muted-foreground font-semibold">phiếu</span></p>
          <p className="text-[10px] sm:text-xs font-medium text-orange-600 mt-1.5 sm:mt-2 truncate">
            Tổng nợ: {formatCurrency(unpaidRevenue)}
          </p>
        </div>

        <div className="bg-card border border-border/60 rounded-2xl p-4 sm:p-5 shadow-sm">
          <div className="flex items-center gap-2 sm:gap-3 mb-2 sm:mb-3">
            <div className="w-6 h-6 sm:w-8 sm:h-8 rounded-full bg-emerald-100 flex items-center justify-center shrink-0">
              <Building2 className="w-3 h-3 sm:w-4 sm:h-4 text-emerald-600" />
            </div>
            <h3 className="text-[10px] sm:text-xs font-bold text-muted-foreground uppercase tracking-wider line-clamp-1">Tỷ lệ lấp đầy</h3>
          </div>
          <p className="text-lg sm:text-2xl font-extrabold text-foreground">{occupancyRate}%</p>
          <p className="text-[10px] sm:text-xs font-medium text-muted-foreground mt-1.5 sm:mt-2 truncate">
            Đang thuê: {occupiedRooms} / {totalRooms} phòng
          </p>
        </div>

        <div className="bg-card border border-border/60 rounded-2xl p-4 sm:p-5 shadow-sm">
          <div className="flex items-center gap-2 sm:gap-3 mb-2 sm:mb-3">
            <div className="w-6 h-6 sm:w-8 sm:h-8 rounded-full bg-purple-100 flex items-center justify-center shrink-0">
              <Users className="w-3 h-3 sm:w-4 sm:h-4 text-purple-600" />
            </div>
            <h3 className="text-[10px] sm:text-xs font-bold text-muted-foreground uppercase tracking-wider line-clamp-1">Khách lưu trú</h3>
          </div>
          <p className="text-lg sm:text-2xl font-extrabold text-foreground truncate">{totalTenants} <span className="text-xs sm:text-base text-muted-foreground font-semibold">người</span></p>
          <p className="text-[10px] sm:text-xs font-medium text-muted-foreground mt-1.5 sm:mt-2 line-clamp-1">
            Đang hoạt động trên hệ thống
          </p>
        </div>
      </div>

      {/* Main Content Grid: Lists */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        
        {/* Left Column: Unpaid Invoices & Recent Activities (Span 2) */}
        <div className="lg:col-span-2 flex flex-col gap-6">
          
          {/* Unpaid Invoices */}
          <div className="bg-card border border-border/60 rounded-2xl shadow-sm overflow-hidden">
            <div className="px-5 py-4 border-b border-border/40 flex justify-between items-center">
              <h3 className="font-bold text-foreground flex items-center gap-2">
                <AlertCircle className="w-4 h-4 text-orange-500" /> 
                Cần thu tiền ({unpaidInvoices.length})
              </h3>
              <Button variant="ghost" size="sm" onClick={() => setActiveTab('invoices')} className="text-xs font-semibold text-primary">
                Xem tất cả
              </Button>
            </div>
            <div className="divide-y divide-border/40 max-h-[300px] overflow-y-auto">
              {isLoadingData ? (
                <div className="p-5 text-center text-sm text-muted-foreground">Đang tải dữ liệu...</div>
              ) : unpaidInvoices.length === 0 ? (
                <div className="p-8 text-center text-sm text-muted-foreground italic">Không có hóa đơn nợ.</div>
              ) : (
                unpaidInvoices.map(inv => (
                  <div key={inv.id} className="p-4 hover:bg-secondary/30 transition-colors flex justify-between items-center cursor-pointer" onClick={() => setActiveTab('invoices')}>
                    <div>
                      <p className="font-bold text-sm text-foreground">{inv.room_name}</p>
                      <p className="text-xs text-muted-foreground mt-0.5">Kỳ: {inv.period}</p>
                    </div>
                    <div className="text-right">
                      <p className="font-bold text-sm text-orange-600">{formatCurrency(inv.total_amount)}</p>
                      <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-orange-100 text-orange-700 mt-1 inline-block">
                        {inv.status === 'PENDING_VERIFICATION' ? 'Chờ xác nhận' : 'Chưa thu'}
                      </span>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Recent Invoices */}
          <div className="bg-card border border-border/60 rounded-2xl shadow-sm overflow-hidden">
            <div className="px-5 py-4 border-b border-border/40">
              <h3 className="font-bold text-foreground flex items-center gap-2">
                <Receipt className="w-4 h-4 text-slate-500" /> 
                Hóa đơn gần đây
              </h3>
            </div>
            <div className="divide-y divide-border/40">
              {recentInvoices.length === 0 ? (
                <div className="p-8 text-center text-sm text-muted-foreground italic">Chưa có hóa đơn nào được tạo.</div>
              ) : (
                recentInvoices.map(inv => (
                  <div key={inv.id} className="p-4 flex justify-between items-center">
                    <div className="flex items-center gap-3">
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 ${inv.status === 'PAID' ? 'bg-emerald-100 text-emerald-600' : 'bg-secondary text-muted-foreground'}`}>
                        {inv.status === 'PAID' ? <CheckCircle2 className="w-4 h-4" /> : <Clock className="w-4 h-4" />}
                      </div>
                      <div>
                        <p className="font-semibold text-sm text-foreground">{inv.room_name} <span className="text-muted-foreground font-normal">· Kỳ {inv.period}</span></p>
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {new Date(inv.created_at).toLocaleDateString('vi-VN')}
                        </p>
                      </div>
                    </div>
                    <p className="font-bold text-sm text-foreground">{formatCurrency(inv.total_amount)}</p>
                  </div>
                ))
              )}
            </div>
          </div>

        </div>

        {/* Right Column: Available Rooms (Span 1) */}
        <div className="lg:col-span-1 relative min-h-[400px]">
          <div className="bg-card border border-border/60 rounded-2xl shadow-sm overflow-hidden flex flex-col lg:absolute lg:inset-0 max-h-[600px] lg:max-h-none">
            <div className="px-5 py-4 border-b border-border/40 flex justify-between items-center shrink-0">
              <h3 className="font-bold text-foreground flex items-center gap-2">
                <DoorOpen className="w-4 h-4 text-blue-500" /> 
                Phòng đang trống ({availableRooms.length})
              </h3>
            </div>
            <div className="divide-y divide-border/40 overflow-y-auto flex-1">
              {isLoadingData ? (
                <div className="p-5 text-center text-sm text-muted-foreground">Đang tải...</div>
              ) : availableRooms.length === 0 ? (
                <div className="p-8 text-center text-sm text-muted-foreground italic">Không có phòng trống.</div>
              ) : (
                availableRooms.map(room => (
                  <div 
                    key={room.id} 
                    className="p-4 hover:bg-secondary/30 transition-colors cursor-pointer group"
                    onClick={() => navigateToHouse(room.house_id)}
                  >
                    <div className="flex justify-between items-start mb-1">
                      <p className="font-bold text-sm text-foreground group-hover:text-primary transition-colors">{room.name}</p>
                      <p className="font-semibold text-sm text-foreground">{room.price ? formatCurrency(room.price) : 'Chưa đặt giá'}</p>
                    </div>
                    <div className="flex items-center justify-between mt-2">
                      <p className="text-xs text-muted-foreground bg-secondary px-2 py-1 rounded-md line-clamp-1">{room.houseName}</p>
                      <ArrowRight className="w-3.5 h-3.5 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>

      </div>
    </div>
  )
}
