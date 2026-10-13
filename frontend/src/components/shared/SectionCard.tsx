import type { ComponentType, ReactNode } from 'react'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'

interface SectionCardProps {
  title: ReactNode
  icon?: ComponentType<{ className?: string }>
  action?: ReactNode
  children: ReactNode
  className?: string
  headerClassName?: string
  bodyClassName?: string
}

// Card có header (title + icon + action) và body — thay khối "list card" lặp trong các view.
export function SectionCard({ title, icon: Icon, action, children, className, headerClassName, bodyClassName }: SectionCardProps) {
  return (
    <Card className={cn('gap-0 p-0', className)}>
      <div className={cn('flex items-center justify-between gap-2 border-b px-5 py-3.5', headerClassName)}>
        <h3 className="flex items-center gap-2 text-sm font-semibold text-foreground">
          {Icon ? <Icon className="size-4 text-muted-foreground" /> : null}
          {title}
        </h3>
        {action}
      </div>
      <div className={cn(bodyClassName)}>{children}</div>
    </Card>
  )
}
