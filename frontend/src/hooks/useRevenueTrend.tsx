import { useEffect, useState } from 'react'
import { getRevenueSummaries } from '@/api/houseCost'

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

// Hook lấy xu hướng doanh thu/chi phí N kỳ gần nhất (tổng hợp các nhà đang chọn)
// bằng cách gọi API summaries cho từng kỳ — backend chưa có endpoint time-series.
// Kết quả lưu kèm requestKey để suy ra trạng thái loading thay vì set đồng bộ trong effect.
export function useRevenueTrend(houseIds: string[], endPeriod: string, count = 6) {
  const [result, setResult] = useState<{ key: string; points: RevenueTrendPoint[] } | null>(null)
  const houseKey = houseIds.join(',')
  const requestKey = `${houseKey}|${endPeriod}|${count}`
  const hasInput = Boolean(houseKey && endPeriod)

  useEffect(() => {
    if (!hasInput) return
    let cancelled = false
    const ids = houseKey.split(',')
    const periods = buildPeriods(endPeriod, count)
    Promise.all(periods.map(p => getRevenueSummaries(ids, p).catch(() => [])))
      .then(results => {
        if (cancelled) return
        const points = periods.map((period, i) => {
          const summaries = results[i] || []
          const revenue = summaries.reduce((acc, s) => acc + s.total_revenue, 0)
          const cost = summaries.reduce((acc, s) => acc + s.total_cost, 0)
          return { period, revenue, cost, profit: revenue - cost }
        })
        setResult({ key: `${houseKey}|${endPeriod}|${count}`, points })
      })
    return () => { cancelled = true }
  }, [houseKey, endPeriod, count, hasInput])

  const isCurrent = result?.key === requestKey
  return {
    points: hasInput && isCurrent ? result.points : [],
    isLoading: hasInput && !isCurrent,
  }
}
