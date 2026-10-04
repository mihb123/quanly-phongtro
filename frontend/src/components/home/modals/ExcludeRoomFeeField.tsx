import { useId } from 'react'
import { Checkbox } from '@/components/ui/checkbox'

interface ExcludeRoomFeeFieldProps {
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  isVacant?: boolean
}

export function ExcludeRoomFeeField({ checked, onCheckedChange, isVacant }: ExcludeRoomFeeFieldProps) {
  const id = useId()
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border/60 p-3">
      <Checkbox id={id} checked={checked} onCheckedChange={(value) => onCheckedChange(Boolean(value))} className="mt-0.5" />
      <label htmlFor={id} className="cursor-pointer space-y-0.5">
        <span className="block text-sm font-medium text-foreground">Không tính tiền phòng</span>
        <span className="block text-xs text-muted-foreground">
          {isVacant ? 'Phòng đã trả — chỉ thu tiền điện, nước và dịch vụ.' : 'Dùng cho phòng đã chuyển đi, chỉ chốt điện nước.'}
        </span>
      </label>
    </div>
  )
}
