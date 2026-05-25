import { Card } from '@/components/ui/card'

interface StatCardProps {
  label: string
  value: string
  subValue: string
}

export function StatCard({ label, value, subValue }: StatCardProps) {
  return (
    <Card className="bg-white/80 border-slate-200/60 shadow-md shadow-slate-200/30 backdrop-blur-2xl p-6 hover:translate-y-[-4px] transition-all duration-300 hover:shadow-xl hover:shadow-slate-200/50 group">
      <div className="flex flex-col gap-1">
        <span className="text-slate-500 text-sm font-semibold">{label}</span>
        <span className="text-3xl font-bold bg-gradient-to-r from-slate-800 to-slate-600 bg-clip-text text-transparent group-hover:from-purple-600 group-hover:to-indigo-500">
          {value}
        </span>
        <span className="text-xs text-slate-500 mt-2 flex items-center gap-1 font-medium">
          <span className="text-emerald-500 font-bold">{subValue.split(' ')[0]}</span>
          {subValue.split(' ').slice(1).join(' ')}
        </span>
      </div>
    </Card>
  )
}
