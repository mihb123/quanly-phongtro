import { useState, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { X, Loader2, Save } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useInvoiceStore } from '@/data/invoiceData'
import { useHouseStore } from '@/data/houseData'
import type { Invoice } from '@/api/invoice'
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm'

const schema = z.object({
  old_electricity_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số điện không được âm').optional(),
  new_electricity_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số điện không được âm').optional(),
  old_water_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số nước không được âm').optional(),
  new_water_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số nước không được âm').optional(),
  tenant_count: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Số người không hợp lệ'),
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
    formState: { errors, isDirty },
  } = useForm<EditInvoiceFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      old_electricity_index: invoice.old_electricity_index,
      new_electricity_index: invoice.new_electricity_index,
      old_water_index: invoice.old_water_index,
      new_water_index: invoice.new_water_index,
      tenant_count: invoice.tenant_count,
      vehicle_count: invoice.vehicle_count,
      other_fee: invoice.other_fee,
      discount: invoice.discount,
    },
  })

  const { handleClose, confirmModal } = useDirtyConfirm(isDirty, onClose, isSubmitting)

  const onSubmit = async (data: EditInvoiceFormValues) => {
    setIsSubmitting(true)
    // The backend CreateInvoice endpoint acts as an upsert for the same room and period
    const res = await updateInvoice({
      room_id: invoice.room_id,
      period: invoice.period,
      old_electricity_index: isElectricityFixed ? undefined : data.old_electricity_index,
      new_electricity_index: isElectricityFixed ? 0 : (data.new_electricity_index ?? 0),
      old_water_index: isWaterFixed ? undefined : data.old_water_index,
      new_water_index: isWaterFixed ? 0 : (data.new_water_index ?? 0),
      tenant_count: data.tenant_count,
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
      className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-0 bg-background/80 backdrop-blur-sm transition-opacity"
      onClick={handleClose}
    >
      <div 
        className="relative bg-card text-card-foreground rounded-3xl shadow-2xl border border-border/40 w-full max-w-lg overflow-hidden safe-fade-in"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex justify-between items-center p-6 border-b border-border/40 bg-muted/30">
          <div>
            <h2 className="text-xl font-extrabold text-foreground">Sửa hóa đơn</h2>
            <p className="text-sm font-medium text-muted-foreground mt-1">
              Phòng {invoice.room_name} - Kỳ {invoice.period}
            </p>
          </div>
          <button 
            type="button"
            onClick={handleClose}
            className="w-8 h-8 flex items-center justify-center rounded-full bg-secondary text-muted-foreground hover:bg-secondary/80 hover:text-foreground transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="p-6 space-y-6">
          {(!isElectricityFixed || !isWaterFixed) && (
            <div className="grid grid-cols-2 gap-4">
              {!isElectricityFixed && (
                <div className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-sm font-bold text-foreground">
                      Chỉ số điện cũ
                    </label>
                    <Input 
                      type="number" 
                      {...register('old_electricity_index', { valueAsNumber: true })}
                      className="h-11 rounded-xl border-border bg-background"
                    />
                    {errors.old_electricity_index && <p className="text-destructive text-xs font-medium">{errors.old_electricity_index.message}</p>}
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-bold text-foreground">
                      Chỉ số điện mới
                    </label>
                    <Input 
                      type="number" 
                      {...register('new_electricity_index', { valueAsNumber: true })}
                      className="h-11 rounded-xl border-border bg-background"
                    />
                    {errors.new_electricity_index && <p className="text-destructive text-xs font-medium">{errors.new_electricity_index.message}</p>}
                  </div>
                </div>
              )}

              {!isWaterFixed && (
                <div className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-sm font-bold text-foreground">
                      Chỉ số nước cũ
                    </label>
                    <Input 
                      type="number" 
                      {...register('old_water_index', { valueAsNumber: true })}
                      className="h-11 rounded-xl border-border bg-background"
                    />
                    {errors.old_water_index && <p className="text-destructive text-xs font-medium">{errors.old_water_index.message}</p>}
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-bold text-foreground">
                      Chỉ số nước mới
                    </label>
                    <Input 
                      type="number" 
                      {...register('new_water_index', { valueAsNumber: true })}
                      className="h-11 rounded-xl border-border bg-background"
                    />
                    {errors.new_water_index && <p className="text-destructive text-xs font-medium">{errors.new_water_index.message}</p>}
                  </div>
                </div>
              )}
            </div>
          )}

          {(isElectricityFixed || isWaterFixed) && (
            <div className="bg-primary/10 border border-primary/20 rounded-xl p-3 text-xs text-primary font-medium space-y-1">
              {isElectricityFixed && <p>⚡ Tiền điện tính theo giá cố định</p>}
              {isWaterFixed && <p>💧 Tiền nước tính theo giá cố định</p>}
            </div>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-bold text-foreground">Số người ở</label>
              <Input 
                type="number" 
                {...register('tenant_count', { valueAsNumber: true })}
                className="h-11 rounded-xl border-border bg-background"
              />
              {errors.tenant_count && <p className="text-destructive text-xs font-medium">{errors.tenant_count.message}</p>}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-bold text-foreground">Số lượng xe</label>
              <Input 
                type="number" 
                {...register('vehicle_count', { valueAsNumber: true })}
                className="h-11 rounded-xl border-border bg-background"
              />
              {errors.vehicle_count && <p className="text-destructive text-xs font-medium">{errors.vehicle_count.message}</p>}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-bold text-foreground">Phí khác (VNĐ)</label>
              <Input 
                type="number" 
                {...register('other_fee', { valueAsNumber: true })}
                className="h-11 rounded-xl border-border bg-background"
              />
              {errors.other_fee && <p className="text-destructive text-xs font-medium">{errors.other_fee.message}</p>}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-bold text-foreground">Giảm giá (VNĐ)</label>
              <Input 
                type="number" 
                {...register('discount', { valueAsNumber: true })}
                className="h-11 rounded-xl border-border bg-background"
              />
              {errors.discount && <p className="text-destructive text-xs font-medium">{errors.discount.message}</p>}
            </div>
          </div>

          <div className="flex gap-3 pt-4 border-t border-border/40">
            <Button 
              type="button" 
              variant="outline" 
              onClick={handleClose}
              className="flex-1 rounded-xl h-12 font-bold cursor-pointer"
            >
              Hủy bỏ
            </Button>
            <Button 
              type="submit" 
              disabled={isSubmitting}
              className="flex-1 rounded-xl h-12 shadow-sm font-bold cursor-pointer flex items-center justify-center gap-2 active:scale-95 transition-all"
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
      {confirmModal}
    </div>
  )
}
