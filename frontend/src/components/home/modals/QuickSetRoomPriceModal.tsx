import { useState, useEffect, useRef } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { X, Save, DollarSign, Users, Copy, Trash2 } from '@/components/icons'
import type { Room } from '@/api/room'
import { useRoomStore } from '@/data/roomData'
import { formatNumber, parseNumber } from '@/utils/format'
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm'

// Modal sửa nhanh giá & sức chứa nhiều phòng cùng lúc (kèm nhân bản/xóa qua nhấn giữ). Vỏ dùng AppModal, giữ nguyên state cục bộ + xác nhận khi dirty.
export function QuickSetRoomPriceModal({ onClose }: { onClose: () => void }) {
  const rooms = useRoomStore(state => state.rooms)
  const deleteRoomStore = useRoomStore(state => state.deleteRoom)
  const createRoomStore = useRoomStore(state => state.createRoom)
  const updateRoomStore = useRoomStore(state => state.updateRoom)
  const [roomData, setRoomData] = useState<Record<string, { name: string, price: string, max_tenants: string }>>({})
  const [isLoading, setIsLoading] = useState(false)
  const [editingNameId, setEditingNameId] = useState<string | null>(null)
  const [showActionsFor, setShowActionsFor] = useState<string | null>(null)
  const [preventClick, setPreventClick] = useState(false)
  const pressTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const handlePressStart = (roomId: string) => {
    setPreventClick(false)
    pressTimer.current = setTimeout(() => {
      setPreventClick(true)
      setShowActionsFor(roomId)
    }, 500)
  }

  const handlePressEnd = () => {
    if (pressTimer.current) clearTimeout(pressTimer.current)
  }

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
      
      const res = await createRoomStore({
        house_id: room.house_id,
        name: newName,
        price: room.price,
        max_tenants: room.max_tenants,
        status: 'AVAILABLE'
      })
      if (!res.success) {
        alert(res.error || "Lỗi khi nhân bản phòng!")
      }
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
      const res = await deleteRoomStore(room.id, room.house_id)
      if (!res.success) {
        alert(res.error || "Lỗi khi xóa phòng!")
      }
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
            service_price: originalRoom.service_price,
            extra_person_threshold: originalRoom.extra_person_threshold,
            extra_person_fee: originalRoom.extra_person_fee,
            extra_vehicle_threshold: originalRoom.extra_vehicle_threshold,
            extra_vehicle_fee: originalRoom.extra_vehicle_fee,
          })
        }
        return Promise.resolve({ success: true, error: undefined })
      })
      const results = await Promise.all(promises)
      const failed = results.filter(r => !r.success)
      if (failed.length > 0) {
         alert(failed[0].error || "Lỗi khi cập nhật giá phòng, vui lòng kiểm tra lại!")
      } else {
         onClose()
      }
    } catch {
      alert("Lỗi khi cập nhật giá phòng, vui lòng kiểm tra lại!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <>
    <AppModal
      open
      onClose={handleClose}
      contentClassName="sm:max-w-3xl"
      title={
        <span className="flex items-center gap-2">
          <DollarSign className="w-5 h-5 text-primary" />
          Sửa giá & công năng phòng nhanh
        </span>
      }
      description="Điều chỉnh giá thuê và sức chứa cho nhiều phòng cùng lúc."
      footer={
        <>
          <Button variant="outline" onClick={handleClose} className="flex-1 sm:flex-none">
            Hủy
          </Button>
          <Button
            disabled={isLoading}
            onClick={handleSaveAll}
            className="flex-1 sm:flex-none"
          >
            {isLoading ? 'Đang lưu...' : <><Save className="w-4 h-4" /> Lưu tất cả</>}
          </Button>
        </>
      }
    >
          <div className="space-y-4">
            <div className="grid grid-cols-12 gap-3 px-3 py-2 text-xs sm:text-xs font-medium text-muted-foreground bg-muted/30 rounded-lg">
              <div className="col-span-4">Tên phòng</div>
              <div className="col-span-5 flex items-center gap-1 sm:gap-2">
                <DollarSign className="w-3 h-3" /> Giá thuê
              </div>
              <div className="col-span-3 flex items-center gap-1 sm:gap-2 justify-end sm:justify-start">
                <Users className="w-3 h-3 hidden sm:block" /> Số người
              </div>
            </div>

            <div className="space-y-2">
              {[...rooms].sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })).map(room => (
                <div key={room.id} className="grid grid-cols-12 gap-3 items-center p-3 rounded-lg hover:bg-muted/30 border border-transparent hover:border-border transition-all relative">
                  {showActionsFor === room.id && (
                    <div className="absolute inset-0 z-10 bg-background/95 backdrop-blur-sm rounded-lg flex items-center justify-center gap-2 sm:gap-4 shadow-sm border border-border/80">
                      <Button variant="outline" onClick={(e) => { e.stopPropagation(); handleDuplicateRoom(room); setShowActionsFor(null); }} disabled={isLoading} className="h-9 px-3 sm:px-4 text-primary border-primary/20 hover:bg-primary/10 font-medium text-xs sm:text-sm">
                        <Copy className="w-4 h-4 sm:mr-2" /> <span className="hidden sm:inline">Nhân bản</span>
                      </Button>
                      <Button variant="outline" onClick={(e) => { e.stopPropagation(); handleDeleteRoom(room); setShowActionsFor(null); }} disabled={isLoading} className="h-9 px-3 sm:px-4 text-destructive border-destructive/20 hover:bg-destructive/10 font-medium text-xs sm:text-sm">
                        <Trash2 className="w-4 h-4 sm:mr-2" /> <span className="hidden sm:inline">Xóa</span>
                      </Button>
                      <Button variant="ghost" size="icon" onClick={(e) => { e.stopPropagation(); setShowActionsFor(null); }} className="absolute right-1 sm:right-2 text-muted-foreground hover:bg-secondary rounded-full">
                        <X className="w-5 h-5"/>
                      </Button>
                    </div>
                  )}
                  <div 
                    className="col-span-4 font-medium text-foreground transition-colors touch-target px-0" 
                    onMouseDown={() => handlePressStart(room.id)}
                    onMouseUp={handlePressEnd}
                    onMouseLeave={handlePressEnd}
                    onTouchStart={() => handlePressStart(room.id)}
                    onTouchEnd={handlePressEnd}
                    onClick={() => {
                      if (!preventClick) setEditingNameId(room.id)
                    }}
                    style={{ WebkitTouchCallout: 'none', userSelect: 'none' }}
                  >
                    {editingNameId === room.id ? (
                      <Input
                        autoFocus
                        type="text"
                        value={roomData[room.id]?.name || ''}
                        onChange={e => handleInputChange(room.id, 'name', e.target.value)}
                        onBlur={() => setEditingNameId(null)}
                        onKeyDown={e => e.key === 'Enter' && setEditingNameId(null)}
                        className="h-9 border-border focus:border-primary focus:ring-primary/20 bg-background text-sm"
                      />
                    ) : (
                      <div className="cursor-pointer hover:text-primary truncate py-1.5" title="Nhấn để sửa tên, nhấn giữ để hiện thao tác">
                        {roomData[room.id]?.name || room.name}
                      </div>
                    )}
                  </div>
                  <div className="col-span-5">
                    <Input
                      type="text"
                      value={formatNumber(roomData[room.id]?.price || '')}
                      onChange={e => handleInputChange(room.id, 'price', e.target.value.replace(/\D/g, ''))}
                      className="h-9 border-border focus:border-primary focus:ring-primary/20 bg-background text-sm font-semibold"
                    />
                  </div>
                  <div className="col-span-3">
                    <Input
                      type="number"
                      value={roomData[room.id]?.max_tenants || ''}
                      onChange={e => handleInputChange(room.id, 'max_tenants', e.target.value)}
                      className="h-9 border-border focus:border-primary focus:ring-primary/20 bg-background text-sm text-center"
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>
    </AppModal>
    {confirmModal}
    </>
  )
}
