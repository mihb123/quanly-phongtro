import { useState, useEffect } from 'react'
import { createPortal } from 'react-dom'
import { Building } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { formatNumber, parseNumber } from '@/utils/format'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'

const houseSchema = z.object({
  name: z.string().min(1, 'Bắt buộc'),
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

export function CreateHouseModal({ onClose }: { onClose: () => void }) {
  const { createHouse } = useHouseStore()
  const createRoom = useRoomStore(state => state.createRoom)
  
  const { register, handleSubmit, control, watch, formState: { errors } } = useForm<HouseFormValues>({
    resolver: zodResolver(houseSchema),
    defaultValues: {
      name: '',
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

  const [floorCountStr, setFloorCountStr] = useState('0')
  const [roomsPerFloor, setRoomsPerFloor] = useState<Record<number, number>>({})
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isLoading) onClose()
    }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [onClose, isLoading])

  const handleFloorCountChange = (val: string) => {
    setFloorCountStr(val);
    const count = parseInt(val) || 0;
    const newRooms = { ...roomsPerFloor };
    for (let i = 1; i <= count; i++) {
        if (newRooms[i] === undefined) newRooms[i] = 1;
    }
    setRoomsPerFloor(newRooms);
  }

  const onSubmit = async (values: HouseFormValues) => {
    setIsLoading(true)
    try {
      const house = await createHouse({ 
         name: values.name, 
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

      if (!house) throw new Error("Create house failed")

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
      alert("Lỗi khi tạo nhà trọ, vui lòng kiểm tra lại!")
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

  return createPortal(
    <div 
      className="fixed inset-0 z-[60] flex items-center justify-center bg-background/80 backdrop-blur-sm p-4"
      onMouseDown={e => {
        if (e.target === e.currentTarget && !isLoading) onClose()
      }}
    >
      <Card className="w-full max-w-2xl bg-card text-card-foreground shadow-xl border border-border/40 safe-fade-in max-h-[95vh] flex flex-col overflow-y-auto">
        <div className="p-4 md:p-6 border-b border-border/40 shrink-0">
          <h2 className="text-xl font-bold text-foreground flex items-center gap-2">
            <Building className="w-5 h-5 text-primary" />
            Tạo nhà trọ mới & Cấu hình tầng
          </h2>
        </div>
        <div className="p-4 md:p-6 overflow-y-auto flex-1">
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 col-span-2">
              <Label>Tên nhà trọ</Label>
              <Input {...register('name')} placeholder="vd: Trọ Cầu Giấy" className="border-border" />
              {errors.name && <span className="text-destructive text-xs">{errors.name.message}</span>}
            </div>
            <div className="space-y-2 col-span-2">
              <Label>Địa chỉ</Label>
              <Input {...register('address')} placeholder="Nhập địa chỉ đầy đủ" className="border-border" />
              {errors.address && <span className="text-destructive text-xs">{errors.address.message}</span>}
            </div>
            
            {/* Điện */}
            <div className="space-y-2 col-span-2 bg-muted/30 rounded-xl p-4 border border-border/50">
              <div className="flex items-center gap-4 mb-2">
                <div className="flex-1">
                  <Label className="text-xs font-bold text-foreground">Cách tính tiền điện</Label>
                  <select {...register('electricity_billing_type')} className="w-full h-8 px-2 rounded-lg border border-border text-sm font-semibold mt-1 bg-background cursor-pointer">
                    <option value="USAGE">Theo nhu cầu (chỉ số)</option>
                    <option value="FIXED">Theo giá mặc định</option>
                  </select>
                </div>
                {electricityBillingType === 'FIXED' && (
                  <div className="flex-1">
                    <Label className="text-xs font-bold text-foreground">Đơn vị tính</Label>
                    <select {...register('electricity_billing_unit')} className="w-full h-8 px-2 rounded-lg border border-border text-sm font-semibold mt-1 bg-background cursor-pointer">
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
                    <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-border h-8 bg-background" />
                  )}
                />
              </div>
            </div>

            {/* Nước */}
            <div className="space-y-2 col-span-2 bg-muted/30 rounded-xl p-4 border border-border/50">
              <div className="flex items-center gap-4 mb-2">
                <div className="flex-1">
                  <Label className="text-xs font-bold text-foreground">Cách tính tiền nước</Label>
                  <select {...register('water_billing_type')} className="w-full h-8 px-2 rounded-lg border border-border text-sm font-semibold mt-1 bg-background cursor-pointer">
                    <option value="USAGE">Theo nhu cầu (chỉ số)</option>
                    <option value="FIXED">Theo giá mặc định</option>
                  </select>
                </div>
                {waterBillingType === 'FIXED' && (
                  <div className="flex-1">
                    <Label className="text-xs font-bold text-foreground">Đơn vị tính</Label>
                    <select {...register('water_billing_unit')} className="w-full h-8 px-2 rounded-lg border border-border text-sm font-semibold mt-1 bg-background cursor-pointer">
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
                    <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-border h-8 bg-background" />
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
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-border h-8 bg-background" />
                )}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-xs">Giá gửi xe / xe (VNĐ)</Label>
              <Controller
                name="parking"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-border h-8 bg-background" />
                )}
              />
            </div>
            <div className="space-y-2 col-span-2">
              <Label className="text-xs">Giá dịch vụ chung / người (VNĐ)</Label>
              <Controller
                name="service"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-border h-8 bg-background" />
                )}
              />
            </div>

            {/* Phụ thu */}
            <div className="space-y-2 col-span-2 bg-secondary/30 rounded-xl p-4 border border-border/50 mt-2">
              <Label className="text-xs font-bold text-foreground block mb-2">Quy định phụ thu (nếu có)</Label>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Phụ thu nếu quá X người</Label>
                  <div className="flex gap-2">
                    <Input type="number" {...register('extra_person_threshold')} placeholder="0" min="0" className="border-border h-8 w-16 bg-background" title="Số người miễn phí" />
                    <Controller
                      name="extra_person_fee"
                      control={control}
                      render={({ field }) => (
                        <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-border h-8 flex-1 bg-background" placeholder="Giá/người (VNĐ)" />
                      )}
                    />
                  </div>
                </div>
                <div className="space-y-1">
                  <Label className="text-xs text-muted-foreground">Phụ thu nếu quá X xe</Label>
                  <div className="flex gap-2">
                    <Input type="number" {...register('extra_vehicle_threshold')} placeholder="0" min="0" className="border-border h-8 w-16 bg-background" title="Số xe miễn phí" />
                    <Controller
                      name="extra_vehicle_fee"
                      control={control}
                      render={({ field }) => (
                        <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-border h-8 flex-1 bg-background" placeholder="Giá/xe (VNĐ)" />
                      )}
                    />
                  </div>
                </div>
              </div>
              <p className="text-[10px] text-muted-foreground mt-2 italic">* Để 0 nếu không áp dụng phụ thu.</p>
            </div>
          </div>
          
          <hr className="my-4 border-border/50" />
          <div className="space-y-4 bg-secondary/20 rounded-xl p-4 border border-border/50">
            <h3 className="font-bold text-foreground text-sm flex items-center gap-2">
              <Building className="w-4 h-4 text-primary"/>
              Cấu trúc số phòng theo tầng
            </h3>
            <p className="text-xs text-muted-foreground">Hệ thống sẽ tự động khởi tạo danh sách phòng dựa vào số lượng bạn cấu hình bên dưới. Nhập 0 nếu không muốn auto-generate.</p>
            <div className="space-y-2">
              <Label>Số tầng của toà nhà (bao gồm cả trệt/thượng)</Label>
              <Input type="number" min="0" max="20" value={floorCountStr} onChange={e => handleFloorCountChange(e.target.value)} className="border-border max-w-[200px] bg-background" />
            </div>
            
            {(parseInt(floorCountStr) || 0) > 0 && (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4 bg-background p-3 rounded-lg border border-border max-h-48 overflow-y-auto">
                {Array.from({ length: parseInt(floorCountStr) || 0 }).map((_, i) => {
                   const floorNo = i + 1;
                   return (
                     <div key={floorNo} className="space-y-1">
                       <Label className="text-xs text-muted-foreground">Số phòng Tầng {floorNo}</Label>
                       <Input 
                         type="number" min="0" 
                         value={roomsPerFloor[floorNo] ?? 1} 
                         onChange={e => setRoomsPerFloor(prev => ({...prev, [floorNo]: parseInt(e.target.value) || 0}))} 
                         className="h-8 text-sm border-border bg-background" 
                       />
                     </div>
                   )
                })}
              </div>
            )}
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={onClose} className="font-bold">Hủy</Button>
            <Button type="submit" disabled={isLoading} className="shadow-sm font-bold">
              {isLoading ? 'Đang khởi tạo...' : 'Xác nhận tạo'}
            </Button>
          </div>
        </form>
        </div>
      </Card>
    </div>,
    document.body
  )
}
