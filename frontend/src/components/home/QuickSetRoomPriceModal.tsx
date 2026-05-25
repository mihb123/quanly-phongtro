import { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { X, Save, DollarSign, Users } from 'lucide-react'
import { updateRoom } from '@/api/room'
import { formatNumber, parseNumber } from '@/utils/format'
import { useRoomStore } from '@/data/roomData'

export function QuickSetRoomPriceModal({ onClose }: { onClose: () => void }) {
  const rooms = useRoomStore(state => state.rooms)
  const refreshCurrentRooms = useRoomStore(state => state.refreshCurrentRooms)
  const [roomData, setRoomData] = useState<Record<string, { price: string, max_tenants: string }>>({})
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    const initialData: Record<string, { price: string, max_tenants: string }> = {}
    rooms.forEach(room => {
      initialData[room.id] = {
        price: room.price?.toString() || '0',
        max_tenants: room.max_tenants?.toString() || '2'
      }
    })
    setRoomData(initialData)
  }, [rooms])

  const handleInputChange = (id: string, field: 'price' | 'max_tenants', value: string) => {
    setRoomData(prev => ({
      ...prev,
      [id]: {
        ...prev[id],
        [field]: value
      }
    }))
  }

  const handleSaveAll = async () => {
    setIsLoading(true)
    try {
      const promises = Object.entries(roomData).map(([id, data]) => {
        const originalRoom = rooms.find(r => r.id === id)
        // Only update if changed
        if (originalRoom && (originalRoom.price?.toString() !== data.price || originalRoom.max_tenants?.toString() !== data.max_tenants)) {
          return updateRoom(id, {
            house_id: originalRoom.house_id,
            price: parseNumber(data.price),
            max_tenants: Number(data.max_tenants)
          })
        }
        return Promise.resolve()
      })
      await Promise.all(promises)
      await refreshCurrentRooms()
      onClose()
    } catch {
      alert("Lỗi khi cập nhật giá phòng, vui lòng kiểm tra lại!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 py-10 overflow-y-auto">
      <Card className="w-full max-w-3xl bg-white shadow-2xl border-0 animate-in zoom-in-95 duration-200 flex flex-col h-full max-h-[85vh]">
        <div className="p-6 border-b border-slate-100 flex justify-between items-center bg-slate-50/50 rounded-t-2xl">
          <div>
            <h2 className="text-xl font-bold text-slate-800 flex items-center gap-2">
              <DollarSign className="w-5 h-5 text-purple-600" />
              Sửa giá & công năng phòng nhanh
            </h2>
            <p className="text-xs text-slate-500 mt-1">Điều chỉnh giá thuê và sức chứa cho nhiều phòng cùng lúc.</p>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} className="rounded-full text-slate-400 hover:text-slate-600 hover:bg-slate-100">
            <X className="w-5 h-5" />
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto p-6">
          <div className="space-y-4">
            <div className="grid grid-cols-12 gap-4 px-3 py-2 text-xs font-bold text-slate-400 uppercase tracking-wider bg-slate-50 rounded-lg">
              <div className="col-span-4">Tên phòng</div>
              <div className="col-span-4 flex items-center gap-2">
                <DollarSign className="w-3 h-3" /> Giá thuê (VNĐ)
              </div>
              <div className="col-span-4 flex items-center gap-2">
                <Users className="w-3 h-3" /> Số người tối đa
              </div>
            </div>

            <div className="space-y-2">
              {rooms.map(room => (
                <div key={room.id} className="grid grid-cols-12 gap-4 items-center p-3 rounded-xl hover:bg-slate-50 border border-transparent hover:border-slate-100 transition-all group">
                  <div className="col-span-4 font-bold text-slate-700 group-hover:text-purple-600 transition-colors">
                    {room.name}
                  </div>
                  <div className="col-span-4">
                    <Input
                      type="text"
                      value={formatNumber(roomData[room.id]?.price || '')}
                      onChange={e => handleInputChange(room.id, 'price', e.target.value.replace(/\D/g, ''))}
                      className="h-9 border-slate-200 focus:border-purple-400 focus:ring-purple-400/20"
                    />
                  </div>
                  <div className="col-span-4">
                    <Input
                      type="number"
                      value={roomData[room.id]?.max_tenants || ''}
                      onChange={e => handleInputChange(room.id, 'max_tenants', e.target.value)}
                      className="h-9 border-slate-200 focus:border-purple-400 focus:ring-purple-400/20"
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="p-6 border-t border-slate-100 flex justify-end gap-3 bg-slate-50/30 rounded-b-2xl">
          <Button variant="outline" onClick={onClose} className="border-slate-200 text-slate-600 hover:bg-white rounded-xl h-11 px-6 font-bold">
            Hủy
          </Button>
          <Button 
            disabled={isLoading}
            onClick={handleSaveAll}
            className="bg-purple-600 hover:bg-purple-700 text-white shadow-lg shadow-purple-600/20 rounded-xl h-11 px-8 font-bold flex items-center gap-2 transition-all active:scale-95"
          >
            {isLoading ? 'Đang lưu...' : <><Save className="w-4 h-4" /> Lưu tất cả</>}
          </Button>
        </div>
      </Card>
    </div>
  )
}
