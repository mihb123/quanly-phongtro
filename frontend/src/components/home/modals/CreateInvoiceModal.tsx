import { useState, useEffect, useCallback, useMemo } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { X, Loader2, ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useInvoiceStore } from '@/data/invoiceData'
import { getInvoices, type CreateInvoicePayload } from '@/api/invoice'
import { getTenantsByRoom } from '@/api/tenant'
import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'
import { useRecommendedHouse } from '@/hooks/useRecommendedHouse'
import { useRecommendedRoom } from '@/hooks/useRecommendedRoom'

// Schema is dynamic based on billing type, but we validate at the base level
const schema = z.object({
  house_id: z.string().min(1, 'Vui lòng chọn nhà trọ'),
  room_id: z.string().min(1, 'Vui lòng chọn phòng'),
  period: z.string().min(1, 'Vui lòng chọn kỳ hóa đơn').regex(/^\d{4}-\d{2}$/, 'Định dạng yyyy-mm không hợp lệ'),
  old_electricity_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số điện cũ không được âm').optional().or(z.nan().transform(() => undefined)),
  new_electricity_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số điện không được âm').optional(),
  old_water_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số nước cũ không được âm').optional().or(z.nan().transform(() => undefined)),
  new_water_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số nước không được âm').optional(),
  tenant_count: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Số người không hợp lệ'),
  vehicle_count: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Số lượng xe không hợp lệ'),
  other_fee: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Phí khác không được âm'),
  discount: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Giảm giá không được âm'),
})

export type InvoiceFormValues = z.infer<typeof schema>

interface Props {
  onClose: () => void
}

