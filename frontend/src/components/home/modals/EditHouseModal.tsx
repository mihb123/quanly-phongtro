import { useState, type FocusEvent } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { useHouseStore } from '@/data/houseData'
import { useSelectedStore } from '@/data/selectedData'
import { formatNumber, parseNumber } from '@/utils/format'
import { useForm, Controller, type Control } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { checkHouseCode, type House } from '@/api/house'
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm'
import { HOUSE_CODE_PATTERN, isHouseCodeTaken } from '@/utils/houseCode'
import { FormSection, FormSubGroup } from './FormSection'
import { HouseDocumentsSection } from './HouseDocumentsSection'

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
  owner_name: z.string().max(100, 'Tối đa 100 ký tự'),
  owner_phone: z.string().max(20, 'Tối đa 20 ký tự'),
  owner_rent_price: z.string(),
  owner_deposit: z.string(),
  rent_start_date: z.string(),
  rent_end_date: z.string(),
}).refine(
  v => !v.rent_start_date || !v.rent_end_date || v.rent_start_date <= v.rent_end_date,
  { path: ['rent_end_date'], message: 'Ngày kết thúc phải sau ngày bắt đầu' },
)

type HouseFormValues = z.infer<typeof houseSchema>

type MoneyFieldName =
  | 'electricity' | 'water' | 'wifi' | 'parking' | 'service'
  | 'extra_person_fee' | 'extra_vehicle_fee' | 'owner_rent_price' | 'owner_deposit'

const FORM_ID = 'edit-house-form'

// Select dùng lại style của Input để các ô trên cùng một hàng thẳng nhau.
const selectClass = 'h-9 w-full cursor-pointer rounded-md border border-input bg-background px-3 text-sm outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30'

// Backend trả ngày dạng RFC3339 (hoặc null); input[type=date] chỉ nhận YYYY-MM-DD.
const toDateInputValue = (raw?: string | null) => (raw ? raw.slice(0, 10) : '')

function FieldError({ message }: { message?: string }) {
  return message ? <span className="text-xs text-destructive">{message}</span> : null
}

