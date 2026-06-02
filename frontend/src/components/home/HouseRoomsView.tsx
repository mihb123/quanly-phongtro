import { useState, useEffect } from 'react'
import { Plus, DoorOpen, Pencil, DollarSign, UserPlus, Trash2, Building } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
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
        <div className="flex justify-between items-center border-b border-border pb-6 gap-4">
          <div className="flex flex-col gap-2">
            <h1 className="text-2xl md:text-3xl font-extrabold tracking-tight text-foreground">Chọn nhà trọ</h1>
            <p className="text-sm md:text-base text-muted-foreground">Vui lòng chọn một nhà trọ để xem danh sách phòng</p>
          </div>
          <Button onClick={() => setShowCreateHouse(true)} className="rounded-xl px-4 font-bold flex items-center gap-2 shrink-0">
             <Plus className="w-4 h-4" /> Tạo nhà
          </Button>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {useHouseStore.getState().houses.map(house => (
            <Card 
              key={house.id}
              onClick={() => {
                useSelectedStore.getState().selectHouse(house)
                setRoomPage(1)
              }}
              className="p-6 bg-card border-border/40 shadow-sm hover:shadow-md transition-all cursor-pointer hover:border-primary/50 group"
            >
              <div className="flex items-center gap-4">
                <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center text-primary group-hover:scale-110 transition-transform shrink-0">
                  <Building className="w-6 h-6" />
                </div>
                <div className="min-w-0">
                  <h3 className="font-bold text-lg text-foreground group-hover:text-primary transition-colors truncate">{house.name}</h3>
                  <p className="text-sm text-muted-foreground truncate">{house.address || 'Chưa cập nhật địa chỉ'}</p>
                </div>
              </div>
            </Card>
          ))}
          {useHouseStore.getState().houses.length === 0 && (
            <div className="col-span-full text-center py-12 text-muted-foreground italic">
              Chưa có nhà trọ nào. Vui lòng tạo nhà trọ trước.
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
            await deleteRoom(roomToDelete.id, roomToDelete.house_id)
            setIsDeleting(false)
            setRoomToDelete(null)
            useRoomStore.getState().refreshCurrentRooms()
          }}
        />
      )}

      <div className="flex flex-col md:flex-row md:items-center justify-between border-b border-border pb-6 gap-4">
        <div className="flex-1">
          <div className="flex items-center justify-between md:justify-start gap-3">
            <h1 className="text-2xl md:text-3xl font-extrabold tracking-tight text-foreground line-clamp-2">{selectedHouse.name}</h1>
            <Button variant="outline" size="sm" onClick={() => useSelectedStore.getState().selectHouse(null)} className="h-8 px-3 rounded-md text-xs font-semibold md:hidden whitespace-nowrap shrink-0 border-primary/20 text-primary hover:bg-primary/10 touch-target">
              Đổi nhà
            </Button>
          </div>
          <p className="text-muted-foreground mt-1 line-clamp-2">{selectedHouse.address}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2 md:gap-3 shrink-0">
           <Button onClick={() => setShowQuickSetPrice(true)} variant="outline" className="flex-1 md:flex-none border-border text-foreground hover:bg-secondary rounded-xl cursor-pointer px-4 font-bold transition-all active:scale-95 flex justify-center items-center gap-2 touch-target">
             <DollarSign className="w-4 h-4" /> Set giá nhanh
           </Button>
           <Button onClick={() => setShowCreateRoom(true)} className="flex-1 md:flex-none rounded-xl cursor-pointer px-4 md:px-6 font-bold transition-all active:scale-95 flex justify-center items-center gap-2 touch-target">
             <Plus className="w-4 h-4" /> Thêm phòng
           </Button>
        </div>
      </div>
      
      {rooms.length === 0 && roomPage === 1 ? (
        <div className="text-center py-20">
          <p className="text-slate-500 italic">Chưa có phòng nào trong nhà trọ này.</p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {[...rooms].sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })).map((room, index) => (
              <Card key={room.id} onClick={() => setEditRoom(room)} className="p-6 bg-card border-border/40 shadow-sm hover:shadow-md transition-shadow group relative cursor-pointer safe-fade-in slide-in-from-bottom-2 fill-mode-both" style={{ animationDelay: `${index * 50}ms` }}>
                <div className="flex justify-between items-start mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary group-hover:scale-110 transition-transform">
                      <DoorOpen className="w-5 h-5" />
                    </div>
                    <div>
                      <h3 className="font-bold text-lg text-foreground">{room.name}</h3>
                      <span className="text-xs font-semibold px-2 py-1 rounded-full bg-secondary text-secondary-foreground border border-border mt-1 inline-block">
                        {room.status === 'AVAILABLE' ? 'Trống' : room.status === 'OCCUPIED' ? 'Đã Thuê' : room.status}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center opacity-0 group-hover:opacity-100 transition-opacity gap-1">
                    <Button variant="ghost" size="icon" onClick={(e) => { e.stopPropagation(); setTenantRoom(room); setTenantModalView('add') }} className="h-8 w-8 text-muted-foreground hover:text-green-600 hover:bg-green-50/50 rounded-full transition-colors cursor-pointer" title="Thêm khách thuê">
                      <UserPlus className="w-4 h-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={(e) => { e.stopPropagation(); setEditRoom(room) }} className="h-8 w-8 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-full transition-colors cursor-pointer" title="Sửa thông tin phòng">
                      <Pencil className="w-4 h-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={(e) => { e.stopPropagation(); setRoomToDelete(room) }} className="h-8 w-8 text-muted-foreground hover:text-red-600 hover:bg-red-50/50 rounded-full transition-colors cursor-pointer" title="Xóa phòng">
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
                <div className="space-y-2 mt-5 text-sm text-muted-foreground">
                  <div className="flex justify-between items-center bg-secondary/50 p-2 rounded-md">
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
            <div className="flex items-center justify-center gap-4 mt-8 pt-6 border-t border-slate-100">
              <Button 
                variant="outline" 
                size="sm" 
                disabled={roomPage === 1}
                onClick={() => setRoomPage(roomPage - 1)}
                className="cursor-pointer"
              >
                Trang trước
              </Button>
              <span className="text-sm font-bold text-slate-600">Trang {roomPage}</span>
              <Button 
                variant="outline" 
                size="sm" 
                disabled={rooms.length < limit}
                onClick={() => setRoomPage(roomPage + 1)}
                className="cursor-pointer"
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
