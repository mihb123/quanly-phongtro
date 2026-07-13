import { useState, type FocusEvent } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useHouseStore } from '@/data/houseData'
import { useSelectedStore } from '@/data/selectedData'
import { formatNumber, parseNumber } from '@/utils/format'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { checkHouseCode, type House } from '@/api/house'
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm'
import { HOUSE_CODE_PATTERN, isHouseCodeTaken } from '@/utils/houseCode'

const houseSchema = z.object({
  name: z.string().min(1, 'Bắt buộc'),
  house_code: z.string().min(1, 'Bắt buộc').max(12, 'Tối đa 12 ký tự').regex(HOUSE_CODE_PATTERN, 'Chỉ dùng chữ, số, dấu gạch ngang hoặc gạch dưới'),
  address: z.string().min(1, 'Bắt buộc'),
  electricity: z.string(),
  water: z.string(),
  wifi: z.string(),
  parking: z.string(),
  service: z.string(),
  electricity_billing_type: z.enum(['USAGE', 'FIXED']),
  water_billing_type: z.enum(['USAGE', 'FIXED']),
  electricity_billing_unit: z.enum(['ROOM', 'PERSON']),
  water_billing_unit: z.enum(['ROOM', 'PERSON']),
  extra_person_threshold: z.string(),
  extra_person_fee: z.string(),
  extra_vehicle_threshold: z.string(),
  extra_vehicle_fee: z.string(),
})

type HouseFormValues = z.infer<typeof houseSchema>

interface EditHouseModalProps {
  house: House
  onClose: () => void
}

