import { useState } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { Room } from '@/api/room'
import { formatNumber, parseNumber } from '@/utils/format'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm'

const roomSchema = z.object({
  name: z.string().min(1, 'Bắt buộc'),
  price: z.string(),
  maxTenants: z.string(),
  electricity: z.string(),
  water: z.string(),
  wifi: z.string(),
  parking: z.string(),
  service: z.string(),
  extraPersonThreshold: z.string(),
  extraPersonFee: z.string(),
  extraVehicleThreshold: z.string(),
  extraVehicleFee: z.string(),
})

type RoomFormValues = z.infer<typeof roomSchema>

export function EditRoomModal({ room, onClose }: { room: Room, onClose: () => void }) {
  const house = useSelectedStore(state => state.selectedHouse)
  const updateRoomStore = useRoomStore(state => state.updateRoom)
  
  const [isLoading, setIsLoading] = useState(false)

  const { register, handleSubmit, control, formState: { errors, isDirty } } = useForm<RoomFormValues>({
    resolver: zodResolver(roomSchema),
    defaultValues: {
      name: room.name || '',
      price: room.price?.toString() || '0',
      maxTenants: room.max_tenants?.toString() || '2',
      electricity: room.electricity_price?.toString() || house?.default_electricity_price?.toString() || '',
      water: room.water_price?.toString() || house?.default_water_price?.toString() || '',
      wifi: room.wifi_price?.toString() || house?.default_wifi_price?.toString() || '',
      parking: room.parking_price?.toString() || house?.default_parking_price?.toString() || '',
      service: room.service_price?.toString() || house?.default_service_price?.toString() || '',
      extraPersonThreshold: room.extra_person_threshold?.toString() || '',
      extraPersonFee: room.extra_person_fee?.toString() || '',
      extraVehicleThreshold: room.extra_vehicle_threshold?.toString() || '',
      extraVehicleFee: room.extra_vehicle_fee?.toString() || '',
    }
  })

  const { handleClose, confirmModal } = useDirtyConfirm(isDirty, onClose, isLoading)

  if (!house) return null

  const onSubmit = async (values: RoomFormValues) => {
    setIsLoading(true)
    try {
      await updateRoomStore(room.id, { 
        house_id: house.id, 
        name: values.name, 
        price: parseNumber(values.price), 
        max_tenants: Number(values.maxTenants),
        status: room.status,
        electricity_price: values.electricity !== '' ? parseNumber(values.electricity) : undefined,
        water_price: values.water !== '' ? parseNumber(values.water) : undefined,
        wifi_price: values.wifi !== '' ? parseNumber(values.wifi) : undefined,
        parking_price: values.parking !== '' ? parseNumber(values.parking) : undefined,
        service_price: values.service !== '' ? parseNumber(values.service) : undefined,
        extra_person_threshold: values.extraPersonThreshold !== '' ? Number(values.extraPersonThreshold) : undefined,
        extra_person_fee: values.extraPersonFee !== '' ? parseNumber(values.extraPersonFee) : undefined,
        extra_vehicle_threshold: values.extraVehicleThreshold !== '' ? Number(values.extraVehicleThreshold) : undefined,
        extra_vehicle_fee: values.extraVehicleFee !== '' ? parseNumber(values.extraVehicleFee) : undefined,
      })
      onClose()
    } catch {
      alert("Lỗi khi cập nhật phòng, vui lòng kiểm tra lại thông tin!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div 
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 overflow-y-auto pt-20 pb-20"
      onMouseDown={e => {
        if (e.target === e.currentTarget && !isLoading) handleClose()
      }}
    >
      <Card className="w-full max-w-xl p-6 bg-white shadow-xl border-0 animate-in zoom-in-95 duration-200">
        <h2 className="text-xl font-bold mb-4 text-slate-800">Sửa thông tin phòng</h2>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 col-span-2">
              <Label>Tên phòng</Label>
              <Input {...register('name')} className="border-slate-200" />
              {errors.name && <span className="text-red-500 text-xs">{errors.name.message}</span>}
            </div>
            <div className="space-y-2">
              <Label>Giá thuê hàng tháng (VNĐ)</Label>
              <Controller
                name="price"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200" />
                )}
              />
            </div>
            <div className="space-y-2">
              <Label>Số người ở tối đa</Label>
              <Input type="number" min="1" {...register('maxTenants')} className="border-slate-200" />
            </div>
          </div>
          
          <hr className="my-2 border-slate-100" />
          <div className="bg-slate-50 p-4 rounded-xl border border-slate-100">
             <h3 className="font-bold text-slate-700 text-sm">Tuỳ chỉnh giá phát sinh riêng</h3>
             <p className="text-xs text-slate-500 mb-3">Nếu để trống, hệ thống sẽ tự động dùng giá mặc định của nhà trọ.</p>
             <div className="grid grid-cols-2 md:grid-cols-3 gap-3 text-sm">
                <div className="space-y-1">
                  <Label className="text-xs">Giá điện / số</Label>
                  <Controller
                    name="electricity"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-slate-200 bg-white" />
                    )}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Giá nước</Label>
                  <Controller
                    name="water"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-slate-200 bg-white" />
                    )}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Giá Wifi</Label>
                  <Controller
                    name="wifi"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-slate-200 bg-white" />
                    )}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Giá gửi xe</Label>
                  <Controller
                    name="parking"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-slate-200 bg-white" />
                    )}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Giá dịch vụ chung</Label>
                  <Controller
                    name="service"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-slate-200 bg-white" />
                    )}
                  />
                </div>
              </div>
          </div>

          <div className="bg-amber-50 p-4 rounded-xl border border-amber-100">
             <h3 className="font-bold text-amber-800 text-sm">Phụ thu vượt mức</h3>
             <p className="text-xs text-amber-600 mb-3">Nếu để trống, hệ thống sẽ tự động dùng cấu hình mặc định của nhà trọ.</p>
             <div className="grid grid-cols-2 gap-3 text-sm">
                <div className="space-y-1">
                  <Label className="text-xs">Số người miễn phí</Label>
                  <Input type="number" min="0" {...register('extraPersonThreshold')} placeholder="Mặc định..." className="h-8 border-amber-200 bg-white" />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Phí / người vượt (VNĐ)</Label>
                  <Controller
                    name="extraPersonFee"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-amber-200 bg-white" />
                    )}
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Số xe miễn phí</Label>
                  <Input type="number" min="0" {...register('extraVehicleThreshold')} placeholder="Mặc định..." className="h-8 border-amber-200 bg-white" />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Phí / xe vượt (VNĐ)</Label>
                  <Controller
                    name="extraVehicleFee"
                    control={control}
                    render={({ field }) => (
                      <Input {...field} placeholder="Mặc định..." value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="h-8 border-amber-200 bg-white" />
                    )}
                  />
                </div>
             </div>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={handleClose} className="border-slate-200 text-slate-600">Hủy</Button>
            <Button type="submit" disabled={isLoading} className="bg-purple-600 text-white hover:bg-purple-700 shadow-md">
              {isLoading ? 'Đang lưu...' : 'Lưu thay đổi'}
            </Button>
          </div>
        </form>
      </Card>
      
      {confirmModal}
    </div>
  )
}
