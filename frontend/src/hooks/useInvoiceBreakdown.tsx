import { useEffect } from 'react'
import { getInvoiceBreakdown } from '@/api/invoice'
import { invalidateQueries, queryKey, useQuery } from '@/lib/queryCache'

export function useInvoiceBreakdown(houseIds: string[], period: string, refreshKey?: unknown) {
  const houseKey = houseIds.join(',')
  const hasInput = Boolean(houseKey && period)
  const key = queryKey('invoices:breakdown-houses', { house_ids: houseKey, period })

  const { data, isLoading } = useQuery(key, () => getInvoiceBreakdown({ house_ids: houseKey, period }), { enabled: hasInput })

  useEffect(() => {
    if (refreshKey !== undefined) invalidateQueries(key)
  }, [refreshKey, key])

  return {
    breakdown: hasInput ? data ?? null : null,
    isLoading: hasInput && isLoading,
  }
}
