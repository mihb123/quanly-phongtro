import { Card } from '@/components/ui/card'

interface StatCardProps {
  label: string
  value: string
  subValue: string
}

export function StatCard({ label, value, subValue }: StatCardProps) {
  return (
    <Card className="bg-card border-border/40 shadow-sm shadow-black/5 p-6 hover:translate-y-[-2px] transition-all duration-300 hover:shadow-md group">
      <div className="flex flex-col gap-1">
        <span className="text-muted-foreground text-sm font-semibold">{label}</span>
        <span className="text-3xl font-bold text-foreground group-hover:text-primary transition-colors">
          {value}
        </span>
        <span className="text-xs text-muted-foreground mt-2 flex items-center gap-1 font-medium">
          <span className="text-emerald-500 font-bold">{subValue.split(' ')[0]}</span>
          {subValue.split(' ').slice(1).join(' ')}
        </span>
      </div>
    </Card>
  )
}
