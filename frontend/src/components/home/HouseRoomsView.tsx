import { memo, useState, useEffect, useCallback } from 'react'
import {
  Plus,
  DoorOpen,
  Pencil,
  DollarSign,
  UserPlus,
  Trash2,
  Building,
  MoreHorizontal,
  LogOut,
  Users,
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
import { refreshTenantDependents } from '@/data/tenantData'
import { StatCard } from '@/components/shared/StatCard'
import { SearchInput } from '@/components/shared/SearchInput'
import { DataPagination } from '@/components/shared/DataPagination'
import { Skeleton } from '@/components/ui/skeleton'
import { useQuery, queryKey } from '@/lib/queryCache'
import { getRoomStats } from '@/api/room'
import { cn } from '@/lib/utils'
import { CreateRoomModal } from './modals/CreateRoomModal'
import { EditRoomModal } from './modals/EditRoomModal'
import { TenantRoomModal } from './modals/TenantRoomModal'
import { ConfirmModal } from './modals/ConfirmModal'
import { QuickSetRoomPriceModal } from './modals/QuickSetRoomPriceModal'
import { CreateHouseModal } from './modals/CreateHouseModal'
import { EditHouseModal } from './modals/EditHouseModal'
import { CheckoutRoomDialog } from './modals/CheckoutRoomDialog'
import type { Room } from '@/api/room'
import type { House } from '@/api/house'

// View danh sách phòng của nhà trọ đang chọn: lưới phòng + thao tác thêm/sửa/xóa/khách thuê, phân trang.
export function HouseRoomsView() {
  const selectedHouse = useSelectedStore(state => state.selectedHouse)
  const {
    rooms, roomTotal, roomPage, roomPageSize, roomSearch, roomsLoading, roomsError, roomsHouseId,
    setRoomPage, setRoomPageSize, setRoomSearch, deleteRoom, fetchRooms,
  } = useRoomStore()
  const { houses, fetchHouses, deleteHouse } = useHouseStore()
  const statsHouseId = selectedHouse?.id ?? ''
  const { data: roomStats } = useQuery(
    queryKey('rooms:stats', { house_id: statsHouseId }),
    () => getRoomStats(statsHouseId),
    { enabled: Boolean(statsHouseId), keepPrevious: false },
  )

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
  const [roomToCheckout, setRoomToCheckout] = useState<Room | null>(null)

  const refreshRoomsAndTenants = refreshTenantDependents

  const openRoom = useCallback((room: Room) => setEditRoom(room), [])
  const addTenant = useCallback((room: Room) => { setTenantRoom(room); setTenantModalView('add') }, [])
  const listTenants = useCallback((room: Room) => { setTenantRoom(room); setTenantModalView('list') }, [])
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    if (houses.length === 0) {
      fetchHouses()
    }
  }, [houses.length, fetchHouses])

  useEffect(() => {
    if (!selectedHouse) return
    if (roomsHouseId && roomsHouseId !== selectedHouse.id && useRoomStore.getState().roomSearch) {
      setRoomSearch('')
      return
    }
    fetchRooms(selectedHouse.id, roomPage)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedHouse, roomPage, roomPageSize, roomSearch, fetchRooms])

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
              onClose={(changed) => {
                setTenantRoom(null)
                if (changed) refreshRoomsAndTenants()
              }}
            />
          )}

          {roomToCheckout && (
            <CheckoutRoomDialog
              room={roomToCheckout}
              onCancel={() => setRoomToCheckout(null)}
              onDone={() => {
                setRoomToCheckout(null)
                refreshRoomsAndTenants()
              }}
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

          <div className="grid grid-cols-[repeat(auto-fit,minmax(140px,1fr))] gap-3">
            <StatCard label="Tổng phòng" value={formatCount(roomStats?.total)} icon={DoorOpen} sub={`Sức chứa: ${formatCount(roomStats?.capacity)} người`} />
            <StatCard label="Đang thuê" value={formatCount(roomStats?.occupied)} icon={Building} tone="positive" sub={roomStats && roomStats.total > 0 ? `Lấp đầy ${Math.round((roomStats.occupied / roomStats.total) * 100)}%` : undefined} />
            <StatCard label="Phòng trống" value={formatCount(roomStats?.available)} icon={DoorOpen} tone="warning" sub={roomStats?.maintenance ? `Bảo trì: ${roomStats.maintenance}` : undefined} />
            <StatCard label="Khách đang ở" value={formatCount(roomStats?.tenants)} icon={Users} tone="info" />
          </div>

          <SearchInput
            label="Tìm phòng theo tên"
            placeholder="Tìm phòng theo tên..."
            value={roomSearch}
            onSearch={setRoomSearch}
            className="max-w-sm"
          />

          {roomsError && (
            <div role="alert" className="flex items-center justify-between gap-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              <span>{roomsError}</span>
              <Button variant="outline" size="sm" onClick={() => useRoomStore.getState().refreshCurrentRooms()}>Thử lại</Button>
            </div>
          )}

          {roomsLoading && rooms.length === 0 ? (
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {Array.from({ length: 6 }, (_, i) => <Skeleton key={i} className="h-24 rounded-xl" />)}
            </div>
          ) : rooms.length === 0 ? (
            <EmptyState
              icon={DoorOpen}
              title={roomSearch ? `Không có phòng nào khớp "${roomSearch}".` : 'Chưa có phòng nào trong nhà trọ này.'}
              className="py-20"
            />
          ) : (
            <>
              <div className={cn('grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 transition-opacity', roomsLoading && 'opacity-60')}>
                {rooms.map(room => (
                  <RoomCard
                    key={room.id}
                    room={room}
                    onOpen={openRoom}
                    onAddTenant={addTenant}
                    onListTenants={listTenants}
                    onCheckout={setRoomToCheckout}
                    onDelete={setRoomToDelete}
                  />
                ))}
              </div>

              <DataPagination
                page={roomPage}
                pageSize={roomPageSize}
                total={roomTotal}
                onPageChange={setRoomPage}
                onPageSizeChange={setRoomPageSize}
                className="border-t border-border/60 pt-4"
              />
            </>
          )}
        </>
      )}
    </div>
  )
}

