import { useState } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useSelectedStore } from '@/data/selectedData'
import { useRoomStore } from '@/data/roomData'
import { formatNumber, parseNumber } from '@/utils/format'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'

const createRoomSchema = z.object({
  name: z.string().min(1, 'Bắt buộc'),
  price: z.string(),
  maxTenants: z.string()
})

type CreateRoomFormValues = z.infer<typeof createRoomSchema>

// Modal thêm phòng mới vào nhà trọ đang chọn. Vỏ dùng AppModal, giữ nguyên RHF/Zod + store roomData.
export function CreateRoomModal({ onClose }: { onClose: () => void }) {
  const houseId = useSelectedStore(state => state.selectedHouse?.id)
  const createRoomStore = useRoomStore(state => state.createRoom)
  
  const [isLoading, setIsLoading] = useState(false)

  const { register, handleSubmit, control, formState: { errors } } = useForm<CreateRoomFormValues>({
    resolver: zodResolver(createRoomSchema),
    defaultValues: {
      name: '',
      price: '0',
      maxTenants: '2'
    }
  })

  if (!houseId) return null

  const onSubmit = async (values: CreateRoomFormValues) => {
    setIsLoading(true)
    try {
      const res = await createRoomStore({ 
        house_id: houseId, 
        name: values.name, 
        price: parseNumber(values.price), 
        max_tenants: Number(values.maxTenants),
        status: 'AVAILABLE'
      })
      if (res.success) {
        onClose()
      } else {
        alert(res.error || "Lỗi khi thêm phòng, vui lòng kiểm tra lại thông tin!")
      }
    } catch {
      alert("Lỗi khi thêm phòng, vui lòng kiểm tra lại thông tin!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <AppModal
      open
      onClose={onClose}
      title="Thêm phòng mới"
      footer={
        <div className="flex w-full gap-2 sm:justify-end sm:gap-3">
          <Button type="button" variant="outline" onClick={onClose} className="flex-1 sm:flex-none">Hủy</Button>
          <Button type="submit" form="create-room-form" disabled={isLoading} className="flex-1 sm:flex-none">
            {isLoading ? 'Đang tạo...' : 'Tạo mới'}
          </Button>
        </div>
      }
    >
      <form id="create-room-form" onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="create-room-name">Tên phòng</Label>
            <Input id="create-room-name" {...register('name')} placeholder="vd: Phòng 101" className="border-border" />
            {errors.name && <span className="text-destructive text-xs">{errors.name.message}</span>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="create-room-price">Giá thuê (VNĐ)</Label>
            <Controller
              name="price"
              control={control}
              render={({ field }) => (
                <Input inputMode="numeric" id="create-room-price" {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-border" />
              )}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="create-room-max-tenants">Số khách thuê tối đa</Label>
            <Input id="create-room-max-tenants" type="number" min="1" {...register('maxTenants')} className="border-border" />
          </div>
      </form>
    </AppModal>
  )
}
