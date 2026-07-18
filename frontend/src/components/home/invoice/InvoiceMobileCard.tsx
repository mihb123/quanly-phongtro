import { Download, Pencil, Send, Trash2 } from '@/components/icons'
import { type Invoice } from '@/api/invoice'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
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

// Thẻ hóa đơn mobile nhóm thông tin theo thứ tự quét và giữ hành động icon-only dễ chạm.
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
          <span className="flex w-full min-w-0 flex-col gap-2">
            <span className="flex min-w-0 items-baseline justify-between gap-3">
              <span className="truncate text-base font-semibold text-foreground">Phòng {invoice.room_name}</span>
              <span className="shrink-0 text-lg font-semibold tabular-nums text-foreground">
                {formatCurrency(invoice.total_amount)}
              </span>
            </span>
            <span className="flex min-w-0 items-center justify-between gap-3">
              <span className="flex min-w-0 items-center gap-1 text-sm font-normal text-muted-foreground">
                {houseName ? (
                  <>
                    <span className="truncate">{houseName}</span>
                    <span aria-hidden="true">·</span>
                  </>
                ) : null}
                <span className="shrink-0">Kỳ {periodLabel}</span>
              </span>
              <StatusBadge status={invoice.status} className="shrink-0" />
            </span>
          </span>
        </Button>
      </CardContent>

      <Separator />

      <CardFooter className="flex items-center gap-1 px-2 py-1">
        <span className="mr-auto min-w-0 truncate px-1 text-xs text-muted-foreground">Lập {createdDate}</span>
        <Button
          type="button"
          variant="ghost"
          size="icon-lg"
          onClick={onEdit}
          className="size-11 cursor-pointer"
          aria-label={`Sửa hóa đơn phòng ${invoice.room_name}`}
          title="Sửa hóa đơn"
        >
          <Pencil data-icon="inline-start" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon-lg"
          onClick={onDownload}
          className="size-11 cursor-pointer"
          aria-label={`Tải hóa đơn phòng ${invoice.room_name}`}
          title="Tải hóa đơn"
        >
          <Download data-icon="inline-start" />
        </Button>
        {canSendZalo ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-lg"
            onClick={onSendZalo}
            className="size-11 cursor-pointer"
            aria-label={`Gửi hóa đơn phòng ${invoice.room_name} qua Zalo`}
            title="Gửi qua Zalo"
          >
            <Send data-icon="inline-start" />
          </Button>
        ) : null}
        <Button
          type="button"
          variant="ghost"
          size="icon-lg"
          onClick={onDelete}
          className="size-11 cursor-pointer"
          aria-label={`Xóa hóa đơn phòng ${invoice.room_name}`}
          title="Xóa hóa đơn"
        >
          <Trash2 data-icon="inline-start" />
        </Button>
      </CardFooter>
    </Card>
  )
}
