import { useState, useMemo } from 'react'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Loader2, Save } from '@/components/icons'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useInvoiceStore } from '@/data/invoiceData'
import { useHouseStore } from '@/data/houseData'
import type { Invoice } from '@/api/invoice'
import { useDirtyConfirm } from '@/hooks/useDirtyConfirm'
import { ExcludeRoomFeeField } from './ExcludeRoomFeeField'
import { OtherFeesEditor } from './OtherFeesEditor'
import { CurrencyInput } from '@/components/ui/currency-input'

const schema = z.object({
  old_electricity_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số điện không được âm').optional(),
  new_electricity_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số điện không được âm').optional(),
  old_water_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số nước không được âm').optional(),
  new_water_index: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Chỉ số nước không được âm').optional(),
  tenant_count: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Số người không hợp lệ'),
  vehicle_count: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Số lượng xe không hợp lệ'),
  other_fees: z.array(z.object({
    name: z.string().max(100, 'Tên khoản tối đa 100 ký tự'),
    amount: z.number().min(0, 'Số tiền không được âm'),
  })),
  discount: z.number({ invalid_type_error: "Vui lòng nhập số" }).min(0, 'Giảm giá không được âm'),
  exclude_room_fee: z.boolean(),
})

type EditInvoiceFormValues = z.infer<typeof schema>

interface Props {
  invoice: Invoice
  onClose: () => void
}

// Modal sửa hóa đơn. Vỏ dùng AppModal (đóng qua handleClose để xác nhận khi form dirty), giữ nguyên RHF/Zod.
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
    control,
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
      other_fees: invoice.other_fees?.length
        ? invoice.other_fees
        : invoice.other_fee > 0 ? [{ name: 'Chi phí phát sinh', amount: invoice.other_fee }] : [],
      discount: invoice.discount,
      exclude_room_fee: invoice.room_fee === 0,
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
      other_fees: data.other_fees.filter(item => item.amount > 0),
      discount: data.discount,
      exclude_room_fee: data.exclude_room_fee,
    })
    setIsSubmitting(false)
    if (res.success) {
      onClose()
    } else {
      alert(res.message)
    }
  }

  return (
    <>
    <AppModal
      open
      onClose={handleClose}
      title="Sửa hóa đơn"
      description={`Phòng ${invoice.room_name} - Kỳ ${invoice.period}`}
      contentClassName="sm:max-w-lg"
    >
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <Controller
            control={control}
            name="exclude_room_fee"
            render={({ field }) => (
              <ExcludeRoomFeeField checked={field.value} onCheckedChange={field.onChange} />
            )}
          />

          {(!isElectricityFixed || !isWaterFixed) && (
            <div className="grid grid-cols-2 gap-4">
              {!isElectricityFixed && (
                <div className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground">
                      Chỉ số điện cũ
                    </label>
                    <Input 
                      type="number" 
                      {...register('old_electricity_index', { valueAsNumber: true })}
                      className="h-11 rounded-lg border-border bg-background"
                    />
                    {errors.old_electricity_index && <p className="text-destructive text-xs font-medium">{errors.old_electricity_index.message}</p>}
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground">
                      Chỉ số điện mới
                    </label>
                    <Input 
                      type="number" 
                      {...register('new_electricity_index', { valueAsNumber: true })}
                      className="h-11 rounded-lg border-border bg-background"
                    />
                    {errors.new_electricity_index && <p className="text-destructive text-xs font-medium">{errors.new_electricity_index.message}</p>}
                  </div>
                </div>
              )}

              {!isWaterFixed && (
                <div className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground">
                      Chỉ số nước cũ
                    </label>
                    <Input 
                      type="number" 
                      {...register('old_water_index', { valueAsNumber: true })}
                      className="h-11 rounded-lg border-border bg-background"
                    />
                    {errors.old_water_index && <p className="text-destructive text-xs font-medium">{errors.old_water_index.message}</p>}
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-foreground">
                      Chỉ số nước mới
                    </label>
                    <Input 
                      type="number" 
                      {...register('new_water_index', { valueAsNumber: true })}
                      className="h-11 rounded-lg border-border bg-background"
                    />
                    {errors.new_water_index && <p className="text-destructive text-xs font-medium">{errors.new_water_index.message}</p>}
                  </div>
                </div>
              )}
            </div>
          )}

          {(isElectricityFixed || isWaterFixed) && (
            <div className="bg-primary/10 border border-primary/20 rounded-lg p-3 text-xs text-primary font-medium space-y-1">
              {isElectricityFixed && <p>⚡ Tiền điện tính theo giá cố định</p>}
              {isWaterFixed && <p>💧 Tiền nước tính theo giá cố định</p>}
            </div>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">Số người ở</label>
              <Input
                type="number"
                {...register('tenant_count', { valueAsNumber: true })}
                className="h-11 rounded-lg border-border bg-background"
              />
              {errors.tenant_count && <p className="text-destructive text-xs font-medium">{errors.tenant_count.message}</p>}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">Số lượng xe</label>
              <Input
                type="number"
                {...register('vehicle_count', { valueAsNumber: true })}
                className="h-11 rounded-lg border-border bg-background"
              />
              {errors.vehicle_count && <p className="text-destructive text-xs font-medium">{errors.vehicle_count.message}</p>}
            </div>
          </div>

          <Controller
            control={control}
            name="other_fees"
            render={({ field }) => <OtherFeesEditor value={field.value} onChange={field.onChange} />}
          />
          {errors.other_fees && <p className="text-destructive text-xs font-medium">Vui lòng kiểm tra lại các khoản phát sinh</p>}

          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">Giảm trừ</label>
            <Controller
              control={control}
              name="discount"
              render={({ field }) => (
                <CurrencyInput
                  value={field.value}
                  onChange={field.onChange}
                  placeholder="0"
                  inputMode="numeric"
                  className="h-11 rounded-lg border-border bg-background tabular-nums"
                />
              )}
            />
            {errors.discount && <p className="text-destructive text-xs font-medium">{errors.discount.message}</p>}
          </div>

          <div className="flex gap-3 pt-4 border-t border-border/40">
            <Button 
              type="button" 
              variant="outline" 
              onClick={handleClose}
              className="flex-1"
            >
              Hủy bỏ
            </Button>
            <Button 
              type="submit" 
              disabled={isSubmitting}
              className="flex-1"
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
    </AppModal>
    {confirmModal}
    </>
  )
}
