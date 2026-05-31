import { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { X, Save, DollarSign, Users, Copy, Trash2 } from 'lucide-react'
import type { Room } from '@/api/room'
import { useRoomStore } from '@/data/roomData'
import { formatNumber, parseNumber } from '@/utils/format'
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm'

export function QuickSetRoomPriceModal({ onClose }: { onClose: () => void }) {
  const rooms = useRoomStore(state => state.rooms)
  const deleteRoomStore = useRoomStore(state => state.deleteRoom)
  const createRoomStore = useRoomStore(state => state.createRoom)
  const updateRoomStore = useRoomStore(state => state.updateRoom)
  const [roomData, setRoomData] = useState<Record<string, { name: string, price: string, max_tenants: string }>>({})
  const [isLoading, setIsLoading] = useState(false)
  const [editingNameId, setEditingNameId] = useState<string | null>(null)

  const [isDirty, setIsDirty] = useState(false)

  const { handleClose, confirmModal } = useDirtyConfirm(isDirty, onClose, isLoading)

  useEffect(() => {
    const initialData: Record<string, { name: string, price: string, max_tenants: string }> = {}
    rooms.forEach(room => {
      initialData[room.id] = {
        name: room.name,
        price: room.price?.toString() || '0',
        max_tenants: room.max_tenants?.toString() || '2'
      }
    })
    setRoomData(initialData)
  }, [rooms])

  const handleInputChange = (id: string, field: 'name' | 'price' | 'max_tenants', value: string) => {
    setIsDirty(true)
    setRoomData(prev => ({
      ...prev,
      [id]: {
        ...prev[id],
        [field]: value
      }
    }))
  }

  const handleDuplicateRoom = async (room: Room) => {
    setIsLoading(true)
    try {
      // Find a unique name
      let i = 1
      let newName = `${room.name} (Copy)`
      while (rooms.some(r => r.name === newName)) {
        i++
        newName = `${room.name} (Copy ${i})`
      }
      
      await createRoomStore({
        house_id: room.house_id,
        name: newName,
        price: room.price,
        max_tenants: room.max_tenants,
        status: 'AVAILABLE'
      })
    } catch {
      alert("Lỗi khi nhân bản phòng!")
    } finally {
      setIsLoading(false)
    }
  }

  const handleDeleteRoom = async (room: Room) => {
    if (!confirm(`Bạn có chắc chắn muốn xóa phòng: ${room.name}?`)) return
    setIsLoading(true)
    try {
      await deleteRoomStore(room.id, room.house_id)
    } catch {
      alert("Lỗi khi xóa phòng!")
    } finally {
      setIsLoading(false)
    }
  }

  const handleSaveAll = async () => {
    setIsLoading(true)
    try {
      const promises = Object.entries(roomData).map(([id, data]) => {
        const originalRoom = rooms.find(r => r.id === id)
        // Only update if changed
        if (originalRoom && (originalRoom.name !== data.name || originalRoom.price?.toString() !== data.price || originalRoom.max_tenants?.toString() !== data.max_tenants)) {
          return updateRoomStore(id, {
            house_id: originalRoom.house_id,
            name: data.name,
            price: parseNumber(data.price),
            max_tenants: Number(data.max_tenants),
            status: originalRoom.status,
            electricity_price: originalRoom.electricity_price,
            water_price: originalRoom.water_price,
            wifi_price: originalRoom.wifi_price,
            parking_price: originalRoom.parking_price,
            service_price: originalRoom.service_price
          })
        }
        return Promise.resolve()
      })
      await Promise.all(promises)
      onClose()
    } catch {
      alert("Lỗi khi cập nhật giá phòng, vui lòng kiểm tra lại!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 py-10 overflow-y-auto"
      onMouseDown={e => {
        if (e.target === e.currentTarget && !isLoading) handleClose()
      }}
    >
      <Card className="w-full max-w-3xl bg-white shadow-2xl border-0 animate-in zoom-in-95 duration-200 flex flex-col h-full max-h-[85vh]">
        <div className="p-6 border-b border-slate-100 flex justify-between items-center bg-slate-50/50 rounded-t-2xl">
          <div>
            <h2 className="text-xl font-bold text-slate-800 flex items-center gap-2">
              <DollarSign className="w-5 h-5 text-purple-600" />
              Sửa giá & công năng phòng nhanh
            </h2>
            <p className="text-xs text-slate-500 mt-1">Điều chỉnh giá thuê và sức chứa cho nhiều phòng cùng lúc.</p>
          </div>
          <Button variant="ghost" size="icon" onClick={handleClose} className="rounded-full text-slate-400 hover:text-slate-600 hover:bg-slate-100">
            <X className="w-5 h-5" />
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto p-6">
          <div className="space-y-4">
            <div className="grid grid-cols-12 gap-4 px-3 py-2 text-xs font-bold text-slate-400 uppercase tracking-wider bg-slate-50 rounded-lg">
              <div className="col-span-3">Tên phòng</div>
              <div className="col-span-4 flex items-center gap-2">
                <DollarSign className="w-3 h-3" /> Giá thuê (VNĐ)
              </div>
              <div className="col-span-3 flex items-center gap-2">
                <Users className="w-3 h-3" /> Số người tối đa
              </div>
              <div className="col-span-2 text-right">Hành động</div>
            </div>

            <div className="space-y-2">
              {[...rooms].sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })).map(room => (
                <div key={room.id} className="grid grid-cols-12 gap-4 items-center p-3 rounded-xl hover:bg-slate-50 border border-transparent hover:border-slate-100 transition-all group">
                  <div className="col-span-3 font-bold text-slate-700 transition-colors" onClick={() => setEditingNameId(room.id)}>
                    {editingNameId === room.id ? (
                      <Input
                        autoFocus
                        type="text"
                        value={roomData[room.id]?.name || ''}
                        onChange={e => handleInputChange(room.id, 'name', e.target.value)}
                        onBlur={() => setEditingNameId(null)}
                        onKeyDown={e => e.key === 'Enter' && setEditingNameId(null)}
                        className="h-9 border-slate-200 focus:border-purple-400 focus:ring-purple-400/20"
                      />
                    ) : (
                      <div className="cursor-pointer group-hover:text-purple-600 truncate py-1.5" title="Click để sửa tên phòng">
                        {roomData[room.id]?.name || room.name}
                      </div>
                    )}
                  </div>
                  <div className="col-span-4">
                    <Input
                      type="text"
                      value={formatNumber(roomData[room.id]?.price || '')}
                      onChange={e => handleInputChange(room.id, 'price', e.target.value.replace(/\D/g, ''))}
                      className="h-9 border-slate-200 focus:border-purple-400 focus:ring-purple-400/20"
                    />
                  </div>
                  <div className="col-span-3">
                    <Input
                      type="number"
                      value={roomData[room.id]?.max_tenants || ''}
                      onChange={e => handleInputChange(room.id, 'max_tenants', e.target.value)}
                      className="h-9 border-slate-200 focus:border-purple-400 focus:ring-purple-400/20"
                    />
                  </div>
                  <div className="col-span-2 flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <Button variant="ghost" size="icon" onClick={() => handleDuplicateRoom(room)} disabled={isLoading} className="h-8 w-8 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-full transition-colors cursor-pointer" title="Nhân bản">
                      <Copy className="w-4 h-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => handleDeleteRoom(room)} disabled={isLoading} className="h-8 w-8 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-full transition-colors cursor-pointer" title="Xóa">
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        <div className="p-6 border-t border-slate-100 flex justify-end gap-3 bg-slate-50/30 rounded-b-2xl">
          <Button variant="outline" onClick={handleClose} className="border-slate-200 text-slate-600 hover:bg-white rounded-xl h-11 px-6 font-bold">
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
      {confirmModal}
    </div>
  )
}
