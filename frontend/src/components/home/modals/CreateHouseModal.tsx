import { useState, useEffect, type FocusEvent } from 'react'
import { Building } from '@/components/icons'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { parseNumber } from '@/utils/format'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { checkHouseCode } from '@/api/house'
import { HOUSE_CODE_PATTERN, generateHouseCode, isHouseCodeTaken } from '@/utils/houseCode'
import { FormSection, FormSubGroup } from './FormSection'
import { FieldError, MoneyInput, selectFieldClass } from './FormFields'

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

const FORM_ID = 'create-house-form'

// Modal tạo nhà trọ mới kèm cấu hình tầng/số phòng tự sinh. Vỏ dùng AppModal, giữ nguyên RHF/Zod + auto sinh house_code.
export function CreateHouseModal({ onClose }: { onClose: () => void }) {
  const { createHouse, houses } = useHouseStore()
  const createRoom = useRoomStore(state => state.createRoom)
  
  const { register, handleSubmit, control, watch, setValue, setError, clearErrors, formState: { errors } } = useForm<HouseFormValues>({
    resolver: zodResolver(houseSchema),
    defaultValues: {
      name: '',
      house_code: '',
      address: '',
      electricity: '0',
      water: '0',
      wifi: '0',
      parking: '0',
      service: '0',
      electricity_billing_type: 'USAGE',
      water_billing_type: 'USAGE',
      electricity_billing_unit: 'ROOM',
      water_billing_unit: 'ROOM',
      extra_person_threshold: '0',
      extra_person_fee: '0',
      extra_vehicle_threshold: '0',
      extra_vehicle_fee: '0',
    }
  })

  const electricityBillingType = watch('electricity_billing_type')
  const waterBillingType = watch('water_billing_type')
  const houseName = watch('name')

  const [floorCountStr, setFloorCountStr] = useState('0')
  const [roomsPerFloor, setRoomsPerFloor] = useState<Record<number, number>>({})
  const [isLoading, setIsLoading] = useState(false)
  const [isHouseCodeTouched, setIsHouseCodeTouched] = useState(false)

  useEffect(() => {
    if (isHouseCodeTouched) return
    setValue('house_code', generateHouseCode(houseName), { shouldValidate: houseName.trim().length > 0 })
  }, [houseName, isHouseCodeTouched, setValue])

  const handleFloorCountChange = (val: string) => {
    setFloorCountStr(val);
    const count = parseInt(val) || 0;
    const newRooms = { ...roomsPerFloor };
    for (let i = 1; i <= count; i++) {
        if (newRooms[i] === undefined) newRooms[i] = 1;
    }
    setRoomsPerFloor(newRooms);
  }

  // Kiểm tra tức thời house_code khi rời ô nhập: cảnh báo ngay nếu đã trùng trong hệ thống.
  const handleHouseCodeBlur = async (e: FocusEvent<HTMLInputElement>) => {
    const code = e.target.value.trim()
    if (!code || !HOUSE_CODE_PATTERN.test(code) || code.length > 12) return // sai định dạng: để zod xử lý
    if (isHouseCodeTaken(houses, code)) {
      setError('house_code', { message: 'Mã nhà đã tồn tại, vui lòng chọn mã khác' })
      return
    }
    try {
      const { available } = await checkHouseCode(code)
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
    // Chặn sớm mã nhà trùng với nhà đã có (backend vẫn là nguồn kiểm tra cuối cùng).
    if (isHouseCodeTaken(houses, values.house_code)) {
      setError('house_code', { message: 'Mã nhà đã tồn tại, vui lòng chọn mã khác' })
      return
    }

    setIsLoading(true)
    try {
      const res = await createHouse({
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

      if (!res.success) {
        setError('house_code', { message: res.error || 'Mã nhà đã tồn tại hoặc không hợp lệ' })
        return
      }
      
      const house = res.house!

      const promises = []
      const floors = parseInt(floorCountStr) || 0
      if (floors > 0) {
        for (let i = 1; i <= floors; i++) {
           const count = roomsPerFloor[i] || 0;
           for (let j = 1; j <= count; j++) {
              const roomName = `P${i}${j < 10 ? '0' + j : j}`;
              promises.push(createRoom({
                 house_id: house.id,
                 name: roomName,
                 price: 0,
                 max_tenants: 2,
                 status: 'AVAILABLE'
              }))
           }
        }
      }
      if (promises.length > 0) {
         await Promise.all(promises);
      }
      onClose()
    } catch {
      setError('house_code', { message: 'Lỗi khi thêm nhà!' })
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
    <AppModal
      open
      onClose={onClose}
      title={
        <span className="flex items-center gap-2">
          <Building className="w-5 h-5 text-primary" />
          Tạo nhà trọ mới & Cấu hình tầng
        </span>
      }
      contentClassName="sm:max-w-2xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={onClose} className="flex-1 sm:flex-none">Hủy</Button>
          <Button type="submit" form={FORM_ID} disabled={isLoading} className="flex-1 sm:flex-none">
            {isLoading ? 'Đang khởi tạo...' : 'Xác nhận tạo'}
          </Button>
        </>
      }
    >
      <form id={FORM_ID} onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        <FormSection title="Thông tin chung">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label>
                Tên nhà trọ <span className="text-destructive">*</span>
              </Label>
              <Input {...register('name')} placeholder="vd: Trọ Cầu Giấy" className="border-input" />
              <FieldError message={errors.name?.message} />
            </div>
            <div className="space-y-1.5">
              <Label>
                Mã nhà (House Code) <span className="text-destructive">*</span>
              </Label>
              <Input
                {...register('house_code', { onChange: () => setIsHouseCodeTouched(true), onBlur: handleHouseCodeBlur })}
                placeholder="vd: ntcg"
                maxLength={12}
                className="border-input"
              />
              <FieldError message={errors.house_code?.message} />
            </div>
            <div className="space-y-1.5 sm:col-span-2">
              <Label>
                Địa chỉ <span className="text-destructive">*</span>
              </Label>
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
                  <select {...register('electricity_billing_type')} className={selectFieldClass}>
                    <option value="USAGE">Theo nhu cầu (chỉ số)</option>
                    <option value="FIXED">Theo giá mặc định</option>
                  </select>
                </div>
                {electricityBillingType === 'FIXED' && (
                  <div className="space-y-1.5">
                    <Label>Đơn vị tính điện</Label>
                    <select {...register('electricity_billing_unit')} className={selectFieldClass}>
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
                  <select {...register('water_billing_type')} className={selectFieldClass}>
                    <option value="USAGE">Theo nhu cầu (chỉ số)</option>
                    <option value="FIXED">Theo giá mặc định</option>
                  </select>
                </div>
                {waterBillingType === 'FIXED' && (
                  <div className="space-y-1.5">
                    <Label>Đơn vị tính nước</Label>
                    <select {...register('water_billing_unit')} className={selectFieldClass}>
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

        <FormSection
          title="Cấu trúc phòng theo tầng"
          hint="Hệ thống tự khởi tạo danh sách phòng theo số lượng bên dưới. Nhập 0 nếu chưa muốn tạo phòng."
        >
          <div className="space-y-1.5">
            <Label>Số tầng của toà nhà (gồm cả trệt/thượng)</Label>
            <Input
              type="number"
              min="0"
              max="20"
              value={floorCountStr}
              onChange={e => handleFloorCountChange(e.target.value)}
              className="max-w-[200px] border-input"
            />
          </div>

          {(parseInt(floorCountStr) || 0) > 0 && (
            <FormSubGroup label="Số phòng mỗi tầng">
              <div className="grid max-h-48 grid-cols-2 gap-3 overflow-y-auto sm:grid-cols-4">
                {Array.from({ length: parseInt(floorCountStr) || 0 }).map((_, i) => {
                  const floorNo = i + 1
                  return (
                    <div key={floorNo} className="space-y-1.5">
                      <Label className="text-xs text-muted-foreground">Tầng {floorNo}</Label>
                      <Input
                        type="number"
                        min="0"
                        value={roomsPerFloor[floorNo] ?? 1}
                        onChange={e => setRoomsPerFloor(prev => ({ ...prev, [floorNo]: parseInt(e.target.value) || 0 }))}
                        className="border-input"
                      />
                    </div>
                  )
                })}
              </div>
            </FormSubGroup>
          )}
        </FormSection>
      </form>
    </AppModal>
  )
}