// Modal cập nhật thông tin nhà trọ (giá mặc định, cách tính điện/nước, phụ thu). Vỏ dùng AppModal, giữ RHF/Zod + xác nhận khi dirty.
export function EditHouseModal({ house, onClose }: EditHouseModalProps) {
  const { updateHouse, houses } = useHouseStore()
  const { selectedHouse, selectHouse } = useSelectedStore()
  
  const { register, handleSubmit, control, watch, setError, clearErrors, formState: { errors, isDirty } } = useForm<HouseFormValues>({
    resolver: zodResolver(houseSchema),
    defaultValues: {
      name: house.name || '',
      house_code: house.house_code || '',
      address: house.address || '',
      electricity: house.default_electricity_price?.toString() || '0',
      water: house.default_water_price?.toString() || '0',
      wifi: house.default_wifi_price?.toString() || '0',
      parking: house.default_parking_price?.toString() || '0',
      service: house.default_service_price?.toString() || '0',
      electricity_billing_type: (house.electricity_billing_type as 'USAGE' | 'FIXED') || 'USAGE',
      water_billing_type: (house.water_billing_type as 'USAGE' | 'FIXED') || 'USAGE',
      electricity_billing_unit: (house.electricity_billing_unit as 'ROOM' | 'PERSON') || 'ROOM',
      water_billing_unit: (house.water_billing_unit as 'ROOM' | 'PERSON') || 'ROOM',
      extra_person_threshold: house.extra_person_threshold?.toString() || '0',
      extra_person_fee: house.extra_person_fee?.toString() || '0',
      extra_vehicle_threshold: house.extra_vehicle_threshold?.toString() || '0',
      extra_vehicle_fee: house.extra_vehicle_fee?.toString() || '0',
    }
  })

  const electricityBillingType = watch('electricity_billing_type')
  const waterBillingType = watch('water_billing_type')

  const [isLoading, setIsLoading] = useState(false)
  const { handleClose, confirmModal } = useDirtyConfirm(isDirty, onClose, isLoading)

  // Kiểm tra tức thời house_code khi rời ô nhập: cảnh báo ngay nếu trùng nhà khác trong hệ thống (bỏ qua chính nhà đang sửa).
  const handleHouseCodeBlur = async (e: FocusEvent<HTMLInputElement>) => {
    const code = e.target.value.trim()
    if (!code || !HOUSE_CODE_PATTERN.test(code) || code.length > 12) return // sai định dạng: để zod xử lý
    if (isHouseCodeTaken(houses, code, house.id)) {
      setError('house_code', { message: 'Mã nhà đã tồn tại, vui lòng chọn mã khác' })
      return
    }
    try {
      const { available } = await checkHouseCode(code, house.id)
      if (!available) {
        setError('house_code', { message: 'Mã nhà đã tồn tại, vui lòng chọn mã khác' })
      } else {
        clearErrors('house_code')
      }
    } catch {
      // Bỏ qua lỗi mạng; backend vẫn validate khi submit.
    }
  }

  const onSubmit = async (values: HouseFormValues) => {
    // Chặn sớm mã nhà trùng với nhà khác (bỏ qua chính nhà đang sửa); backend vẫn là nguồn kiểm tra cuối cùng.
    if (isHouseCodeTaken(houses, values.house_code, house.id)) {
      setError('house_code', { message: 'Mã nhà đã tồn tại, vui lòng chọn mã khác' })
      return
    }

    setIsLoading(true)
    try {
      const res = await updateHouse(house.id, {
         name: values.name, 
         house_code: values.house_code,
         address: values.address,
         default_electricity_price: parseNumber(values.electricity),
         default_water_price: parseNumber(values.water),
         default_wifi_price: parseNumber(values.wifi),
         default_parking_price: parseNumber(values.parking),
         default_service_price: parseNumber(values.service),
         electricity_billing_type: values.electricity_billing_type,
         water_billing_type: values.water_billing_type,
         electricity_billing_unit: values.electricity_billing_unit,
         water_billing_unit: values.water_billing_unit,
         extra_person_threshold: parseInt(values.extra_person_threshold) || 0,
         extra_person_fee: parseNumber(values.extra_person_fee),
         extra_vehicle_threshold: parseInt(values.extra_vehicle_threshold) || 0,
         extra_vehicle_fee: parseNumber(values.extra_vehicle_fee),
      })

      if (res.success && res.house) {
        if (selectedHouse?.id === res.house.id) {
          selectHouse(res.house)
        }
        onClose()
      } else {
        setError('house_code', { message: res.error || 'Mã nhà đã tồn tại hoặc không hợp lệ' })
      }
    } catch {
      setError('house_code', { message: 'Lỗi khi sửa nhà!' })
    } finally {
      setIsLoading(false)
    }
  }

  // Returns the label for the electricity price input based on billing type
  const getElectricityPriceLabel = () => {
    if (electricityBillingType === 'FIXED') {
      return 'Giá điện mặc định (VNĐ)'
    }
    return 'Giá điện mặc định / số (VNĐ)'
  }

  // Returns the label for the water price input based on billing type
  const getWaterPriceLabel = () => {
    if (waterBillingType === 'FIXED') {
      return 'Giá nước mặc định (VNĐ)'
    }
    return 'Giá nước mặc định / khối (VNĐ)'
  }

  return (
    <>
    <AppModal open onClose={handleClose} title="Cập nhật thông tin nhà trọ" contentClassName="sm:max-w-2xl">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 col-span-2">
              <Label>Tên nhà trọ</Label>
              <Input {...register('name')} placeholder="vd: Trọ Cầu Giấy" className="border-slate-200 dark:border-slate-700" />
              {errors.name && <span className="text-red-500 text-xs">{errors.name.message}</span>}
            </div>
            <div className="space-y-2 col-span-2">
              <Label>Mã nhà (House Code)</Label>
              <Input {...register('house_code', { onBlur: handleHouseCodeBlur })} placeholder="vd: ntcg" maxLength={12} className="border-slate-200 dark:border-slate-700" />
              {errors.house_code && <span className="text-red-500 text-xs">{errors.house_code.message}</span>}
            </div>
            <div className="space-y-2 col-span-2">
              <Label>Địa chỉ</Label>
              <Input {...register('address')} placeholder="Nhập địa chỉ đầy đủ" className="border-slate-200 dark:border-slate-700" />
              {errors.address && <span className="text-red-500 text-xs">{errors.address.message}</span>}
            </div>
            
            {/* Điện */}
            <div className="space-y-2 col-span-2 bg-yellow-50/50 dark:bg-yellow-500/10 rounded-xl p-3 border border-yellow-100 dark:border-yellow-500/20">
              <div className="flex items-center gap-4 mb-2">
                <div className="flex-1">
                  <Label className="text-xs font-bold text-yellow-700 dark:text-yellow-500">Cách tính tiền điện</Label>
                  <select {...register('electricity_billing_type')} className="w-full h-8 px-2 rounded-lg border border-yellow-200 dark:border-yellow-500/20 text-sm font-semibold mt-1 bg-white dark:bg-slate-900 cursor-pointer">
                    <option value="USAGE">Theo nhu cầu (chỉ số)</option>
                    <option value="FIXED">Theo giá mặc định</option>
                  </select>
                </div>
                {electricityBillingType === 'FIXED' && (
                  <div className="flex-1">
                    <Label className="text-xs font-bold text-yellow-700 dark:text-yellow-500">Đơn vị tính</Label>
                    <select {...register('electricity_billing_unit')} className="w-full h-8 px-2 rounded-lg border border-yellow-200 dark:border-yellow-500/20 text-sm font-semibold mt-1 bg-white dark:bg-slate-900 cursor-pointer">
                      <option value="ROOM">Theo phòng</option>
                      <option value="PERSON">Theo người</option>
                    </select>
                  </div>
                )}
              </div>
              <div className="space-y-1">
                <Label className="text-xs">{getElectricityPriceLabel()}</Label>
                <Controller
                  name="electricity"
                  control={control}
                  render={({ field }) => (
                    <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 dark:border-slate-700 h-8" />
                  )}
                />
              </div>
            </div>

            {/* Nước */}
            <div className="space-y-2 col-span-2 bg-blue-50/50 dark:bg-blue-500/10 rounded-xl p-3 border border-blue-100 dark:border-blue-500/20">
              <div className="flex items-center gap-4 mb-2">
                <div className="flex-1">
                  <Label className="text-xs font-bold text-blue-700 dark:text-blue-400">Cách tính tiền nước</Label>
                  <select {...register('water_billing_type')} className="w-full h-8 px-2 rounded-lg border border-blue-200 dark:border-blue-500/20 text-sm font-semibold mt-1 bg-white dark:bg-slate-900 cursor-pointer">
                    <option value="USAGE">Theo nhu cầu (chỉ số)</option>
                    <option value="FIXED">Theo giá mặc định</option>
                  </select>
                </div>
                {waterBillingType === 'FIXED' && (
                  <div className="flex-1">
                    <Label className="text-xs font-bold text-blue-700 dark:text-blue-400">Đơn vị tính</Label>
                    <select {...register('water_billing_unit')} className="w-full h-8 px-2 rounded-lg border border-blue-200 dark:border-blue-500/20 text-sm font-semibold mt-1 bg-white dark:bg-slate-900 cursor-pointer">
                      <option value="ROOM">Theo phòng</option>
                      <option value="PERSON">Theo người</option>
                    </select>
                  </div>
                )}
              </div>
              <div className="space-y-1">
                <Label className="text-xs">{getWaterPriceLabel()}</Label>
                <Controller
                  name="water"
                  control={control}
                  render={({ field }) => (
                    <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 dark:border-slate-700 h-8" />
                  )}
                />
              </div>
            </div>

            {/* Phí khác */}
            <div className="space-y-2">
              <Label className="text-xs">Giá Wifi / phòng (VNĐ)</Label>
              <Controller
                name="wifi"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 dark:border-slate-700 h-8" />
                )}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-xs">Giá gửi xe / xe (VNĐ)</Label>
              <Controller
                name="parking"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 dark:border-slate-700 h-8" />
                )}
              />
            </div>
            <div className="space-y-2 col-span-2">
              <Label className="text-xs">Giá dịch vụ chung / người (VNĐ)</Label>
              <Controller
                name="service"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 dark:border-slate-700 h-8" />
                )}
              />
            </div>

            {/* Phụ thu */}
            <div className="space-y-2 col-span-2 bg-slate-50/50 dark:bg-slate-800/50 rounded-xl p-3 border border-slate-200 dark:border-slate-700 mt-2">
              <Label className="text-xs font-bold text-slate-700 dark:text-slate-300 block mb-2">Quy định phụ thu (nếu có)</Label>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <Label className="text-xs">Phụ thu nếu quá X người</Label>
                  <div className="flex gap-2">
                    <Input type="number" {...register('extra_person_threshold')} placeholder="0" min="0" className="border-slate-200 dark:border-slate-700 h-8 w-16" title="Số người miễn phí" />
                    <Controller
                      name="extra_person_fee"
                      control={control}
                      render={({ field }) => (
                        <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 dark:border-slate-700 h-8 flex-1" placeholder="Giá/người (VNĐ)" />
                      )}
                    />
                  </div>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Phụ thu nếu quá X xe</Label>
                  <div className="flex gap-2">
                    <Input type="number" {...register('extra_vehicle_threshold')} placeholder="0" min="0" className="border-slate-200 dark:border-slate-700 h-8 w-16" title="Số xe miễn phí" />
                    <Controller
                      name="extra_vehicle_fee"
                      control={control}
                      render={({ field }) => (
                        <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 dark:border-slate-700 h-8 flex-1" placeholder="Giá/xe (VNĐ)" />
                      )}
                    />
                  </div>
                </div>
              </div>
              <p className="text-[10px] text-slate-500 dark:text-slate-400 mt-2 italic">* Để 0 nếu không áp dụng phụ thu.</p>
            </div>
          </div>
          
          <div className="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={handleClose} className="border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-300">Hủy</Button>
            <Button type="submit" disabled={isLoading} className="bg-purple-600 text-white hover:bg-purple-700 shadow-md">
              {isLoading ? 'Đang cập nhật...' : 'Cập nhật'}
            </Button>
          </div>
        </form>
    </AppModal>
    {confirmModal}
    </>
  )
}