const formatCount = (value?: number) => (value === undefined ? '—' : value.toLocaleString('vi-VN'))

interface RoomCardProps {
  room: Room
  onOpen: (room: Room) => void
  onAddTenant: (room: Room) => void
  onListTenants: (room: Room) => void
  onCheckout: (room: Room) => void
  onDelete: (room: Room) => void
}

const RoomCard = memo(function RoomCard({ room, onOpen, onAddTenant, onListTenants, onCheckout, onDelete }: RoomCardProps) {
  const tenantCount = room.tenant_count ?? 0
  const canCheckout = tenantCount > 0 || room.status === 'OCCUPIED'
  const stop = (fn: (room: Room) => void) => (e: React.MouseEvent) => { e.stopPropagation(); fn(room) }

  return (
    <Card
      onClick={() => onOpen(room)}
      className="group relative cursor-pointer gap-2 p-3 transition-shadow hover:shadow-md active:bg-muted/40 sm:p-4"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <h3 className="truncate text-base font-semibold text-foreground">{room.name}</h3>
          {room.contract_path && <VerifiedBadge title="Đã tải hợp đồng thuê" />}
          <StatusBadge status={room.status} />
        </div>

        <div className="hidden items-center gap-1 transition-opacity md:flex pointer-fine:opacity-0 pointer-fine:group-hover:opacity-100 pointer-fine:group-focus-within:opacity-100">
          <Button variant="ghost" size="icon-sm" onClick={stop(onAddTenant)} className="text-muted-foreground" aria-label={`Thêm khách thuê phòng ${room.name}`} title="Thêm khách thuê">
            <UserPlus />
          </Button>
          {canCheckout && (
            <Button variant="ghost" size="icon-sm" onClick={stop(onCheckout)} className="text-muted-foreground" aria-label={`Trả phòng ${room.name}`} title="Trả phòng (xoá hết người thuê)">
              <LogOut />
            </Button>
          )}
          <Button variant="ghost" size="icon-sm" onClick={stop(onOpen)} className="text-muted-foreground" aria-label={`Sửa phòng ${room.name}`} title="Sửa thông tin phòng">
            <Pencil />
          </Button>
          <Button variant="ghost" size="icon-sm" onClick={stop(onDelete)} className="text-muted-foreground hover:text-destructive" aria-label={`Xóa phòng ${room.name}`} title="Xóa phòng">
            <Trash2 />
          </Button>
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={e => e.stopPropagation()}
                className="-mr-1 -mt-1 shrink-0 text-muted-foreground md:hidden"
                aria-label={`Thao tác phòng ${room.name}`}
              >
                <MoreHorizontal className="size-4" />
              </Button>
            }
          />
          <DropdownMenuContent side="bottom" align="end">
            <DropdownMenuGroup>
              <DropdownMenuItem onClick={stop(onAddTenant)}>
                <UserPlus /> Thêm khách thuê
              </DropdownMenuItem>
              {canCheckout && (
                <DropdownMenuItem onClick={stop(onCheckout)}>
                  <LogOut /> Trả phòng
                </DropdownMenuItem>
              )}
              <DropdownMenuItem onClick={stop(onOpen)}>
                <Pencil /> Sửa thông tin
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" onClick={stop(onDelete)}>
                <Trash2 /> Xóa phòng
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <div className="grid grid-cols-2 gap-x-3 text-sm">
        <div className="min-w-0">
          <p className="text-xs text-muted-foreground">Giá thuê</p>
          <p className="truncate font-semibold tabular-nums text-foreground">{(room.price || 0).toLocaleString('vi-VN')}đ</p>
        </div>
        <button
          type="button"
          onClick={stop(onListTenants)}
          className="min-w-0 rounded-md text-left underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
          aria-label={`Xem ${tenantCount} khách thuê phòng ${room.name}`}
        >
          <span className="block text-xs text-muted-foreground">Đang ở</span>
          <span className="block truncate font-medium tabular-nums text-foreground">{tenantCount} / {room.max_tenants} người</span>
        </button>
      </div>
    </Card>
  )
})
