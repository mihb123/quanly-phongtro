import { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useHouseStore } from '@/data/houseData'
import { useSelectedStore } from '@/data/selectedData'
import { formatNumber, parseNumber } from '@/utils/format'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import type { House } from '@/api/house'

const houseSchema = z.object({
  name: z.string().min(1, 'Bắt buộc'),
  address: z.string().min(1, 'Bắt buộc'),
  electricity: z.string(),
  water: z.string(),
  wifi: z.string(),
  parking: z.string(),
  service: z.string(),
})

type HouseFormValues = z.infer<typeof houseSchema>

interface EditHouseModalProps {
  house: House
  onClose: () => void
}

export function EditHouseModal({ house, onClose }: EditHouseModalProps) {
  const { fetchHouses, updateHouse } = useHouseStore()
  const { selectedHouse, selectHouse } = useSelectedStore()
  
  const { register, handleSubmit, control, formState: { errors } } = useForm<HouseFormValues>({
    resolver: zodResolver(houseSchema),
    defaultValues: {
      name: house.name || '',
      address: house.address || '',
      electricity: house.default_electricity_price?.toString() || '0',
      water: house.default_water_price?.toString() || '0',
      wifi: house.default_wifi_price?.toString() || '0',
      parking: house.default_parking_price?.toString() || '0',
      service: house.default_service_price?.toString() || '0'
    }
  })

  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isLoading) onClose()
    }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [onClose, isLoading])

  const onSubmit = async (values: HouseFormValues) => {
    setIsLoading(true)
    try {
      const updatedHouse = await updateHouse(house.id, { 
         name: values.name, 
         address: values.address,
         default_electricity_price: parseNumber(values.electricity),
         default_water_price: parseNumber(values.water),
         default_wifi_price: parseNumber(values.wifi),
         default_parking_price: parseNumber(values.parking),
         default_service_price: parseNumber(values.service)
      })

      if (updatedHouse) {
        await fetchHouses()
        if (selectedHouse?.id === updatedHouse.id) {
          selectHouse(updatedHouse)
        }
        onClose()
      } else {
        alert("Lỗi khi cập nhật nhà trọ, vui lòng kiểm tra lại!")
      }
    } catch {
      alert("Lỗi khi cập nhật nhà trọ, vui lòng kiểm tra lại!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div 
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 overflow-y-auto pt-20 pb-20"
      onMouseDown={e => {
        if (e.target === e.currentTarget && !isLoading) onClose()
      }}
    >
      <Card className="w-full max-w-2xl p-6 bg-white shadow-xl border-0 animate-in zoom-in-95 duration-200">
        <h2 className="text-xl font-bold mb-4 text-slate-800">Cập nhật thông tin nhà trọ</h2>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 col-span-2">
              <Label>Tên nhà trọ</Label>
              <Input {...register('name')} placeholder="vd: Trọ Cầu Giấy" className="border-slate-200" />
              {errors.name && <span className="text-red-500 text-xs">{errors.name.message}</span>}
            </div>
            <div className="space-y-2 col-span-2">
              <Label>Địa chỉ</Label>
              <Input {...register('address')} placeholder="Nhập địa chỉ đầy đủ" className="border-slate-200" />
              {errors.address && <span className="text-red-500 text-xs">{errors.address.message}</span>}
            </div>
            
            {/* Phí mặc định */}
            <div className="space-y-2">
              <Label className="text-xs">Giá điện mặc định / số (VNĐ)</Label>
              <Controller
                name="electricity"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 h-8" />
                )}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-xs">Giá nước mặc định (theo khối hoặc người)</Label>
              <Controller
                name="water"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 h-8" />
                )}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-xs">Giá Wifi / phòng (VNĐ)</Label>
              <Controller
                name="wifi"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 h-8" />
                )}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-xs">Giá gửi xe / xe (VNĐ)</Label>
              <Controller
                name="parking"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 h-8" />
                )}
              />
            </div>
            <div className="space-y-2 col-span-2">
              <Label className="text-xs">Giá dịch vụ chung / người (VNĐ)</Label>
              <Controller
                name="service"
                control={control}
                render={({ field }) => (
                  <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200 h-8" />
                )}
              />
            </div>
          </div>
          
          <div className="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={onClose} className="border-slate-200 text-slate-600">Hủy</Button>
            <Button type="submit" disabled={isLoading} className="bg-purple-600 text-white hover:bg-purple-700 shadow-md">
              {isLoading ? 'Đang cập nhật...' : 'Cập nhật'}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
