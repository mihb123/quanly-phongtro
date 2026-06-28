import type { ComponentType, ReactNode } from 'react'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'

interface SectionCardProps {
  title: ReactNode
  icon?: ComponentType<{ className?: string }>
  action?: ReactNode
  children: ReactNode
  className?: string
  bodyClassName?: string
}

// Card có header (title + icon + action) và body — thay khối "list card" lặp trong các view.
export function SectionCard({ title, icon: Icon, action, children, className, bodyClassName }: SectionCardProps) {
  return (
    <Card className={cn('gap-0 p-0', className)}>
      <div className="flex items-center justify-between gap-2 border-b border-border/60 px-5 py-4">
        <h3 className="flex items-center gap-2 font-bold text-foreground">
          {Icon ? <Icon className="size-4 text-muted-foreground" /> : null}
          {title}
        </h3>
        {action}
      </div>
      <div className={cn(bodyClassName)}>{children}</div>
    </Card>
  )
}
