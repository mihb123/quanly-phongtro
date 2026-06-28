import { useState, useEffect } from 'react'
import { AppModal } from '@/components/shared/AppModal'
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

// Modal chọn phòng để thêm khách thuê. Vỏ dùng AppModal; nội dung là danh sách phòng bấm chọn.
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

  return (
    <AppModal
      open
      onClose={onClose}
      title="Chọn phòng để thêm khách thuê"
      footer={
        <Button onClick={onClose} variant="outline" className="font-bold cursor-pointer">Hủy</Button>
      }
    >
      {loading ? (
        <p className="text-sm text-muted-foreground text-center py-4">Đang tải danh sách phòng...</p>
      ) : (
        <div className="space-y-4 max-h-200 overflow-y-auto pr-2">
          {rooms.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">Nhà này chưa có phòng nào.</p>
          ) : (
            rooms.map(r => {
              const currentTenants = tenants.filter(t => t.room_id === r.id).length
              const isFull = currentTenants >= r.max_tenants

              return (
                <button
                  key={r.id}
                  onClick={() => onSelect(r)}
                  disabled={isFull}
                  className="w-full flex items-center justify-between p-3 rounded-xl border border-border hover:border-primary/50 hover:bg-secondary/50 transition-colors text-left disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
                >
                  <div>
                    <span className="font-bold text-foreground block">{r.name}</span>
                    <span className="text-xs text-muted-foreground font-semibold">{currentTenants} / {r.max_tenants} người</span>
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
    </AppModal>
  )
}
