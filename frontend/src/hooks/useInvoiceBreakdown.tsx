import { useEffect, useState } from 'react'
import { getInvoiceBreakdown, type InvoiceBreakdown } from '@/api/invoice'

export function useInvoiceBreakdown(houseIds: string[], period: string, refreshKey?: unknown) {
  const [result, setResult] = useState<{ key: string; data: InvoiceBreakdown | null } | null>(null)
  const houseKey = houseIds.join(',')
  const requestKey = `${houseKey}|${period}`
  const hasInput = Boolean(houseKey && period)

  useEffect(() => {
    if (!hasInput) return
    let cancelled = false
    getInvoiceBreakdown({ house_ids: houseKey, period })
      .catch(() => null)
      .then(data => {
        if (!cancelled) setResult({ key: `${houseKey}|${period}`, data })
      })
    return () => { cancelled = true }
  }, [houseKey, period, hasInput, refreshKey])

  const isCurrent = result?.key === requestKey
  return {
    breakdown: hasInput && isCurrent ? result.data : null,
    isLoading: hasInput && !isCurrent,
  }
}
