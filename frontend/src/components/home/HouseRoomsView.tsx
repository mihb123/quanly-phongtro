import { useState, useEffect } from 'react'
import {
  Plus,
  DoorOpen,
  Pencil,
  DollarSign,
  UserPlus,
  Trash2,
  Building,
  MoreHorizontal,
} from '@/components/icons'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { PageHeader } from '@/components/shared/PageHeader'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { VerifiedBadge } from '@/components/shared/VerifiedBadge'
import { EmptyState } from '@/components/shared/EmptyState'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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
import { EditHouseModal } from './modals/EditHouseModal'
import type { Room } from '@/api/room'
import type { House } from '@/api/house'

// View danh sách phòng của nhà trọ đang chọn: lưới phòng + thao tác thêm/sửa/xóa/khách thuê, phân trang.
export function HouseRoomsView() {
  const selectedHouse = useSelectedStore(state => state.selectedHouse)
  const { rooms, roomPage, setRoomPage, deleteRoom, fetchRooms } = useRoomStore()
  const { tenantsByHouse, fetchTenants } = useTenantStore()
  const { houses, fetchHouses, deleteHouse } = useHouseStore()

  const limit = 25

  const [showCreateHouse, setShowCreateHouse] = useState(false)
  const [houseToEdit, setHouseToEdit] = useState<House | null>(null)
  const [houseToDelete, setHouseToDelete] = useState<House | null>(null)
  const [isDeletingHouse, setIsDeletingHouse] = useState(false)

  const [showCreateRoom, setShowCreateRoom] = useState(false)
  const [editRoom, setEditRoom] = useState<Room | null>(null)

  const [tenantRoom, setTenantRoom] = useState<Room | null>(null)
  const [tenantModalView, setTenantModalView] = useState<'list' | 'add'>('list')

  const [showQuickSetPrice, setShowQuickSetPrice] = useState(false)
  const [roomToDelete, setRoomToDelete] = useState<Room | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    if (houses.length === 0) {
      fetchHouses()
    }
  }, [houses.length, fetchHouses])

  useEffect(() => {
    if (selectedHouse) {
      fetchRooms(selectedHouse.id, roomPage)
      fetchTenants(selectedHouse.id)
    }
  }, [selectedHouse, roomPage, fetchRooms, fetchTenants])

  return (
    <div className="space-y-6 safe-fade-in">
      {/* House Modals */}
      {showCreateHouse && (
        <CreateHouseModal onClose={() => setShowCreateHouse(false)} />
      )}
      {houseToEdit && (
        <EditHouseModal house={houseToEdit} onClose={() => setHouseToEdit(null)} />
      )}
      {houseToDelete && (
        <ConfirmModal
          title={`Xóa nhà: ${houseToDelete.name}`}
          message="Toàn bộ phòng, khách thuê, hóa đơn, thanh toán và chi phí của nhà này sẽ bị xóa vĩnh viễn. Không có cách khôi phục."
          confirmPhrase={houseToDelete.name}
          confirmText="Xóa ngay"
          cancelText="Bỏ qua"
          isLoading={isDeletingHouse}
          onCancel={() => setHouseToDelete(null)}
          onConfirm={async () => {
            setIsDeletingHouse(true)
            const res = await deleteHouse(houseToDelete.id)
            if (res.success) {
              await fetchHouses()
              if (selectedHouse?.id === houseToDelete.id) {
                useSelectedStore.getState().selectHouse(null)
              }
            } else {
              alert(res.error || "Lỗi khi xóa nhà trọ, vui lòng thử lại!")
            }
            setIsDeletingHouse(false)
            setHouseToDelete(null)
          }}
        />
      )}

      {/* Screen 1: No house selected */}
      {!selectedHouse ? (
        <>
          <PageHeader
            title="Chọn nhà trọ"
            description="Vui lòng chọn một nhà trọ để xem danh sách phòng"
            className="border-b border-border pb-6"
            action={
              <Button onClick={() => setShowCreateHouse(true)}>
                <Plus data-icon="inline-start" /> Tạo nhà
              </Button>
            }
          />

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {houses.map(house => (
              <Card
                key={house.id}
                onClick={() => {
                  useSelectedStore.getState().selectHouse(house)
                  setRoomPage(1)
                }}
                className="cursor-pointer p-6 transition-all hover:shadow-md group"
              >
                <div className="flex items-center justify-between gap-4">
                  <div className="flex items-center gap-4 min-w-0 flex-1">
                    <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                      <Building className="size-5" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <h3 className="truncate text-base font-semibold text-foreground">{house.name}</h3>
                      <p className="truncate text-sm text-muted-foreground">{house.address || 'Chưa cập nhật địa chỉ'}</p>
                    </div>
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={(e) => e.stopPropagation()}
                          className="text-muted-foreground hover:text-foreground shrink-0"
                          title="Tùy chọn nhà trọ"
                        >
                          <MoreHorizontal className="size-4" />
                        </Button>
                      }
                    />
                    <DropdownMenuContent side="bottom" align="end">
                      <DropdownMenuGroup>
                        <DropdownMenuItem onClick={(e) => { e.stopPropagation(); setHouseToEdit(house) }}>
                          <Pencil />
                          Sửa thông tin
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          variant="destructive"
                          onClick={(e) => { e.stopPropagation(); setHouseToDelete(house) }}
                        >
                          <Trash2 />
                          Xóa nhà trọ
                        </DropdownMenuItem>
                      </DropdownMenuGroup>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </Card>
            ))}
            {houses.length === 0 && (
              <div className="col-span-full">
                <EmptyState icon={Building} title="Chưa có nhà trọ nào. Vui lòng tạo nhà trọ trước." />
              </div>
            )}
          </div>
        </>
      ) : (
        /* Screen 2: House selected */
        <>
          {/* Room Modals */}
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
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <Button variant="ghost" size="icon-sm" className="text-muted-foreground hover:text-foreground shrink-0" title="Tùy chọn nhà trọ">
                        <MoreHorizontal className="size-4" />
                      </Button>
                    }
                  />
                  <DropdownMenuContent side="bottom" align="start">
                    <DropdownMenuGroup>
                      <DropdownMenuItem onClick={() => setHouseToEdit(selectedHouse)}>
                        <Pencil />
                        Sửa thông tin
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        variant="destructive"
                        onClick={() => setHouseToDelete(selectedHouse)}
                      >
                        <Trash2 />
                        Xóa nhà trọ
                      </DropdownMenuItem>
                    </DropdownMenuGroup>
                  </DropdownMenuContent>
                </DropdownMenu>
                <Button variant="outline" size="sm" onClick={() => useSelectedStore.getState().selectHouse(null)} className="md:hidden">
                  Đổi nhà
                </Button>
              </span>
            }
            description={selectedHouse.address}
            action={
              <>
                <Button onClick={() => setHouseToEdit(selectedHouse)} variant="outline" className="hidden sm:inline-flex">
                  <Pencil data-icon="inline-start" /> Sửa thông tin
                </Button>
                <Button onClick={() => setShowQuickSetPrice(true)} variant="outline" className="flex-1 md:flex-none">
                  <DollarSign data-icon="inline-start" /> Set giá nhanh
                </Button>
                <Button onClick={() => setShowCreateRoom(true)} className="flex-1 md:flex-none">
                  <Plus data-icon="inline-start" /> Thêm phòng
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
                  <Card key={room.id} onClick={() => setEditRoom(room)} className="group relative cursor-pointer p-5 transition-shadow hover:shadow-md safe-fade-in fill-mode-both" style={{ animationDelay: `${index * 30}ms` }}>
                    <div className="flex justify-between items-start">
                      <div className="flex items-center gap-3">
                        <div className="flex size-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                          <DoorOpen className="size-4" />
                        </div>
                        <div className="space-y-1">
                          <div className="flex items-center gap-1.5">
                            <h3 className="text-base font-semibold text-foreground">{room.name}</h3>
                            {room.contract_path && <VerifiedBadge title="Đã tải hợp đồng thuê" />}
                          </div>
                          <StatusBadge status={room.status} />
                        </div>
                      </div>
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button variant="ghost" size="icon-sm" onClick={(e) => { e.stopPropagation(); setTenantRoom(room); setTenantModalView('add') }} className="text-muted-foreground" title="Thêm khách thuê">
                          <UserPlus />
                        </Button>
                        <Button variant="ghost" size="icon-sm" onClick={(e) => { e.stopPropagation(); setEditRoom(room) }} className="text-muted-foreground" title="Sửa thông tin phòng">
                          <Pencil />
                        </Button>
                        <Button variant="ghost" size="icon-sm" onClick={(e) => { e.stopPropagation(); setRoomToDelete(room) }} className="text-muted-foreground hover:text-destructive" title="Xóa phòng">
                          <Trash2 />
                        </Button>
                      </div>
                    </div>
                    <div className="space-y-1 text-sm text-muted-foreground">
                      <div className="flex justify-between items-center rounded-md bg-secondary/50 px-2 py-1.5">
                        <span>Giá thuê:</span>
                        <span className="font-semibold tabular-nums text-foreground">{(room.price || 0).toLocaleString()}đ</span>
                      </div>
                      <div className="flex justify-between items-center px-2 py-1.5">
                        <span>Số người đang ở:</span>
                        <span
                          className="font-medium tabular-nums text-foreground cursor-pointer hover:underline"
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
                  <span className="text-sm font-medium tabular-nums text-muted-foreground">Trang {roomPage}</span>
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
        </>
      )}
    </div>
  )
}
