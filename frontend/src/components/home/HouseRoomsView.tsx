import { useState, useEffect } from 'react'
import { Plus, DoorOpen, Pencil, DollarSign, UserPlus, Trash2, Building } from '@/components/icons'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { PageHeader } from '@/components/shared/PageHeader'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { EmptyState } from '@/components/shared/EmptyState'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'
import { useHouseStore } from '@/data/houseData'
import { useTenantStore } from '@/data/tenantData'
import { CreateRoomModal } from './modals/CreateRoomModal'
import { EditRoomModal } from './modals/EditRoomModal'
import { TenantRoomModal } from './modals/TenantRoomModal'
import { ConfirmModal } from './modals/ConfirmModal'
import { QuickSetRoomPriceModal } from './modals/QuickSetRoomPriceModal'
import { CreateHouseModal } from './modals/CreateHouseModal'
import type { Room } from '@/api/room'

// View danh sách phòng của nhà trọ đang chọn: lưới phòng + thao tác thêm/sửa/xóa/khách thuê, phân trang.
export function HouseRoomsView() {
  const selectedHouse = useSelectedStore(state => state.selectedHouse)
  const { rooms, roomPage, setRoomPage, deleteRoom, fetchRooms } = useRoomStore()
  const { tenantsByHouse, fetchTenants } = useTenantStore()

  const limit = 25

  const [showCreateHouse, setShowCreateHouse] = useState(false)
  const [showCreateRoom, setShowCreateRoom] = useState(false)
  const [editRoom, setEditRoom] = useState<Room | null>(null)

  const [tenantRoom, setTenantRoom] = useState<Room | null>(null)
  const [tenantModalView, setTenantModalView] = useState<'list' | 'add'>('list')

  const [showQuickSetPrice, setShowQuickSetPrice] = useState(false)
  const [roomToDelete, setRoomToDelete] = useState<Room | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    if (selectedHouse) {
      fetchRooms(selectedHouse.id, roomPage)
      fetchTenants(selectedHouse.id)
    }
  }, [selectedHouse, roomPage, fetchRooms, fetchTenants])

  if (!selectedHouse) {
    return (
      <div className="space-y-6 safe-fade-in">
        {showCreateHouse && <CreateHouseModal onClose={() => setShowCreateHouse(false)} />}
        <PageHeader
          title="Chọn nhà trọ"
          description="Vui lòng chọn một nhà trọ để xem danh sách phòng"
          className="border-b border-border pb-6"
          action={
            <Button onClick={() => setShowCreateHouse(true)} className="font-bold">
              <Plus className="size-4" /> Tạo nhà
            </Button>
          }
        />

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {useHouseStore.getState().houses.map(house => (
            <Card
              key={house.id}
              onClick={() => {
                useSelectedStore.getState().selectHouse(house)
                setRoomPage(1)
              }}
              className="cursor-pointer p-6 transition-all hover:shadow-md group"
            >
              <div className="flex items-center gap-4">
                <div className="flex size-12 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary transition-transform group-hover:scale-110">
                  <Building className="size-6" />
                </div>
                <div className="min-w-0">
                  <h3 className="font-bold text-lg text-foreground group-hover:text-primary transition-colors truncate">{house.name}</h3>
                  <p className="text-sm text-muted-foreground truncate">{house.address || 'Chưa cập nhật địa chỉ'}</p>
                </div>
              </div>
            </Card>
          ))}
          {useHouseStore.getState().houses.length === 0 && (
            <div className="col-span-full">
              <EmptyState icon={Building} title="Chưa có nhà trọ nào. Vui lòng tạo nhà trọ trước." />
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6 safe-fade-in">
      {/* Modals */}
      {showCreateRoom && (
        <CreateRoomModal onClose={() => setShowCreateRoom(false)} />
      )}
      {editRoom && (
        <EditRoomModal room={editRoom} onClose={() => setEditRoom(null)} />
      )}

      {tenantRoom && (
        <TenantRoomModal
          room={tenantRoom}
          initialView={tenantModalView}
          onClose={() => setTenantRoom(null)}
        />
      )}

      {showQuickSetPrice && (
        <QuickSetRoomPriceModal onClose={() => setShowQuickSetPrice(false)} />
      )}
      {roomToDelete && (
        <ConfirmModal
          title={`Xóa phòng: ${roomToDelete.name}`}
          message="Bạn có chắc chắn muốn xóa phòng trọ này không? Dữ liệu không thể khôi phục."
          confirmText="Xóa ngay"
          cancelText="Bỏ qua"
          isLoading={isDeleting}
          onCancel={() => setRoomToDelete(null)}
          onConfirm={async () => {
            setIsDeleting(true)
            const res = await deleteRoom(roomToDelete.id, roomToDelete.house_id)
            if (!res.success) {
              alert(res.error || "Lỗi khi xóa phòng trọ!")
            }
            setIsDeleting(false)
            setRoomToDelete(null)
            useRoomStore.getState().refreshCurrentRooms()
          }}
        />
      )}

      <PageHeader
        className="border-b border-border pb-6"
        title={
          <span className="flex items-center gap-3">
            <span className="line-clamp-2">{selectedHouse.name}</span>
            <Button variant="outline" size="sm" onClick={() => useSelectedStore.getState().selectHouse(null)} className="font-semibold md:hidden touch-target">
              Đổi nhà
            </Button>
          </span>
        }
        description={selectedHouse.address}
        action={
          <>
            <Button onClick={() => setShowQuickSetPrice(true)} variant="outline" className="flex-1 font-bold touch-target md:flex-none">
              <DollarSign className="size-4" /> Set giá nhanh
            </Button>
            <Button onClick={() => setShowCreateRoom(true)} className="flex-1 font-bold touch-target md:flex-none">
              <Plus className="size-4" /> Thêm phòng
            </Button>
          </>
        }
      />

      {rooms.length === 0 && roomPage === 1 ? (
        <EmptyState icon={DoorOpen} title="Chưa có phòng nào trong nhà trọ này." className="py-20" />
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {[...rooms].sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })).map((room, index) => (
              <Card key={room.id} onClick={() => setEditRoom(room)} className="group relative cursor-pointer p-6 transition-shadow hover:shadow-md safe-fade-in slide-in-from-bottom-2 fill-mode-both" style={{ animationDelay: `${index * 50}ms` }}>
                <div className="flex justify-between items-start">
                  <div className="flex items-center gap-3">
                    <div className="flex size-10 items-center justify-center rounded-xl bg-primary/10 text-primary transition-transform group-hover:scale-110">
                      <DoorOpen className="size-5" />
                    </div>
                    <div className="space-y-1">
                      <h3 className="font-bold text-lg text-foreground">{room.name}</h3>
                      <StatusBadge status={room.status} />
                    </div>
                  </div>
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <Button variant="ghost" size="icon-sm" onClick={(e) => { e.stopPropagation(); setTenantRoom(room); setTenantModalView('add') }} className="rounded-full text-muted-foreground hover:bg-success/10 hover:text-success" title="Thêm khách thuê">
                      <UserPlus className="size-4" />
                    </Button>
                    <Button variant="ghost" size="icon-sm" onClick={(e) => { e.stopPropagation(); setEditRoom(room) }} className="rounded-full text-muted-foreground hover:bg-primary/10 hover:text-primary" title="Sửa thông tin phòng">
                      <Pencil className="size-4" />
                    </Button>
                    <Button variant="ghost" size="icon-sm" onClick={(e) => { e.stopPropagation(); setRoomToDelete(room) }} className="rounded-full text-muted-foreground hover:bg-destructive/10 hover:text-destructive" title="Xóa phòng">
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                </div>
                <div className="space-y-2 text-sm text-muted-foreground">
                  <div className="flex justify-between items-center rounded-md bg-secondary/50 p-2">
                    <span>Giá thuê:</span>
                    <span className="font-semibold text-foreground text-base">{(room.price || 0).toLocaleString()}đ</span>
                  </div>
                  <div className="flex justify-between items-center p-2">
                    <span>Số người đang ở:</span>
                    <span
                      className="font-semibold text-foreground cursor-pointer hover:text-primary hover:underline transition-colors"
                      title="Xem danh sách khách thuê"
                      onClick={(e) => { e.stopPropagation(); setTenantRoom(room); setTenantModalView('list') }}
                    >
                      {tenantsByHouse[selectedHouse.id]?.filter(t => t.room_id === room.id).length || 0} / {room.max_tenants} người
                    </span>
                  </div>
                </div>
              </Card>
            ))}
          </div>

          {/* Pagination Controls */}
          {(roomPage > 1 || rooms.length === limit) && (
            <div className="flex items-center justify-center gap-4 mt-8 pt-6 border-t border-border/60">
              <Button
                variant="outline"
                size="sm"
                disabled={roomPage === 1}
                onClick={() => setRoomPage(roomPage - 1)}
              >
                Trang trước
              </Button>
              <span className="text-sm font-bold text-muted-foreground">Trang {roomPage}</span>
              <Button
                variant="outline"
                size="sm"
                disabled={rooms.length < limit}
                onClick={() => setRoomPage(roomPage + 1)}
              >
                Trang sau
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
