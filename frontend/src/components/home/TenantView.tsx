import { useState, useEffect } from 'react'
import { Building, Plus, Users } from '@/components/icons'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { PageHeader } from '@/components/shared/PageHeader'
import { EmptyState } from '@/components/shared/EmptyState'
import { useHouseStore } from '@/data/houseData'
import { useTenantStore } from '@/data/tenantData'
import { TenantRoomModal } from '@/components/home/modals/TenantRoomModal'
import { SelectRoomModal } from '@/components/home/modals/SelectRoomModal'
import { VerifiedBadge } from '@/components/shared/VerifiedBadge'
import { type House } from '@/api/house'
import { type Room } from '@/api/room'
import { type Tenant } from '@/api/tenant'
import { useRoomStore } from '@/data/roomData'
import { toast } from 'sonner'
import { StayingBadge, TenantMobileCard } from './tenant/TenantMobileCard'

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
    const normalizedPhone = phone.trim()
    if (!normalizedPhone) return

    navigator.clipboard.writeText(normalizedPhone).then(() => {
      toast.success('Sao chép thành công', { description: normalizedPhone })
    })
  }

  return (
    <Card className="relative gap-3 p-3 sm:gap-4 sm:p-6">
      {isFetchingRoom && (
        <div className="absolute inset-0 z-10 flex items-center justify-center rounded-xl bg-background/50 backdrop-blur-sm">
          <div className="text-muted-foreground font-medium">Đang tải thông tin phòng...</div>
        </div>
      )}
      <div className="flex items-center justify-between gap-3">
        <h2 className="flex min-w-0 items-center gap-2 text-base font-semibold text-foreground">
          <Building className="size-4 text-muted-foreground" />
          <span className="truncate">{house.name}</span>
        </h2>
        <Button size="lg" onClick={() => setIsSelectRoomModalOpen(true)} className="cursor-pointer">
          <Plus data-icon="inline-start" /> Thêm khách thuê
        </Button>
      </div>

      <div className="md:overflow-hidden md:rounded-lg md:border md:bg-card">
        {/* Mobile Card View */}
        <div className="block md:hidden">
          {loading ? (
            <div className="p-8 text-center text-muted-foreground italic">Đang tải dữ liệu...</div>
          ) : tenants.length === 0 ? (
            <EmptyState icon={Users} title="Chưa có khách thuê nào trong nhà này." />
          ) : (
            <ul className="flex flex-col gap-2">
              {tenants.map(tenant => (
                <TenantMobileCard
                  key={tenant.id}
                  tenant={tenant}
                  onEdit={() => handleTenantClick(tenant)}
                  onOpenRoom={() => handleRoomClick(tenant.room_id)}
                  onCopyPhone={() => handlePhoneClick(tenant.phone)}
                />
              ))}
            </ul>
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
                      <div className="flex items-center gap-1.5">
                        <span
                          onClick={() => handleTenantClick(t)}
                          className="cursor-pointer underline-offset-4 hover:underline"
                          title="Sửa người thuê"
                        >
                          {t.full_name}
                        </span>
                        {t.cccd_path && <VerifiedBadge title="Đã tải ảnh CCCD" />}
                      </div>
                    </TableCell>
                    <TableCell className="font-medium text-foreground">
                      <span
                        onClick={() => handleRoomClick(t.room_id)}
                        className="cursor-pointer underline-offset-4 hover:underline"
                        title="Xem danh sách người thuê phòng"
                      >
                        {t.room_name || 'N/A'}
                      </span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {t.phone?.trim() ? (
                        <span
                          onClick={() => handlePhoneClick(t.phone)}
                          className="cursor-pointer underline-offset-4 hover:underline"
                          title="Sao chép số điện thoại"
                        >
                          {t.phone.trim()}
                        </span>
                      ) : (
                        <span className="italic">Chưa cập nhật</span>
                      )}
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
    <div className="flex flex-col gap-6 safe-fade-in">
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
