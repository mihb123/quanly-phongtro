import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

// Tông màu trạng thái nghiệp vụ — ánh xạ sang token ngữ nghĩa trong index.css (không hardcode màu).
type StatusTone = 'success' | 'warning' | 'info' | 'destructive' | 'neutral'

const toneClass: Record<StatusTone, string> = {
  success: 'bg-success/10 text-success',
  warning: 'bg-warning/10 text-warning',
  info: 'bg-info/10 text-info',
  destructive: 'bg-destructive/10 text-destructive',
  neutral: 'bg-secondary text-secondary-foreground',
}

// Nhãn + tông cho trạng thái hóa đơn (PAID/UNPAID/PENDING_VERIFICATION) và phòng (AVAILABLE/OCCUPIED).
const STATUS_MAP: Record<string, { label: string; tone: StatusTone }> = {
  PAID: { label: 'Đã thu', tone: 'success' },
  UNPAID: { label: 'Chưa thu', tone: 'warning' },
  PENDING_VERIFICATION: { label: 'Chờ xác nhận', tone: 'info' },
  AVAILABLE: { label: 'Phòng trống', tone: 'neutral' },
  OCCUPIED: { label: 'Đang thuê', tone: 'success' },
}

interface StatusBadgeProps {
  status: string
  className?: string
}

// Badge trạng thái dùng chung: tự suy nhãn tiếng Việt + tông màu từ mã trạng thái.
export function StatusBadge({ status, className }: StatusBadgeProps) {
  const conf = STATUS_MAP[status] ?? { label: status, tone: 'neutral' as StatusTone }
  return (
    <Badge variant="ghost" className={cn('font-semibold', toneClass[conf.tone], className)}>
      {conf.label}
    </Badge>
  )
}
