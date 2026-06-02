import { createPortal } from 'react-dom'
import { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
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
      await createRoomStore({ 
        house_id: houseId, 
        name: values.name, 
        price: parseNumber(values.price), 
        max_tenants: Number(values.maxTenants),
        status: 'AVAILABLE'
      })
      onClose()
    } catch {
      alert("Lỗi khi thêm phòng, vui lòng kiểm tra lại thông tin!")
    } finally {
      setIsLoading(false)
    }
  }

  return createPortal(
    <div 
      className="fixed inset-0 z-[60] flex items-center justify-center bg-background/80 backdrop-blur-sm px-4 p-4 sm:p-0"
      onMouseDown={e => {
        if (e.target === e.currentTarget && !isLoading) onClose()
      }}
    >
      <Card className="w-full max-w-sm p-6 bg-card text-card-foreground shadow-xl border border-border/40 safe-fade-in max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-bold mb-4 text-foreground">Thêm phòng mới</h2>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label>Tên phòng</Label>
            <Input {...register('name')} placeholder="vd: Phòng 101" className="border-border" />
            {errors.name && <span className="text-destructive text-xs">{errors.name.message}</span>}
          </div>
          <div className="space-y-2">
            <Label>Giá thuê (VNĐ)</Label>
            <Controller
              name="price"
              control={control}
              render={({ field }) => (
                <Input {...field} value={formatNumber(field.value)} onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))} className="border-border" />
              )}
            />
          </div>
          <div className="space-y-2">
            <Label>Số khách thuê tối đa</Label>
            <Input type="number" min="1" {...register('maxTenants')} className="border-border" />
          </div>
          <div className="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={onClose} className="font-bold">Hủy</Button>
            <Button type="submit" disabled={isLoading} className="shadow-sm font-bold">
              {isLoading ? 'Đang tạo...' : 'Tạo mới'}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  , document.body)
}
