import { Calendar, CheckCircle2, Copy, DoorOpen, Phone } from '@/components/icons'
import { type Tenant } from '@/api/tenant'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'

interface TenantMobileCardProps {
  tenant: Tenant
  onEdit: () => void
  onOpenRoom: () => void
  onCopyPhone: () => void
}

// Badge trạng thái đang ở dùng chung cho danh sách khách thuê trên mobile và desktop.
export function StayingBadge() {
  return (
    <Badge variant="ghost" className="bg-success/10 font-medium text-success">
      <CheckCircle2 /> Đang ở
    </Badge>
  )
}

// Thẻ khách thuê mobile ưu tiên tên, phòng và số điện thoại với vùng chạm dễ thao tác.
export function TenantMobileCard({ tenant, onEdit, onOpenRoom, onCopyPhone }: TenantMobileCardProps) {
  return (
    <li className="rounded-lg bg-secondary/25 ring-1 ring-foreground/10">
      <div className="flex items-center gap-2 px-3 pt-2">
        <Button
          type="button"
          variant="ghost"
          size="lg"
          onClick={onEdit}
          className="min-w-0 flex-1 cursor-pointer justify-start px-0"
          aria-label={`Chỉnh sửa khách thuê ${tenant.full_name}`}
        >
          <span className="truncate text-base font-semibold text-foreground">{tenant.full_name}</span>
        </Button>
        <StayingBadge />
      </div>

      <div className="flex items-center gap-2 px-3 pb-2 text-sm text-muted-foreground">
        <Calendar className="size-4 shrink-0" aria-hidden="true" />
        <span>Bắt đầu {new Date(tenant.start_date).toLocaleDateString('vi-VN')}</span>
      </div>

      <Separator />

      <div className="grid grid-cols-2 gap-2 p-2">
        <Button
          type="button"
          variant="outline"
          size="lg"
          onClick={onOpenRoom}
          className="min-w-0 cursor-pointer justify-start"
          aria-label={`Xem phòng ${tenant.room_name || 'chưa xác định'}`}
        >
          <DoorOpen data-icon="inline-start" />
          <span className="truncate">Phòng {tenant.room_name || 'N/A'}</span>
        </Button>
        <Button
          type="button"
          variant="outline"
          size="lg"
          onClick={onCopyPhone}
          className="min-w-0 cursor-pointer justify-start"
          aria-label={`Sao chép số điện thoại ${tenant.phone}`}
        >
          <Phone data-icon="inline-start" />
          <span className="truncate tabular-nums">{tenant.phone}</span>
          <Copy data-icon="inline-end" />
        </Button>
      </div>
    </li>
  )
}
