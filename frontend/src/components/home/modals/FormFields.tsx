import { Controller, type Control, type FieldPath, type FieldValues } from 'react-hook-form'
import { Input } from '@/components/ui/input'
import { formatNumber } from '@/utils/format'
import { cn } from '@/lib/utils'

// Select dùng lại style của Input để các ô trên cùng một hàng thẳng nhau.
export const selectFieldClass = 'h-9 w-full cursor-pointer rounded-md border border-input bg-background px-3 text-sm outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30'

export function FieldError({ message }: { message?: string }) {
  return message ? <span className="text-xs text-destructive">{message}</span> : null
}

// Ô nhập tiền: hiển thị có dấu phân cách, lưu lại chuỗi số thuần.
export function MoneyInput<T extends FieldValues>({ control, name, className, placeholder }: {
  control: Control<T>
  name: FieldPath<T>
  className?: string
  placeholder?: string
}) {
  return (
    <Controller
      name={name}
      control={control}
      render={({ field }) => (
        <Input
          {...field}
          inputMode="numeric"
          placeholder={placeholder}
          value={formatNumber(field.value)}
          onChange={e => field.onChange(e.target.value.replace(/\D/g, ''))}
          className={cn('border-input', className)}
        />
      )}
    />
  )
}
