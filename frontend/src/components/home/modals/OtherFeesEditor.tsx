import { Plus, Trash2 } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { CurrencyInput } from '@/components/ui/currency-input'
import type { InvoiceFeeItem } from '@/api/invoice'
import { formatCurrency } from '@/utils/format'

interface OtherFeesEditorProps {
  value: InvoiceFeeItem[]
  onChange: (value: InvoiceFeeItem[]) => void
}

export function OtherFeesEditor({ value, onChange }: OtherFeesEditorProps) {
  const total = value.reduce((acc, item) => acc + (item.amount || 0), 0)

  const updateItem = (index: number, patch: Partial<InvoiceFeeItem>) => {
    onChange(value.map((item, i) => (i === index ? { ...item, ...patch } : item)))
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-medium text-foreground">Chi phí phát sinh</p>
        {value.length > 0 && (
          <span className="text-xs text-muted-foreground tabular-nums">Tổng: {formatCurrency(total)}</span>
        )}
      </div>

      {value.map((item, index) => (
        <div key={index} className="flex items-center gap-2">
          <Input
            value={item.name}
            onChange={(e) => updateItem(index, { name: e.target.value })}
            aria-label={`Tên khoản phát sinh ${index + 1}`}
            placeholder="Tên khoản, VD: Vệ sinh trả phòng"
            maxLength={100}
            className="h-11 min-w-0 flex-1 rounded-lg border-border bg-background"
          />
          <CurrencyInput
            value={item.amount}
            onChange={(amount) => updateItem(index, { amount })}
            aria-label={`Số tiền khoản phát sinh ${index + 1}`}
            placeholder="Số tiền"
            inputMode="numeric"
            className="h-11 w-32 shrink-0 rounded-lg border-border bg-background text-right tabular-nums sm:w-40"
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={() => onChange(value.filter((_, i) => i !== index))}
            className="shrink-0 text-muted-foreground hover:text-destructive"
            aria-label="Xoá khoản phát sinh"
          >
            <Trash2 className="size-4" />
          </Button>
        </div>
      ))}

      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => onChange([...value, { name: '', amount: 0 }])}
      >
        <Plus data-icon="inline-start" /> Thêm khoản phát sinh
      </Button>
    </div>
  )
}
