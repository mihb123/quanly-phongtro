import type { ComponentType, ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  icon?: ComponentType<{ className?: string }>
  title: string
  description?: string
  action?: ReactNode
  className?: string
}

// Trạng thái rỗng dùng chung cho các danh sách (không có hóa đơn/phòng/khách...).
export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center gap-3 px-6 py-10 text-center', className)}>
      {Icon ? (
        <span className="flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
          <Icon className="size-6" />
        </span>
      ) : null}
      <div className="space-y-1">
        <p className="font-semibold text-foreground">{title}</p>
        {description ? <p className="text-sm text-muted-foreground">{description}</p> : null}
      </div>
      {action}
    </div>
  )
}
