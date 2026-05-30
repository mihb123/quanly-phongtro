import { useState, useEffect } from 'react'
import { Plus, DoorOpen, Pencil, DollarSign, UserPlus, Trash2 } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'
import { CreateRoomModal } from './modals/CreateRoomModal'
import { EditRoomModal } from './modals/EditRoomModal'
import { TenantRoomModal } from './modals/TenantRoomModal'
import { ConfirmModal } from './modals/ConfirmModal'
import { QuickSetRoomPriceModal } from './modals/QuickSetRoomPriceModal'
import type { Room } from '@/api/room'

export function HouseRoomsView() {
  const selectedHouse = useSelectedStore(state => state.selectedHouse)
  const { rooms, roomPage, setRoomPage, deleteRoom, fetchRooms } = useRoomStore()

  const limit = 25

  const [showCreateRoom, setShowCreateRoom] = useState(false)
  const [editRoom, setEditRoom] = useState<Room | null>(null)
  const [tenantRoom, setTenantRoom] = useState<Room | null>(null)
  const [showQuickSetPrice, setShowQuickSetPrice] = useState(false)
  const [roomToDelete, setRoomToDelete] = useState<Room | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    if (selectedHouse) {
      fetchRooms(selectedHouse.id, roomPage)
    }
  }, [selectedHouse, roomPage, fetchRooms])

  if (!selectedHouse) return null

  return (
    <div className="space-y-6 animate-in fade-in zoom-in duration-300">
      {/* Modals */}
      {showCreateRoom && (
        <CreateRoomModal onClose={() => setShowCreateRoom(false)} />
      )}
      {editRoom && (
        <EditRoomModal room={editRoom} onClose={() => setEditRoom(null)} />
      )}
      {tenantRoom && (
        <TenantRoomModal room={tenantRoom} onClose={() => setTenantRoom(null)} />
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

      <div className="flex justify-between items-center border-b border-slate-200 pb-6">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-slate-800">{selectedHouse.name}</h1>
          <p className="text-slate-500 mt-1">{selectedHouse.address}</p>
        </div>
        <div className="flex items-center gap-3">
           <Button onClick={() => setShowQuickSetPrice(true)} variant="outline" className="border-purple-200 text-purple-600 hover:bg-purple-50 rounded-xl cursor-pointer h-10 px-4 font-bold transition-all active:scale-95 flex items-center gap-2">
             <DollarSign className="w-4 h-4" /> Set giá nhanh
           </Button>
           <Button onClick={() => setShowCreateRoom(true)} className="bg-purple-600 hover:bg-purple-700 shadow-md text-white rounded-xl cursor-pointer h-10 px-6 font-bold transition-all active:scale-95 flex items-center gap-2">
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
            {rooms.map(room => (
              <Card key={room.id} onClick={() => setEditRoom(room)} className="p-6 bg-white/80 border-slate-200/60 shadow-sm hover:shadow-md transition-shadow group relative cursor-pointer">
                <div className="flex justify-between items-start mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-lg bg-purple-100 flex items-center justify-center text-purple-600 group-hover:scale-110 transition-transform">
                      <DoorOpen className="w-5 h-5" />
                    </div>
                    <div>
                      <h3 className="font-bold text-lg text-slate-800">{room.name}</h3>
                      <span className="text-xs font-semibold px-2 py-1 rounded-full bg-slate-100 text-slate-600 border border-slate-200 mt-1 inline-block">
                        {room.status === 'AVAILABLE' ? 'Trống' : room.status === 'OCCUPIED' ? 'Đã Thuê' : room.status}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center opacity-0 group-hover:opacity-100 transition-opacity gap-1">
                    <Button variant="ghost" size="icon" onClick={(e) => { e.stopPropagation(); setTenantRoom(room) }} className="h-8 w-8 text-slate-400 hover:text-green-600 hover:bg-green-50 rounded-full transition-colors cursor-pointer" title="Thêm khách thuê">
                      <UserPlus className="w-4 h-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={(e) => { e.stopPropagation(); setEditRoom(room) }} className="h-8 w-8 text-slate-400 hover:text-purple-600 hover:bg-purple-50 rounded-full transition-colors cursor-pointer" title="Sửa thông tin phòng">
                      <Pencil className="w-4 h-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={(e) => { e.stopPropagation(); setRoomToDelete(room) }} className="h-8 w-8 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-full transition-colors cursor-pointer" title="Xóa phòng">
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
                <div className="space-y-2 mt-5 text-sm text-slate-600">
                  <div className="flex justify-between items-center bg-slate-50 p-2 rounded-md">
                    <span>Giá thuê:</span>
                    <span className="font-semibold text-slate-800 text-base">{(room.price || 0).toLocaleString()}đ</span>
                  </div>
                  <div className="flex justify-between items-center p-2">
                    <span>Sức chứa:</span>
                    <span className="font-semibold text-slate-800">{room.max_tenants} người</span>
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