export function CreateInvoiceModal({ onClose }: Props) {
  const { houses } = useHouseStore()
  const { rooms, fetchRooms } = useRoomStore()
  const selectedHouse = useSelectedStore(state => state.selectedHouse)
  const { createNewInvoice } = useInvoiceStore()

  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isAdditionalOpen, setIsAdditionalOpen] = useState(false)

  // Default to current month
  const today = new Date()
  const currentMonth = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}`

  const initialHouseFallback = selectedHouse?.id || (houses.length === 1 ? houses[0].id : '')
  const { recommendedHouseId, isLoading: isHouseLoading, saveSelectedHouse } = useRecommendedHouse(houses, currentMonth, initialHouseFallback)

  const {
    register,
    handleSubmit,
    control,
    setValue,
    formState: { errors },
  } = useForm<InvoiceFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      house_id: initialHouseFallback,
      room_id: '',
      period: currentMonth,
      old_electricity_index: undefined,
      new_electricity_index: 0,
      old_water_index: undefined,
      new_water_index: 0,
      tenant_count: 0,
      vehicle_count: 0,
      other_fee: 0,
      discount: 0,
    },
  })

  const watchHouseId = useWatch({ control, name: 'house_id' })
  const watchRoomId = useWatch({ control, name: 'room_id' })
  const watchPeriod = useWatch({ control, name: 'period' }) || currentMonth

  const { recommendedRoomId, isLoadingRoom } = useRecommendedRoom(watchHouseId, watchPeriod, rooms)

  // Resolve billing config from the selected house
  const selectedHouseData = useMemo(() => {
    return houses.find(h => h.id === watchHouseId) || null
  }, [houses, watchHouseId])

  const isElectricityFixed = selectedHouseData?.electricity_billing_type === 'FIXED'
  const isWaterFixed = selectedHouseData?.water_billing_type === 'FIXED'

  useEffect(() => {
    if (recommendedHouseId && (!watchHouseId || watchHouseId === initialHouseFallback)) {
      setValue('house_id', recommendedHouseId)
    }
  }, [recommendedHouseId, setValue, watchHouseId, initialHouseFallback])

  useEffect(() => {
    if (recommendedRoomId) {
      setValue('room_id', recommendedRoomId)
    } else {
      setValue('room_id', '')
    }
  }, [recommendedRoomId, setValue])

  useEffect(() => {
    if (watchHouseId) {
      saveSelectedHouse(watchHouseId)
    }
  }, [watchHouseId]) // Removed saveSelectedHouse from dependency to avoid infinite loop if it changes reference, though it's stable

  useEffect(() => {
    if (watchHouseId) {
      fetchRooms(watchHouseId, 1)
    }
  }, [watchHouseId, fetchRooms])

  useEffect(() => {
    async function fetchOldIndex() {
      if (!watchRoomId) {
        setValue('old_electricity_index', undefined)
        setValue('old_water_index', undefined)
        return
      }
      try {
        const data = await getInvoices({ room_id: watchRoomId, limit: 1 })
        if (data && data.length > 0) {
          setValue('old_electricity_index', data[0].new_electricity_index)
          setValue('old_water_index', data[0].new_water_index)
        } else {
          setValue('old_electricity_index', undefined)
          setValue('old_water_index', undefined)
        }

        const tenantsRes = await getTenantsByRoom(watchRoomId)
        if (tenantsRes && tenantsRes.data) {
          setValue('tenant_count', tenantsRes.data.length)
        } else {
          setValue('tenant_count', 0)
        }
      } catch (error) {
        console.error('Failed to fetch previous invoice or tenants:', error)
      }
    }
    fetchOldIndex()
  }, [watchRoomId, setValue])

  // Handle Escape key
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose()
  }, [onClose])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  const onSubmit = async (data: InvoiceFormValues) => {
    setIsSubmitting(true)
    const payload: CreateInvoicePayload = {
      room_id: data.room_id,
      period: data.period,
      new_electricity_index: isElectricityFixed ? 0 : (data.new_electricity_index ?? 0),
      new_water_index: isWaterFixed ? 0 : (data.new_water_index ?? 0),
      tenant_count: data.tenant_count,
      vehicle_count: data.vehicle_count,
      other_fee: data.other_fee,
      discount: data.discount,
    }
    
    if (!isElectricityFixed && typeof data.old_electricity_index === 'number') {
      payload.old_electricity_index = data.old_electricity_index;
    }
    
    if (!isWaterFixed && typeof data.old_water_index === 'number') {
      payload.old_water_index = data.old_water_index;
    }

    const res = await createNewInvoice(payload)
    setIsSubmitting(false)
    if (res.success) {
      onClose()
    } else {
      alert(res.message)
    }
  }

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-0"
      onClick={onClose}
    >
      <div className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm transition-opacity" />
      
      <div 
        className="relative bg-white rounded-3xl shadow-2xl w-full max-w-lg overflow-hidden animate-in fade-in zoom-in-95 duration-200"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex justify-between items-center p-6 border-b border-slate-100 bg-slate-50/50">
          <div>
            <h2 className="text-xl font-extrabold text-slate-800">Tạo hóa đơn mới</h2>
            <p className="text-sm font-medium text-slate-500 mt-1">Hóa đơn điện, nước, dịch vụ</p>
          </div>
          <button 
            onClick={onClose}
            className="w-8 h-8 flex items-center justify-center rounded-full bg-slate-100 text-slate-500 hover:bg-slate-200 hover:text-slate-700 transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="p-6 space-y-6">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-bold text-slate-700">Nhà trọ</label>
              <select 
                {...register('house_id')}
                className="w-full h-11 px-3 rounded-xl border border-slate-200 focus:border-purple-500 focus:ring focus:ring-purple-200 outline-none transition-all text-sm font-medium bg-slate-50 disabled:opacity-50"
                disabled={isHouseLoading}
              >
                <option value="">-- Chọn nhà trọ --</option>
                {houses.map(h => (
                  <option key={h.id} value={h.id}>{h.name}</option>
                ))}
              </select>
              {errors.house_id && <p className="text-red-500 text-xs font-medium">{errors.house_id.message}</p>}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-bold text-slate-700">Phòng</label>
              <select 
                {...register('room_id')}
                className="w-full h-11 px-3 rounded-xl border border-slate-200 focus:border-purple-500 focus:ring focus:ring-purple-200 outline-none transition-all text-sm font-medium bg-slate-50 disabled:opacity-50"
                disabled={!watchHouseId || isLoadingRoom}
              >
                <option value="">{isLoadingRoom ? 'Đang tìm phòng...' : '-- Chọn phòng --'}</option>
                {rooms.map(r => (
                  <option key={r.id} value={r.id}>{r.name}</option>
                ))}
              </select>
              {errors.room_id && <p className="text-red-500 text-xs font-medium">{errors.room_id.message}</p>}
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-bold text-slate-700">Kỳ hóa đơn</label>
            <Input 
              type="month"
              {...register('period')}
              className="h-11 rounded-xl bg-slate-50"
            />
            {errors.period && <p className="text-red-500 text-xs font-medium">{errors.period.message}</p>}
          </div>

          {/* Chỉ hiển thị input chỉ số khi loại tính phí là USAGE */}
          {(!isElectricityFixed || !isWaterFixed) && (
            <div className="space-y-4">
              {!isElectricityFixed && (
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <label className="text-sm font-bold text-slate-700">Chỉ số điện cũ (tùy chọn)</label>
                    <Input 
                      type="number" 
                      {...register('old_electricity_index', { valueAsNumber: true })}
                      className="h-11 rounded-xl bg-slate-50"
                      placeholder="Tự động tính nếu trống"
                    />
                    {errors.old_electricity_index && <p className="text-red-500 text-xs font-medium">{errors.old_electricity_index.message}</p>}
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-bold text-slate-700">Chỉ số điện mới</label>
                    <Input 
                      type="number" 
                      {...register('new_electricity_index', { valueAsNumber: true })}
                      className="h-11 rounded-xl bg-slate-50"
                      placeholder="Ví dụ: 1520"
                    />
                    {errors.new_electricity_index && <p className="text-red-500 text-xs font-medium">{errors.new_electricity_index.message}</p>}
                  </div>
                </div>
              )}

              {!isWaterFixed && (
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <label className="text-sm font-bold text-slate-700">Chỉ số nước cũ (tùy chọn)</label>
                    <Input 
                      type="number" 
                      {...register('old_water_index', { valueAsNumber: true })}
                      className="h-11 rounded-xl bg-slate-50"
                      placeholder="Tự động tính nếu trống"
                    />
                    {errors.old_water_index && <p className="text-red-500 text-xs font-medium">{errors.old_water_index.message}</p>}
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-bold text-slate-700">Chỉ số nước mới</label>
                    <Input 
                      type="number" 
                      {...register('new_water_index', { valueAsNumber: true })}
                      className="h-11 rounded-xl bg-slate-50"
                      placeholder="Ví dụ: 105"
                    />
                    {errors.new_water_index && <p className="text-red-500 text-xs font-medium">{errors.new_water_index.message}</p>}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Info khi FIXED */}
          {(isElectricityFixed || isWaterFixed) && (
            <div className="bg-blue-50 border border-blue-100 rounded-xl p-3 text-xs text-blue-700 font-medium space-y-1">
              {isElectricityFixed && (
                <p>⚡ Tiền điện tính theo giá mặc định ({selectedHouseData?.electricity_billing_unit === 'PERSON' ? '/ người' : '/ phòng'})</p>
              )}
              {isWaterFixed && (
                <p>💧 Tiền nước tính theo giá mặc định ({selectedHouseData?.water_billing_unit === 'PERSON' ? '/ người' : '/ phòng'})</p>
              )}
            </div>
          )}

          <div className="border border-slate-200 rounded-xl overflow-hidden bg-white">
            <button 
              type="button" 
              onClick={() => setIsAdditionalOpen(!isAdditionalOpen)}
              className="w-full flex justify-between items-center p-4 hover:bg-slate-50 transition-colors focus:outline-none"
            >
              <span className="text-sm font-bold text-slate-700">Thông tin bổ sung</span>
              <ChevronDown className={`w-4 h-4 text-slate-500 transition-transform ${isAdditionalOpen ? 'rotate-180' : ''}`} />
            </button>
            
            {isAdditionalOpen && (
              <div className="p-4 pt-0 space-y-4 border-t border-slate-100 mt-1">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <label className="text-sm font-bold text-slate-700">Số người ở</label>
                    <Input 
                      type="number" 
                      {...register('tenant_count', { valueAsNumber: true })}
                      className="h-11 rounded-xl bg-slate-50"
                      placeholder="Ví dụ: 2"
                    />
                    {errors.tenant_count && <p className="text-red-500 text-xs font-medium">{errors.tenant_count.message}</p>}
                  </div>

                  <div className="space-y-2">
                    <label className="text-sm font-bold text-slate-700">Số lượng xe</label>
                    <Input 
                      type="number" 
                      {...register('vehicle_count', { valueAsNumber: true })}
                      className="h-11 rounded-xl bg-slate-50"
                      placeholder="Ví dụ: 2"
                    />
                    {errors.vehicle_count && <p className="text-red-500 text-xs font-medium">{errors.vehicle_count.message}</p>}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <label className="text-sm font-bold text-slate-700">Phí khác (VNĐ)</label>
                    <Input 
                      type="number" 
                      {...register('other_fee', { valueAsNumber: true })}
                      className="h-11 rounded-xl bg-slate-50"
                      placeholder="0"
                    />
                    {errors.other_fee && <p className="text-red-500 text-xs font-medium">{errors.other_fee.message}</p>}
                  </div>

                  <div className="space-y-2">
                    <label className="text-sm font-bold text-slate-700">Giảm giá (VNĐ)</label>
                    <Input 
                      type="number" 
                      {...register('discount', { valueAsNumber: true })}
                      className="h-11 rounded-xl bg-slate-50"
                      placeholder="0"
                    />
                    {errors.discount && <p className="text-red-500 text-xs font-medium">{errors.discount.message}</p>}
                  </div>
                </div>
              </div>
            )}
          </div>

          <div className="flex gap-3 pt-4 border-t border-slate-100">
            <Button 
              type="button" 
              variant="outline" 
              onClick={onClose}
              className="flex-1 rounded-xl h-12 font-bold cursor-pointer"
            >
              Hủy bỏ
            </Button>
            <Button 
              type="submit" 
              disabled={isSubmitting}
              className="flex-1 rounded-xl h-12 bg-purple-600 hover:bg-purple-700 text-white font-bold cursor-pointer"
            >
              {isSubmitting ? <Loader2 className="w-5 h-5 animate-spin" /> : 'Tạo hóa đơn'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}
