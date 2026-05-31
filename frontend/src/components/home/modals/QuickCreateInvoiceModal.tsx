import { useState, useEffect, useMemo } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { X, Save, Zap, Droplets, CheckCircle2 } from 'lucide-react'
import { useRoomStore } from '@/data/roomData'
import { useHouseStore } from '@/data/houseData'
import { type Room } from '@/api/room'
import { getInvoices, createInvoice, type Invoice } from '@/api/invoice'
import { useInvoiceStore } from '@/data/invoiceData'
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm'

export function QuickCreateInvoiceModal({ onClose }: { onClose: () => void }) {
  const { houses } = useHouseStore()
  const { getRoomsByHouse } = useRoomStore()
  const { fetchInvoices, invoiceFilter } = useInvoiceStore()
  
  const [selectedHouseId, setSelectedHouseId] = useState<string>(invoiceFilter.house_id || '')
  const [rooms, setRooms] = useState<Room[]>([])
  const [latestInvoices, setLatestInvoices] = useState<Record<string, Invoice>>({})
  
  const today = new Date()
  const currentMonth = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}`
  const [period, setPeriod] = useState(currentMonth)
  
  const [invoiceData, setInvoiceData] = useState<Record<string, { new_electricity: string, new_water: string, vehicle_count: string, saved: boolean, error?: string, is_paid?: boolean }>>({})
  const [isLoading, setIsLoading] = useState(false)
  const [isFetchingData, setIsFetchingData] = useState(false)

  // Resolve billing type from the selected house
  const selectedHouseData = useMemo(() => {
    return houses.find(h => h.id === selectedHouseId) || null
  }, [houses, selectedHouseId])

  const isElectricityUsage = selectedHouseData?.electricity_billing_type !== 'FIXED'
  const isWaterUsage = selectedHouseData?.water_billing_type !== 'FIXED'

  const [isDirty, setIsDirty] = useState(false)

  const { handleClose, confirmModal } = useDirtyConfirm(isDirty, onClose, isLoading)

  useEffect(() => {
    if (!selectedHouseId) {
      setRooms([])
      setLatestInvoices({})
      setInvoiceData({})
      return
    }

    const fetchData = async () => {
      setIsFetchingData(true)
      try {
        const fetchedRooms = await getRoomsByHouse(selectedHouseId)
        const occupiedRooms = fetchedRooms.filter(r => r.status === 'OCCUPIED')
        setRooms(occupiedRooms.sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })))
        
        const allInvoices = await getInvoices({ house_id: selectedHouseId, limit: 1000 })
        const sortedInvoices = [...(allInvoices || [])].sort((a, b) => b.period.localeCompare(a.period))
        
        const latest: Record<string, Invoice> = {}
        const periodInvoices: Record<string, Invoice> = {}
        
        sortedInvoices.forEach(inv => {
          if (inv.period === period) {
            periodInvoices[inv.room_id] = inv
          }
          if (!latest[inv.room_id] && inv.period < period) {
            latest[inv.room_id] = inv
          }
        })
        setLatestInvoices(latest)
        
        const initialData: Record<string, { new_electricity: string, new_water: string, vehicle_count: string, saved: boolean, is_paid?: boolean }> = {}
        occupiedRooms.forEach(r => {
          const invForPeriod = periodInvoices[r.id]
          const prevInv = latest[r.id]
          
          if (invForPeriod) {
            initialData[r.id] = { 
              new_electricity: invForPeriod.new_electricity_index.toString(), 
              new_water: invForPeriod.new_water_index.toString(), 
              vehicle_count: invForPeriod.vehicle_count?.toString() || '0',
              saved: true,
              is_paid: invForPeriod.status === 'PAID'
            }
          } else {
            // Default vehicle count to the previous invoice's count if available, otherwise 0
            initialData[r.id] = { 
              new_electricity: '', 
              new_water: '', 
              vehicle_count: prevInv?.vehicle_count?.toString() || '0', 
              saved: false, 
              is_paid: false 
            }
          }
        })
        setInvoiceData(initialData)
        setIsDirty(false)
      } catch (err) {
        console.error("Lỗi khi tải dữ liệu", err)
      } finally {
        setIsFetchingData(false)
      }
    }
    fetchData()
  }, [selectedHouseId, period, getRoomsByHouse])

  const handleInputChange = (id: string, field: 'new_electricity' | 'new_water' | 'vehicle_count', value: string) => {
    setIsDirty(true)
    setInvoiceData(prev => ({
      ...prev,
      [id]: {
        ...prev[id],
        [field]: value,
        saved: false,
        error: undefined
      }
    }))
  }

  const handleSaveAll = async () => {
    if (!selectedHouseId || !period) return
    setIsLoading(true)
    let hasError = false
    let hasSaved = false
    
    try {
      const promises = rooms.map(async (room) => {
        const data = invoiceData[room.id]
        if (!data || data.saved) return Promise.resolve()

        // For USAGE type, both indices must be filled
        if (isElectricityUsage && !data.new_electricity) return Promise.resolve()
        if (isWaterUsage && !data.new_water) return Promise.resolve()
        // For FIXED type, skip index validation — no input needed
        if (!isElectricityUsage && !isWaterUsage && data.saved) return Promise.resolve()
        
        const newElec = isElectricityUsage ? Number(data.new_electricity) : 0
        const newWater = isWaterUsage ? Number(data.new_water) : 0
        
        if (isElectricityUsage) {
          const oldElec = latestInvoices[room.id]?.new_electricity_index || 0
          if (newElec < oldElec) {
            setInvoiceData(prev => ({
              ...prev,
              [room.id]: { ...prev[room.id], error: 'Chỉ số điện mới phải lớn hơn hoặc bằng chỉ số cũ' }
            }))
            hasError = true
            return Promise.resolve()
          }
        }

        if (isWaterUsage) {
          const oldWater = latestInvoices[room.id]?.new_water_index || 0
          if (newWater < oldWater) {
            setInvoiceData(prev => ({
              ...prev,
              [room.id]: { ...prev[room.id], error: 'Chỉ số nước mới phải lớn hơn hoặc bằng chỉ số cũ' }
            }))
            hasError = true
            return Promise.resolve()
          }
        }

        try {
          await createInvoice({
            room_id: room.id,
            period: period,
            new_electricity_index: newElec,
            new_water_index: newWater,
            vehicle_count: Number(data.vehicle_count || 0),
            other_fee: 0,
            discount: 0
          })
          
          setInvoiceData(prev => ({
            ...prev,
            [room.id]: { ...prev[room.id], saved: true, error: undefined }
          }))
          hasSaved = true
        } catch (error: unknown) {
          const err = error as { response?: { data?: { message?: string } } };
          hasError = true
          setInvoiceData(prev => ({
            ...prev,
            [room.id]: { ...prev[room.id], error: err.response?.data?.message || 'Lỗi lưu hóa đơn' }
          }))
        }
      })
      await Promise.all(promises)
      
      if (hasSaved) {
        await fetchInvoices()
      }
      if (!hasError && hasSaved) {
        onClose()
      } else if (!hasSaved && !hasError) {
        // nothing to save
        onClose()
      }
    } catch {
      alert("Lỗi khi tạo hóa đơn, vui lòng kiểm tra lại!")
    } finally {
      setIsLoading(false)
    }
  }

  // Compute how many utility columns to show (0, 1, or 2)
  const showElecColumn = isElectricityUsage
  const showWaterColumn = isWaterUsage
  const bothFixed = !showElecColumn && !showWaterColumn
  const onlyOneColumn = (showElecColumn ? 1 : 0) + (showWaterColumn ? 1 : 0) === 1

  // Compute column spans dynamically — use static class names to avoid Tailwind purge
  const roomSpanClass = 'col-span-2'
  const vehicleSpanClass = 'col-span-2'
  const utilitySpanClass = onlyOneColumn ? 'col-span-8' : 'col-span-4'

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 py-10 overflow-y-auto"
      onMouseDown={e => {
        if (e.target === e.currentTarget && !isLoading) handleClose()
      }}
    >
      <Card className="w-full max-w-4xl bg-white shadow-2xl border-0 animate-in zoom-in-95 duration-200 flex flex-col h-full max-h-[90vh]">
        <div className="p-6 border-b border-slate-100 flex justify-between items-center bg-slate-50/50 rounded-t-2xl">
          <div>
            <h2 className="text-xl font-bold text-slate-800 flex items-center gap-2">
              <Zap className="w-5 h-5 text-purple-600" />
              Ghi chỉ số điện nước & Tạo hóa đơn nhanh
            </h2>
            <p className="text-xs text-slate-500 mt-1">Ghi nhanh chỉ số điện nước mới cho nhiều phòng cùng lúc.</p>
          </div>
          <Button variant="ghost" size="icon" onClick={handleClose} className="rounded-full text-slate-400 hover:text-slate-600 hover:bg-slate-100">
            <X className="w-5 h-5" />
          </Button>
        </div>

        <div className="p-4 bg-white border-b border-slate-100 flex gap-4 items-end">
          <div className="flex-1 max-w-[200px]">
            <label className="block text-xs font-bold text-slate-500 mb-1">Chọn nhà trọ</label>
            <select 
              className="w-full h-10 px-3 rounded-lg border border-slate-200 focus:border-purple-500 focus:ring focus:ring-purple-200 outline-none transition-all text-sm font-semibold"
              value={selectedHouseId}
              onChange={(e) => setSelectedHouseId(e.target.value)}
            >
              <option value="">-- Chọn nhà trọ --</option>
              {houses.map(h => (
                <option key={h.id} value={h.id}>{h.name}</option>
              ))}
            </select>
          </div>
          
          <div className="flex-1 max-w-[200px]">
            <label className="block text-xs font-bold text-slate-500 mb-1">Kỳ hóa đơn</label>
            <input 
              type="month"
              className="w-full h-10 px-3 rounded-lg border border-slate-200 focus:border-purple-500 focus:ring focus:ring-purple-200 outline-none transition-all text-sm font-semibold"
              value={period}
              onChange={(e) => setPeriod(e.target.value)}
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-6 bg-slate-50/30">
          {!selectedHouseId ? (
            <div className="h-full flex flex-col items-center justify-center text-slate-400">
              <Zap className="w-12 h-12 mb-3 text-slate-300" />
              <p>Vui lòng chọn nhà trọ để bắt đầu ghi điện nước</p>
            </div>
          ) : isFetchingData ? (
            <div className="h-full flex items-center justify-center">
              <div className="w-8 h-8 border-4 border-purple-200 border-t-purple-600 rounded-full animate-spin"></div>
            </div>
          ) : rooms.length === 0 ? (
            <div className="text-center text-slate-500 mt-10">Nhà trọ này chưa có phòng nào đang thuê.</div>
          ) : bothFixed ? (
            /* Both electricity and water are FIXED — just show a save button per room */
            <div className="space-y-4">
              <div className="bg-blue-50 border border-blue-100 rounded-xl p-4 text-sm text-blue-700 font-medium">
                <p>⚡ Tiền điện tính theo giá mặc định ({selectedHouseData?.electricity_billing_unit === 'PERSON' ? '/ người' : '/ phòng'})</p>
                <p>💧 Tiền nước tính theo giá mặc định ({selectedHouseData?.water_billing_unit === 'PERSON' ? '/ người' : '/ phòng'})</p>
                <p className="mt-2 text-xs text-blue-500">Bấm "Lưu tất cả" để tạo hóa đơn cho các phòng.</p>
              </div>
              
              <div className="grid grid-cols-12 gap-4 px-3 py-2 text-xs font-bold text-slate-400 uppercase tracking-wider bg-slate-100 rounded-lg mt-4">
                <div className={roomSpanClass}>Phòng</div>
                <div className="col-span-4">Trạng thái</div>
                <div className="col-span-6 flex items-center gap-2">Số lượng xe (Nhập số xe hiện tại)</div>
              </div>

              <div className="space-y-3">
                {rooms.map(room => {
                  const data = invoiceData[room.id] || { new_electricity: '', new_water: '', vehicle_count: '', saved: false }
                  return (
                    <div key={room.id} className={`grid grid-cols-12 gap-4 items-center p-4 rounded-xl bg-white border transition-all ${data.saved ? 'border-green-200 bg-green-50/30' : data.error ? 'border-red-200 bg-red-50/30' : 'border-slate-200 hover:border-purple-300'}`}>
                      <div className={`${roomSpanClass} font-bold text-slate-700 flex items-center gap-2`}>
                        {room.name}
                      </div>
                      <div className="col-span-4 flex items-center gap-2">
                        {data.is_paid ? (
                          <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-500">Đã thu</span>
                        ) : data.saved ? (
                          <CheckCircle2 className="w-4 h-4 text-green-500" />
                        ) : null}
                      </div>
                      <div className="col-span-6">
                         <Input
                           type="number"
                           placeholder="Số xe"
                           value={data.vehicle_count}
                           onChange={e => handleInputChange(room.id, 'vehicle_count', e.target.value)}
                           disabled={data.is_paid}
                           className="h-10 w-24 border-slate-200 focus:border-purple-400 focus:ring-purple-400/20"
                         />
                      </div>
                      {data.error && (
                        <div className="col-span-12 mt-1 text-xs text-red-500 font-medium">{data.error}</div>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Header row */}
              <div className="grid grid-cols-12 gap-4 px-3 py-2 text-xs font-bold text-slate-400 uppercase tracking-wider bg-slate-100 rounded-lg">
                <div className={roomSpanClass}>Phòng</div>
                <div className={vehicleSpanClass}>Số lượng xe</div>
                {showElecColumn && (
                  <div className={`${utilitySpanClass} flex items-center gap-2`}>
                    <Zap className="w-3 h-3 text-yellow-500" /> Chỉ số điện (Mới - Cũ)
                  </div>
                )}
                {showWaterColumn && (
                  <div className={`${utilitySpanClass} flex items-center gap-2`}>
                    <Droplets className="w-3 h-3 text-blue-500" /> Chỉ số nước (Mới - Cũ)
                  </div>
                )}
              </div>

              {/* Fixed billing info banner */}
              {(!showElecColumn || !showWaterColumn) && (
                <div className="bg-blue-50 border border-blue-100 rounded-lg p-3 text-xs text-blue-700 font-medium">
                  {!showElecColumn && <p>⚡ Tiền điện tính theo giá mặc định ({selectedHouseData?.electricity_billing_unit === 'PERSON' ? '/ người' : '/ phòng'})</p>}
                  {!showWaterColumn && <p>💧 Tiền nước tính theo giá mặc định ({selectedHouseData?.water_billing_unit === 'PERSON' ? '/ người' : '/ phòng'})</p>}
                </div>
              )}

              <div className="space-y-3">
                {rooms.map(room => {
                  const data = invoiceData[room.id] || { new_electricity: '', new_water: '', vehicle_count: '', saved: false }
                  const oldElec = latestInvoices[room.id]?.new_electricity_index || 0
                  const oldWater = latestInvoices[room.id]?.new_water_index || 0
                  
                  const elecUsed = data.new_electricity ? Number(data.new_electricity) - oldElec : 0
                  const waterUsed = data.new_water ? Number(data.new_water) - oldWater : 0
                  
                  return (
                    <div key={room.id} className={`grid grid-cols-12 gap-4 items-center p-4 rounded-xl bg-white border transition-all ${data.saved ? 'border-green-200 bg-green-50/30' : data.error ? 'border-red-200 bg-red-50/30' : 'border-slate-200 hover:border-purple-300'}`}>
                      <div className={`${roomSpanClass} font-bold text-slate-700 flex items-center gap-2`}>
                        {room.name}
                        {data.is_paid ? (
                          <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-500">Đã thu</span>
                        ) : data.saved ? (
                          <CheckCircle2 className="w-4 h-4 text-green-500" />
                        ) : null}
                      </div>
                      <div className={vehicleSpanClass}>
                         <Input
                           type="number"
                           placeholder="Số xe"
                           value={data.vehicle_count}
                           onChange={e => handleInputChange(room.id, 'vehicle_count', e.target.value)}
                           disabled={data.is_paid}
                           className="h-10 w-full border-slate-200 focus:border-purple-400 focus:ring-purple-400/20"
                         />
                      </div>
                      {showElecColumn && (
                        <div className={`${utilitySpanClass} relative`}>
                          <div className="flex items-center gap-2">
                            <Input
                              type="number"
                              placeholder="Số mới"
                              value={data.new_electricity}
                              onChange={e => handleInputChange(room.id, 'new_electricity', e.target.value)}
                              disabled={data.is_paid}
                              className={`h-10 w-28 font-bold ${elecUsed < 0 ? 'text-red-500 border-red-300 focus:ring-red-200' : 'border-slate-200 focus:border-purple-400 focus:ring-purple-400/20'}`}
                            />
                            <div className="text-sm font-medium text-slate-400">
                              - <span className="text-slate-500" title="Số cũ">{oldElec}</span> = 
                              <span className={`ml-1 font-bold ${elecUsed < 0 ? 'text-red-500' : elecUsed > 0 ? 'text-yellow-600' : 'text-slate-600'}`}>
                                {data.new_electricity ? elecUsed : '?'}
                              </span>
                            </div>
                          </div>
                        </div>
                      )}
                      {showWaterColumn && (
                        <div className={`${utilitySpanClass} relative`}>
                          <div className="flex items-center gap-2">
                            <Input
                              type="number"
                              placeholder="Số mới"
                              value={data.new_water}
                              onChange={e => handleInputChange(room.id, 'new_water', e.target.value)}
                              disabled={data.is_paid}
                              className={`h-10 w-28 font-bold ${waterUsed < 0 ? 'text-red-500 border-red-300 focus:ring-red-200' : 'border-slate-200 focus:border-purple-400 focus:ring-purple-400/20'}`}
                            />
                            <div className="text-sm font-medium text-slate-400">
                              - <span className="text-slate-500" title="Số cũ">{oldWater}</span> = 
                              <span className={`ml-1 font-bold ${waterUsed < 0 ? 'text-red-500' : waterUsed > 0 ? 'text-blue-600' : 'text-slate-600'}`}>
                                {data.new_water ? waterUsed : '?'}
                              </span>
                            </div>
                          </div>
                        </div>
                      )}
                      {data.error && (
                        <div className="col-span-12 mt-1 text-xs text-red-500 font-medium">
                          {data.error}
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>

        <div className="p-6 border-t border-slate-100 flex justify-end gap-3 bg-white rounded-b-2xl">
          <Button variant="outline" onClick={handleClose} className="border-slate-200 text-slate-600 hover:bg-slate-50 rounded-xl h-11 px-6 font-bold">
            Hủy
          </Button>
          <Button 
            disabled={isLoading || !selectedHouseId || rooms.length === 0}
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
