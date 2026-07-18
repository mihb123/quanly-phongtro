import type { ComponentType } from 'react'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'

// Tông màu của stat card — ánh xạ sang token ngữ nghĩa, không hardcode bg-*-100.
type StatTone = 'default' | 'positive' | 'negative' | 'warning' | 'info'

const iconChipClass: Record<StatTone, string> = {
  default: 'bg-muted text-muted-foreground',
  positive: 'bg-success/15 text-success',
  negative: 'bg-destructive/10 text-destructive',
  warning: 'bg-warning/15 text-warning',
  info: 'bg-info/15 text-info',
}

// Giá trị giữ màu foreground trung tính (tone chỉ thể hiện qua icon chip);
// riêng negative giữ màu destructive vì là cảnh báo nghiệp vụ thực sự.
const valueClass: Record<StatTone, string> = {
  default: 'text-foreground',
  positive: 'text-foreground',
  negative: 'text-destructive',
  warning: 'text-foreground',
  info: 'text-foreground',
}

interface StatCardProps {
  label: string
  value: string
  icon: ComponentType<{ className?: string }>
  tone?: StatTone
  sub?: React.ReactNode
  onClick?: () => void
}

// Thẻ thống kê dùng chung cho Dashboard/Revenue: gộp 5 bản inline trùng lặp, thống nhất theo token.
export function StatCard({ label, value, icon: Icon, tone = 'default', sub, onClick }: StatCardProps) {
  const clickable = typeof onClick === 'function'
  return (
    <Card
      size="sm"
      onClick={onClick}
      className={cn(
        'gap-2 p-4 sm:p-5',
        clickable && 'cursor-pointer transition-colors hover:bg-secondary/30',
      )}
    >
      <div className="flex items-center gap-2 sm:gap-3">
        <span className={cn('flex size-8 shrink-0 items-center justify-center rounded-full', iconChipClass[tone])}>
          <Icon className="size-4" />
        </span>
        <h3 className="line-clamp-1 text-xs font-medium text-muted-foreground">
          {label}
        </h3>
      </div>
      <p className={cn('truncate text-lg font-semibold tabular-nums sm:text-2xl', valueClass[tone])}>{value}</p>
      {sub ? <div className="text-xs text-muted-foreground">{sub}</div> : null}
    </Card>
  )
}