// Ô nhập tiền: hiển thị có dấu phân cách, lưu lại chuỗi số thuần.
function MoneyInput({ control, name, className, placeholder }: {
  control: Control<HouseFormValues>
  name: MoneyFieldName
  className?: string
  placeholder?: string
}) {
  return (
    <Controller
      name={name}
      control={control}
      render={({ field }) => (
        <Input
          {...field}
          inputMode="numeric"
          placeholder={placeholder}
          value={formatNumber(field.value)}
          onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))}
          className={cn('border-input', className)}
        />
      )}
    />
  )
}

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
      owner_name: house.owner_name || '',
      owner_phone: house.owner_phone || '',
      owner_rent_price: house.owner_rent_price?.toString() || '0',
      owner_deposit: house.owner_deposit?.toString() || '0',
      rent_start_date: toDateInputValue(house.rent_start_date),
      rent_end_date: toDateInputValue(house.rent_end_date),
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
         owner_name: values.owner_name.trim(),
         owner_phone: values.owner_phone.trim(),
         owner_rent_price: parseNumber(values.owner_rent_price),
         owner_deposit: parseNumber(values.owner_deposit),
         rent_start_date: values.rent_start_date,
         rent_end_date: values.rent_end_date,
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
    <AppModal
      open
      onClose={handleClose}
      title="Cập nhật thông tin nhà trọ"
      contentClassName="sm:max-w-2xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>Hủy</Button>
          <Button type="submit" form={FORM_ID} disabled={isLoading} className="bg-purple-600 text-white hover:bg-purple-700 shadow-md">
            {isLoading ? 'Đang cập nhật...' : 'Cập nhật'}
          </Button>
        </>
      }
    >
      <form id={FORM_ID} onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        <FormSection title="Thông tin chung">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label>Tên nhà trọ</Label>
              <Input {...register('name')} placeholder="vd: Trọ Cầu Giấy" className="border-input" />
              <FieldError message={errors.name?.message} />
            </div>
            <div className="space-y-1.5">
              <Label>Mã nhà (House Code)</Label>
              <Input {...register('house_code', { onBlur: handleHouseCodeBlur })} placeholder="vd: ntcg" maxLength={12} className="border-input" />
              <FieldError message={errors.house_code?.message} />
            </div>
            <div className="space-y-1.5 sm:col-span-2">
              <Label>Địa chỉ</Label>
              <Input {...register('address')} placeholder="Nhập địa chỉ đầy đủ" className="border-input" />
              <FieldError message={errors.address?.message} />
            </div>
          </div>
        </FormSection>

        <FormSection title="Cấu hình chi phí" hint="Đơn giá mặc định áp dụng cho mọi phòng của nhà này.">
          <FormSubGroup label="Điện & nước">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-3">
                <div className="space-y-1.5">
                  <Label>Cách tính tiền điện</Label>
                  <select {...register('electricity_billing_type')} className={selectClass}>
                    <option value="USAGE">Theo nhu cầu (chỉ số)</option>
                    <option value="FIXED">Theo giá mặc định</option>
                  </select>
                </div>
                {electricityBillingType === 'FIXED' && (
                  <div className="space-y-1.5">
                    <Label>Đơn vị tính điện</Label>
                    <select {...register('electricity_billing_unit')} className={selectClass}>
                      <option value="ROOM">Theo phòng</option>
                      <option value="PERSON">Theo người</option>
                    </select>
                  </div>
                )}
                <div className="space-y-1.5">
                  <Label>{getElectricityPriceLabel()}</Label>
                  <MoneyInput control={control} name="electricity" />
                </div>
              </div>

              <div className="space-y-3">
                <div className="space-y-1.5">
                  <Label>Cách tính tiền nước</Label>
                  <select {...register('water_billing_type')} className={selectClass}>
                    <option value="USAGE">Theo nhu cầu (chỉ số)</option>
                    <option value="FIXED">Theo giá mặc định</option>
                  </select>
                </div>
                {waterBillingType === 'FIXED' && (
                  <div className="space-y-1.5">
                    <Label>Đơn vị tính nước</Label>
                    <select {...register('water_billing_unit')} className={selectClass}>
                      <option value="ROOM">Theo phòng</option>
                      <option value="PERSON">Theo người</option>
                    </select>
                  </div>
                )}
                <div className="space-y-1.5">
                  <Label>{getWaterPriceLabel()}</Label>
                  <MoneyInput control={control} name="water" />
                </div>
              </div>
            </div>
          </FormSubGroup>

          <FormSubGroup label="Dịch vụ khác">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div className="space-y-1.5">
                <Label>Wifi / phòng (VNĐ)</Label>
                <MoneyInput control={control} name="wifi" />
              </div>
              <div className="space-y-1.5">
                <Label>Gửi xe / xe (VNĐ)</Label>
                <MoneyInput control={control} name="parking" />
              </div>
              <div className="space-y-1.5">
                <Label>Dịch vụ chung / người (VNĐ)</Label>
                <MoneyInput control={control} name="service" />
              </div>
            </div>
          </FormSubGroup>

          <FormSubGroup label="Phụ thu (để 0 nếu không áp dụng)">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label>Phụ thu nếu quá X người</Label>
                <div className="flex gap-2">
                  <Input type="number" {...register('extra_person_threshold')} placeholder="0" min="0" className="w-16 border-input" title="Số người miễn phí" />
                  <MoneyInput control={control} name="extra_person_fee" className="flex-1" placeholder="Giá/người (VNĐ)" />
                </div>
              </div>
              <div className="space-y-1.5">
                <Label>Phụ thu nếu quá X xe</Label>
                <div className="flex gap-2">
                  <Input type="number" {...register('extra_vehicle_threshold')} placeholder="0" min="0" className="w-16 border-input" title="Số xe miễn phí" />
                  <MoneyInput control={control} name="extra_vehicle_fee" className="flex-1" placeholder="Giá/xe (VNĐ)" />
                </div>
              </div>
            </div>
          </FormSubGroup>
        </FormSection>

        <FormSection title="Thông tin thuê nhà" hint="Dùng cho mô hình thuê nguyên căn rồi cho thuê lại từng phòng.">
          <FormSubGroup label="Chủ nhà & điều khoản hợp đồng">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label>Tên chủ nhà</Label>
                <Input {...register('owner_name')} placeholder="vd: Nguyễn Văn A" className="border-input" />
                <FieldError message={errors.owner_name?.message} />
              </div>
              <div className="space-y-1.5">
                <Label>Số điện thoại</Label>
                <Input {...register('owner_phone')} placeholder="vd: 0912345678" className="border-input" />
                <FieldError message={errors.owner_phone?.message} />
              </div>
              <div className="space-y-1.5">
                <Label>Tiền thuê / tháng (VNĐ)</Label>
                <MoneyInput control={control} name="owner_rent_price" />
              </div>
              <div className="space-y-1.5">
                <Label>Tiền cọc (VNĐ)</Label>
                <MoneyInput control={control} name="owner_deposit" />
              </div>
              <div className="space-y-1.5">
                <Label>Ngày bắt đầu thuê</Label>
                <Input type="date" {...register('rent_start_date')} className="border-input" />
              </div>
              <div className="space-y-1.5">
                <Label>Ngày kết thúc thuê</Label>
                <Input type="date" {...register('rent_end_date')} className="border-input" />
                <FieldError message={errors.rent_end_date?.message} />
              </div>
            </div>
            <p className="mt-3 text-xs text-muted-foreground italic">* Tiền thuê ở đây là điều khoản hợp đồng; số thực chi mỗi tháng khai ở mục Chi phí vận hành.</p>
          </FormSubGroup>

          <HouseDocumentsSection house={house} />
          <p className="text-xs text-muted-foreground italic">* CCCD chủ nhà và hợp đồng thuê nhà được lưu ngay khi tải lên hoặc xóa.</p>
        </FormSection>
      </form>
    </AppModal>
    {confirmModal}
    </>
  )
}
