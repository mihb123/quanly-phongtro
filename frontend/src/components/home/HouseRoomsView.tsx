import { Plus, DoorOpen, Pencil, DollarSign } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import type { House } from '@/api/house'
import type { Room } from '@/api/room'

export function HouseRoomsView({ 
  house, 
  rooms, 
  onOpenCreateRoom, 
  onEditRoom,
  onClickRoom,
  onOpenQuickSetRoomPrice,
  page,
  onPageChange,
  limit
}: { 
  house: House, 
  rooms: Room[], 
  onOpenCreateRoom: () => void, 
  onEditRoom: (r: Room) => void,
  onClickRoom: (r: Room) => void,
  onOpenQuickSetRoomPrice?: () => void,
  page: number,
  onPageChange: (p: number) => void,
  limit: number
}) {
  return (
    <div className="space-y-6 animate-in fade-in zoom-in duration-300">
      <div className="flex justify-between items-center border-b border-slate-200 pb-6">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-slate-800">{house.name}</h1>
          <p className="text-slate-500 mt-1">{house.address}</p>
        </div>
        <div className="flex items-center gap-3">
           <Button onClick={onOpenQuickSetRoomPrice} variant="outline" className="border-purple-200 text-purple-600 hover:bg-purple-50 rounded-xl cursor-pointer h-10 px-4 font-bold transition-all active:scale-95 flex items-center gap-2">
             <DollarSign className="w-4 h-4" /> Set giá nhanh
           </Button>
           <Button onClick={onOpenCreateRoom} className="bg-purple-600 hover:bg-purple-700 shadow-md text-white rounded-xl cursor-pointer h-10 px-6 font-bold transition-all active:scale-95 flex items-center gap-2">
             <Plus className="w-4 h-4" /> Thêm phòng
           </Button>
        </div>
      </div>
      
      {rooms.length === 0 && page === 1 ? (
        <div className="text-center py-20">
          <p className="text-slate-500 italic">Chưa có phòng nào trong nhà trọ này.</p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {rooms.map(room => (
              <Card key={room.id} onClick={() => onClickRoom(room)} className="p-6 bg-white/80 border-slate-200/60 shadow-sm hover:shadow-md transition-shadow group relative cursor-pointer">
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
                  <div className="flex items-center">
                    <Button variant="ghost" size="icon" onClick={(e) => { e.stopPropagation(); onEditRoom(room) }} className="h-8 w-8 text-slate-400 hover:text-purple-600 hover:bg-purple-50 rounded-full transition-colors cursor-pointer">
                      <Pencil className="w-4 h-4" />
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
                    <span className="font-semibold text-slate-800">{room.max_tennants} người</span>
                  </div>
                </div>
              </Card>
            ))}
          </div>

          {/* Pagination Controls */}
          {(page > 1 || rooms.length === limit) && (
            <div className="flex items-center justify-center gap-4 mt-8 pt-6 border-t border-slate-100">
              <Button 
                variant="outline" 
                size="sm" 
                disabled={page === 1}
                onClick={() => onPageChange(page - 1)}
                className="cursor-pointer"
              >
                Trang trước
              </Button>
              <span className="text-sm font-bold text-slate-600">Trang {page}</span>
              <Button 
                variant="outline" 
                size="sm" 
                disabled={rooms.length < limit}
                onClick={() => onPageChange(page + 1)}
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
