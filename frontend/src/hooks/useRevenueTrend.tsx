import { useMemo } from 'react'
import { getRevenueTrend } from '@/api/houseCost'
import { queryKey, useQuery } from '@/lib/queryCache'

export interface RevenueTrendPoint {
  period: string // yyyy-mm
  revenue: number
  cost: number
  profit: number
}

// Sinh danh sách `count` kỳ (yyyy-mm) liên tiếp, kết thúc tại endPeriod (tăng dần).
function buildPeriods(endPeriod: string, count: number): string[] {
  const [year, month] = endPeriod.split('-').map(Number)
  if (!year || !month) return []
  const periods: string[] = []
  for (let i = count - 1; i >= 0; i--) {
    const d = new Date(year, month - 1 - i, 1)
    periods.push(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`)
  }
  return periods
}

// Xu hướng doanh thu/chi phí N kỳ gần nhất của các nhà đang chọn, lấy bằng 1 request nhiều kỳ.
export function useRevenueTrend(houseIds: string[], endPeriod: string, count = 6) {
  const houseKey = houseIds.join(',')
  const periods = useMemo(() => buildPeriods(endPeriod, count), [endPeriod, count])
  const hasInput = Boolean(houseKey && periods.length)

  const { data, isLoading } = useQuery(
    queryKey('revenue:trend', { house_ids: houseKey, periods: periods.join(',') }),
    () => getRevenueTrend(houseKey.split(','), periods),
    { enabled: hasInput },
  )

  const points = useMemo<RevenueTrendPoint[]>(() => {
    if (!data) return []
    return periods.map(period => {
      const rows = data.filter(s => s.period === period)
      const revenue = rows.reduce((acc, s) => acc + s.total_revenue, 0)
      const cost = rows.reduce((acc, s) => acc + s.total_cost, 0)
      return { period, revenue, cost, profit: revenue - cost }
    })
  }, [data, periods])

  return { points: hasInput ? points : [], isLoading: hasInput && isLoading }
}
