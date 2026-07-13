import { useState, useEffect } from 'react'
import { Building, Plus, CheckCircle2, Users } from '@/components/icons'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { PageHeader } from '@/components/shared/PageHeader'
import { EmptyState } from '@/components/shared/EmptyState'
import { useHouseStore } from '@/data/houseData'
import { useTenantStore } from '@/data/tenantData'
import { TenantRoomModal } from '@/components/home/modals/TenantRoomModal'
import { SelectRoomModal } from '@/components/home/modals/SelectRoomModal'
import { type House } from '@/api/house'
import { type Room } from '@/api/room'
import { type Tenant } from '@/api/tenant'
import { useRoomStore } from '@/data/roomData'
import { toast } from 'sonner'

// Badge "Đang ở" cho khách thuê — dùng token success thay cho màu hardcode.
function StayingBadge() {
  return (
    <Badge variant="ghost" className="bg-success/10 text-success font-bold uppercase tracking-wider">
      <CheckCircle2 className="size-3" /> Đang ở
    </Badge>
  )
}

// Bảng khách thuê của 1 nhà trọ: card mobile + table desktop, mở modal thêm/sửa/xem phòng.
function HouseTenantTable({ house }: { house: House }) {
  const { tenantsByHouse, loadingByHouse, fetchTenants } = useTenantStore()
  const getRoomsByHouse = useRoomStore(state => state.getRoomsByHouse)

  const tenants = tenantsByHouse[house.id] || []
  const loading = loadingByHouse[house.id] ?? true

  const [isSelectRoomModalOpen, setIsSelectRoomModalOpen] = useState(false)
  const [selectedRoom, setSelectedRoom] = useState<Room | null>(null)
  const [modalView, setModalView] = useState<'list' | 'add' | 'edit'>('list')
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null)
  const [isFetchingRoom, setIsFetchingRoom] = useState(false)

  useEffect(() => {
    fetchTenants(house.id)
  }, [fetchTenants, house.id])

  const handleRoomClick = async (roomId: string) => {
    if (!roomId) return;
    setIsFetchingRoom(true)
    try {
      const rooms = await getRoomsByHouse(house.id)
      const room = rooms.find(r => r.id === roomId)
      if (room) {
        setSelectedRoom(room)
        setModalView('list')
        setEditingTenant(null)
      }
    } catch (err) {
      console.error(err)
    } finally {
      setIsFetchingRoom(false)
    }
  }

  const handleTenantClick = async (tenant: Tenant) => {
    if (!tenant.room_id) return;
    setIsFetchingRoom(true)
    try {
      const rooms = await getRoomsByHouse(house.id)
      const room = rooms.find(r => r.id === tenant.room_id)
      if (room) {
        setSelectedRoom(room)
        setEditingTenant(tenant)
        setModalView('edit')
      }
    } catch (err) {
      console.error(err)
    } finally {
      setIsFetchingRoom(false)
    }
  }

  const handlePhoneClick = (phone: string) => {
    navigator.clipboard.writeText(phone).then(() => {
      toast.success('Sao chép thành công', { description: phone })
    })
  }

  return (
    <Card className="relative mb-6 gap-4 p-6">
      {isFetchingRoom && (
        <div className="absolute inset-0 z-10 flex items-center justify-center rounded-4xl bg-background/50 backdrop-blur-sm">
          <div className="text-muted-foreground font-medium">Đang tải thông tin phòng...</div>
        </div>
      )}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h2 className="flex items-center gap-2 text-xl font-bold text-foreground">
          <Building className="size-5 text-primary" />
          {house.name}
        </h2>
        <Button onClick={() => setIsSelectRoomModalOpen(true)} className="font-bold shadow-sm">
          <Plus className="size-4" /> Thêm khách thuê
        </Button>
      </div>

      <div className="overflow-hidden rounded-2xl border border-border/60 bg-card">
        {/* Mobile Card View */}
        <div className="block md:hidden divide-y divide-border/40">
          {loading ? (
            <div className="p-8 text-center text-muted-foreground italic">Đang tải dữ liệu...</div>
          ) : tenants.length === 0 ? (
            <EmptyState icon={Users} title="Chưa có khách thuê nào trong nhà này." />
          ) : (
            tenants.map(t => (
              <div key={t.id} className="p-4 hover:bg-secondary/40 transition-colors">
                <div className="flex justify-between items-start mb-2">
                  <div>
                    <p
                      onClick={() => handleTenantClick(t)}
                      className="font-bold text-primary text-base cursor-pointer hover:underline"
                    >
                      {t.full_name}
                    </p>
                    <p className="text-sm font-bold text-foreground mt-0.5">
                      Phòng: <span onClick={() => handleRoomClick(t.room_id)} className="cursor-pointer hover:text-primary hover:underline">{t.room_name || 'N/A'}</span>
                    </p>
                  </div>
                  <StayingBadge />
                </div>
                <div className="flex justify-between items-center mt-3 pt-3 border-t border-border/40">
                  <span className="text-xs font-medium text-muted-foreground">
                    Từ: {new Date(t.start_date).toLocaleDateString('vi-VN')}
                  </span>
                  <span
                    onClick={() => handlePhoneClick(t.phone)}
                    className="text-sm font-medium text-muted-foreground cursor-pointer hover:text-primary transition-colors touch-target px-2 py-1 -mr-2"
                  >
                    {t.phone}
                  </span>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Desktop Table View */}
        <div className="hidden md:block overflow-x-auto">
          <Table className="whitespace-nowrap">
            <TableHeader>
              <TableRow className="bg-secondary/30">
                <TableHead>Tên khách thuê</TableHead>
                <TableHead>Phòng đang ở</TableHead>
                <TableHead>Số điện thoại</TableHead>
                <TableHead>Ngày bắt đầu ở</TableHead>
                <TableHead>Tình trạng</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={5} className="py-8 text-center text-muted-foreground italic">
                    Đang tải dữ liệu...
                  </TableCell>
                </TableRow>
              ) : tenants.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="py-8 text-center text-muted-foreground italic">
                    Chưa có khách thuê nào trong nhà này.
                  </TableCell>
                </TableRow>
              ) : (
                tenants.map(t => (
                  <TableRow key={t.id}>
                    <TableCell className="font-semibold text-foreground">
                      <span
                        onClick={() => handleTenantClick(t)}
                        className="cursor-pointer hover:text-primary hover:underline transition-colors"
                        title="Sửa người thuê"
                      >
                        {t.full_name}
                      </span>
                    </TableCell>
                    <TableCell className="font-medium text-foreground">
                      <span
                        onClick={() => handleRoomClick(t.room_id)}
                        className="cursor-pointer hover:text-primary hover:underline transition-colors"
                        title="Xem danh sách người thuê phòng"
                      >
                        {t.room_name || 'N/A'}
                      </span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      <span
                        onClick={() => handlePhoneClick(t.phone)}
                        className="cursor-pointer hover:text-primary hover:underline transition-colors"
                        title="Sao chép số điện thoại"
                      >
                        {t.phone}
                      </span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">{new Date(t.start_date).toLocaleDateString('vi-VN')}</TableCell>
                    <TableCell><StayingBadge /></TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      {isSelectRoomModalOpen && (
        <SelectRoomModal
          houseId={house.id}
          tenants={tenants}
          onSelect={(r) => {
            setSelectedRoom(r)
            setModalView('add')
            setIsSelectRoomModalOpen(false)
          }}
          onClose={() => setIsSelectRoomModalOpen(false)}
        />
      )}

      {selectedRoom && (
        <TenantRoomModal
          room={selectedRoom}
          initialView={modalView}
          initialEditingTenant={editingTenant}
          onClose={(changed?: boolean) => {
            setSelectedRoom(null)
            setEditingTenant(null)
            if (changed) fetchTenants(house.id) // refresh list after closing modal if data changed
          }}
        />
      )}
    </Card>
  )
}

// View danh sách khách thuê — nhóm theo từng nhà trọ.
export function TenantsView() {
  const { houses } = useHouseStore()

  return (
    <div className="space-y-8 safe-fade-in">
      <PageHeader
        title="Danh sách Khách thuê"
        description="Thông tin khách thuê được nhóm theo từng nhà trọ."
      />

      {houses.length === 0 ? (
        <Card className="p-8">
          <EmptyState icon={Building} title="Bạn chưa có nhà trọ nào để quản lý khách thuê." />
        </Card>
      ) : (
        houses.map(house => (
          <HouseTenantTable key={house.id} house={house} />
        ))
      )}
    </div>
  )
}
