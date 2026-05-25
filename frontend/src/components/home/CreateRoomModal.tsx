import { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { createRoom } from '@/api/room'
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

export function CreateRoomModal({ onClose }: { onClose: () => void }) {
  const houseId = useSelectedStore(state => state.selectedHouse?.id)
  const refreshCurrentRooms = useRoomStore(state => state.refreshCurrentRooms)
  
  const [isLoading, setIsLoading] = useState(false)

  const { register, handleSubmit, control, formState: { errors } } = useForm<CreateRoomFormValues>({
    resolver: zodResolver(createRoomSchema),
    defaultValues: {
      name: '',
      price: '0',
      maxTenants: '2'
    }
  })

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isLoading) onClose()
    }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [onClose, isLoading])

  if (!houseId) return null

  const onSubmit = async (values: CreateRoomFormValues) => {
    setIsLoading(true)
    try {
      await createRoom({ 
        house_id: houseId, 
        name: values.name, 
        price: parseNumber(values.price), 
        max_tenants: Number(values.maxTenants),
        status: 'AVAILABLE'
      })
      await refreshCurrentRooms()
      onClose()
    } catch {
      alert("Lỗi khi thêm phòng, vui lòng kiểm tra lại thông tin!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div 
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 backdrop-blur-sm px-4"
      onMouseDown={e => {
        if (e.target === e.currentTarget && !isLoading) onClose()
      }}
    >
      <Card className="w-full max-w-sm p-6 bg-white shadow-xl border-0 animate-in zoom-in-95 duration-200">
        <h2 className="text-xl font-bold mb-4 text-slate-800">Thêm phòng mới</h2>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label>Tên phòng</Label>
            <Input {...register('name')} placeholder="vd: Phòng 101" className="border-slate-200" />
            {errors.name && <span className="text-red-500 text-xs">{errors.name.message}</span>}
          </div>
          <div className="space-y-2">
            <Label>Giá thuê (VNĐ)</Label>
            <Controller
              name="price"
              control={control}
              render={({ field }) => (
                <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-slate-200" />
              )}
            />
          </div>
          <div className="space-y-2">
            <Label>Số khách thuê tối đa</Label>
            <Input type="number" min="1" {...register('maxTenants')} className="border-slate-200" />
          </div>
          <div className="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={onClose} className="border-slate-200 text-slate-600">Hủy</Button>
            <Button type="submit" disabled={isLoading} className="bg-purple-600 text-white hover:bg-purple-700 shadow-md">
              {isLoading ? 'Đang tạo...' : 'Tạo mới'}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
