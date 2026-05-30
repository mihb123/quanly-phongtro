import { useState, useCallback, useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { X, Loader2, Save } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useInvoiceStore } from '@/data/invoiceData'
import { useHouseStore } from '@/data/houseData'
import type { Invoice } from '@/api/invoice'

const schema = z.object({
  new_electricity_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số điện không được âm').optional(),
  new_water_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số nước không được âm').optional(),
  vehicle_count: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Số lượng xe không hợp lệ'),
  other_fee: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Phí khác không được âm'),
  discount: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Giảm giá không được âm'),
})

type EditInvoiceFormValues = z.infer<typeof schema>

interface Props {
  invoice: Invoice
  onClose: () => void
}

export function EditInvoiceModal({ invoice, onClose }: Props) {
  const { houses } = useHouseStore()
  const { updateInvoice } = useInvoiceStore()
  
  const [isSubmitting, setIsSubmitting] = useState(false)

  // Find the house to determine billing type
  const house = useMemo(() => houses.find(h => h.id === invoice.house_id), [houses, invoice.house_id])

  const isElectricityFixed = house?.electricity_billing_type === 'FIXED'
  const isWaterFixed = house?.water_billing_type === 'FIXED'

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<EditInvoiceFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      new_electricity_index: invoice.new_electricity_index,
      new_water_index: invoice.new_water_index,
      vehicle_count: invoice.vehicle_count,
      other_fee: invoice.other_fee,
      discount: invoice.discount,
    },
  })

  // Handle Escape key
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose()
  }, [onClose])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  const onSubmit = async (data: EditInvoiceFormValues) => {
    setIsSubmitting(true)
    // The backend CreateInvoice endpoint acts as an upsert for the same room and period
    const res = await updateInvoice({
      room_id: invoice.room_id,
      period: invoice.period,
      new_electricity_index: isElectricityFixed ? 0 : (data.new_electricity_index ?? 0),
      new_water_index: isWaterFixed ? 0 : (data.new_water_index ?? 0),
      vehicle_count: data.vehicle_count,
      other_fee: data.other_fee,
      discount: data.discount,
    })
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
            <h2 className="text-xl font-extrabold text-slate-800">Sửa hóa đơn</h2>
            <p className="text-sm font-medium text-slate-500 mt-1">
              Phòng {invoice.room_name} - Kỳ {invoice.period}
            </p>
          </div>
          <button 
            onClick={onClose}
            className="w-8 h-8 flex items-center justify-center rounded-full bg-slate-100 text-slate-500 hover:bg-slate-200 hover:text-slate-700 transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="p-6 space-y-6">
          {(!isElectricityFixed || !isWaterFixed) && (
            <div className="grid grid-cols-2 gap-4">
              {!isElectricityFixed && (
                <div className="space-y-2">
                  <label className="text-sm font-bold text-slate-700">
                    Chỉ số điện mới (Cũ: {invoice.old_electricity_index})
                  </label>
                  <Input 
                    type="number" 
                    {...register('new_electricity_index', { valueAsNumber: true })}
                    className="h-11 rounded-xl bg-slate-50"
                  />
                  {errors.new_electricity_index && <p className="text-red-500 text-xs font-medium">{errors.new_electricity_index.message}</p>}
                </div>
              )}

              {!isWaterFixed && (
                <div className="space-y-2">
                  <label className="text-sm font-bold text-slate-700">
                    Chỉ số nước mới (Cũ: {invoice.old_water_index})
                  </label>
                  <Input 
                    type="number" 
                    {...register('new_water_index', { valueAsNumber: true })}
                    className="h-11 rounded-xl bg-slate-50"
                  />
                  {errors.new_water_index && <p className="text-red-500 text-xs font-medium">{errors.new_water_index.message}</p>}
                </div>
              )}
            </div>
          )}

          {(isElectricityFixed || isWaterFixed) && (
            <div className="bg-blue-50 border border-blue-100 rounded-xl p-3 text-xs text-blue-700 font-medium space-y-1">
              {isElectricityFixed && <p>⚡ Tiền điện tính theo giá cố định</p>}
              {isWaterFixed && <p>💧 Tiền nước tính theo giá cố định</p>}
            </div>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-bold text-slate-700">Số lượng xe</label>
              <Input 
                type="number" 
                {...register('vehicle_count', { valueAsNumber: true })}
                className="h-11 rounded-xl bg-slate-50"
              />
              {errors.vehicle_count && <p className="text-red-500 text-xs font-medium">{errors.vehicle_count.message}</p>}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-bold text-slate-700">Phí khác (VNĐ)</label>
              <Input 
                type="number" 
                {...register('other_fee', { valueAsNumber: true })}
                className="h-11 rounded-xl bg-slate-50"
              />
              {errors.other_fee && <p className="text-red-500 text-xs font-medium">{errors.other_fee.message}</p>}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-bold text-slate-700">Giảm giá (VNĐ)</label>
              <Input 
                type="number" 
                {...register('discount', { valueAsNumber: true })}
                className="h-11 rounded-xl bg-slate-50"
              />
              {errors.discount && <p className="text-red-500 text-xs font-medium">{errors.discount.message}</p>}
            </div>
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
              className="flex-1 rounded-xl h-12 bg-blue-600 hover:bg-blue-700 text-white font-bold cursor-pointer flex items-center justify-center gap-2"
            >
              {isSubmitting ? (
                <Loader2 className="w-5 h-5 animate-spin" />
              ) : (
                <>
                  <Save className="w-4 h-4" />
                  Lưu thay đổi
                </>
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}
