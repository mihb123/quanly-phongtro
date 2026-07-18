import { Download, Pencil, Send, Trash2 } from '@/components/icons'
import { type Invoice } from '@/api/invoice'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import { formatCurrency } from '@/utils/format'

interface InvoiceMobileCardProps {
  invoice: Invoice
  houseName?: string
  onOpen: () => void
  onEdit: () => void
  onDownload: () => void
  onSendZalo: () => void
  onDelete: () => void
}

// Thẻ hóa đơn mobile gom thông tin quan trọng lên đầu và đặt hành động thành nút có nhãn rõ ràng.
export function InvoiceMobileCard({
  invoice,
  houseName,
  onOpen,
  onEdit,
  onDownload,
  onSendZalo,
  onDelete,
}: InvoiceMobileCardProps) {
  const canSendZalo = invoice.status === 'UNPAID'
  const [periodYear, periodMonth] = invoice.period.split('-')
  const periodLabel = periodMonth && periodYear ? `${periodMonth}/${periodYear}` : invoice.period
  const createdDate = new Date(invoice.created_at).toLocaleDateString('vi-VN')

  return (
    <Card size="sm" className="gap-0 p-0">
      <CardContent className="p-0">
        <Button
          type="button"
          variant="ghost"
          size="lg"
          onClick={onOpen}
          className="h-auto w-full cursor-pointer justify-start whitespace-normal rounded-none p-3 text-left"
          aria-label={`Xem hóa đơn phòng ${invoice.room_name}, kỳ ${invoice.period}`}
        >
          <span className="grid w-full min-w-0 grid-cols-[minmax(0,1fr)_auto] items-start gap-3">
            <span className="flex min-w-0 flex-col gap-1">
              <span className="truncate text-base font-semibold text-foreground">Phòng {invoice.room_name}</span>
              {houseName ? <span className="truncate text-sm font-normal text-muted-foreground">{houseName}</span> : null}
              <span className="text-sm font-normal text-muted-foreground">
                Kỳ {periodLabel} · Lập {createdDate}
              </span>
            </span>
            <span className="flex shrink-0 flex-col items-end gap-2">
              <span className="text-base font-semibold tabular-nums text-foreground">
                {formatCurrency(invoice.total_amount)}
              </span>
              <StatusBadge status={invoice.status} />
            </span>
          </span>
        </Button>
      </CardContent>

      <Separator />

      <CardFooter
        className={cn(
          'grid gap-1 px-2 py-1',
          canSendZalo ? 'grid-cols-4' : 'grid-cols-3',
        )}
      >
        <Button type="button" variant="ghost" size="lg" onClick={onEdit} className="min-w-0 cursor-pointer px-1">
          <Pencil data-icon="inline-start" />
          <span className="truncate">Sửa</span>
        </Button>
        <Button type="button" variant="ghost" size="lg" onClick={onDownload} className="min-w-0 cursor-pointer px-1">
          <Download data-icon="inline-start" />
          <span className="truncate">Tải</span>
        </Button>
        {canSendZalo ? (
          <Button type="button" variant="ghost" size="lg" onClick={onSendZalo} className="min-w-0 cursor-pointer px-1">
            <Send data-icon="inline-start" />
            <span className="truncate">Gửi</span>
          </Button>
        ) : null}
        <Button type="button" variant="ghost" size="lg" onClick={onDelete} className="min-w-0 cursor-pointer px-1">
          <Trash2 data-icon="inline-start" />
          <span className="truncate">Xóa</span>
        </Button>
      </CardFooter>
    </Card>
  )
}
