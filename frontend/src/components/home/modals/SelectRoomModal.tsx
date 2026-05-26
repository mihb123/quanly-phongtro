import { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import type { Room } from '@/api/room'
import { type Tenant } from '@/api/tenant'
import { useRoomStore } from '@/data/roomData'

interface SelectRoomModalProps {
  houseId: string
  tenants: Tenant[]
  onSelect: (room: Room) => void
  onClose: () => void
}

export function SelectRoomModal({ houseId, tenants, onSelect, onClose }: SelectRoomModalProps) {
  const [rooms, setRooms] = useState<Room[]>([])
  const [loading, setLoading] = useState(true)
  const getRoomsByHouse = useRoomStore(state => state.getRoomsByHouse)

  useEffect(() => {
    const fetchRooms = async () => {
      try {
        const data = await getRoomsByHouse(houseId)
        setRooms(data)
      } catch (err) {
        console.error(err)
      } finally {
        setLoading(false)
      }
    }
    fetchRooms()
  }, [houseId, getRoomsByHouse])

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [onClose])

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 backdrop-blur-sm" onClick={onClose}>
      <Card className="w-full max-w-md bg-white shadow-xl border-0 p-6 animate-in zoom-in-95 duration-200" onClick={e => e.stopPropagation()}>
        <h3 className="text-lg font-bold text-slate-800 mb-4">Chọn phòng để thêm khách thuê</h3>
        
        {loading ? (
          <p className="text-sm text-slate-500 text-center py-4">Đang tải danh sách phòng...</p>
        ) : (
          <div className="space-y-4 max-h-200 overflow-y-auto pr-2">
            {rooms.length === 0 ? (
              <p className="text-sm text-slate-500 text-center py-4">Nhà này chưa có phòng nào.</p>
            ) : (
              rooms.map(r => {
                const currentTenants = tenants.filter(t => t.room_id === r.id).length
                const isFull = currentTenants >= r.max_tenants
                
                return (
                  <button
                    key={r.id}
                    onClick={() => onSelect(r)}
                    disabled={isFull}
                    className="w-full flex items-center justify-between p-3 rounded-xl border border-slate-200 hover:border-blue-400 hover:bg-blue-50 transition-colors text-left disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
                  >
                    <div>
                      <span className="font-bold text-slate-700 block">{r.name}</span>
                      <span className="text-xs text-slate-500 font-semibold">{currentTenants} / {r.max_tenants} người</span>
                    </div>
                    <span className={`text-xs font-semibold px-2 py-1 rounded-full ${isFull ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>
                      {isFull ? 'Đã đầy' : 'Có thể thêm'}
                    </span>
                  </button>
                )
              })
            )}
          </div>
        )}
        
        <div className="mt-6 flex justify-end">
          <Button onClick={onClose} variant="outline" className="font-bold cursor-pointer">Hủy</Button>
        </div>
      </Card>
    </div>
  )
}
