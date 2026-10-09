import { useId } from 'react'
import { ChevronLeft, ChevronRight } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { PAGE_SIZE_OPTIONS } from '@/lib/storage'
import { cn } from '@/lib/utils'

interface DataPaginationProps {
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
  className?: string
}

function visiblePages(page: number, pageCount: number): (number | 'gap')[] {
  if (pageCount <= 5) return Array.from({ length: pageCount }, (_, i) => i + 1)
  const pages = new Set([1, pageCount, page - 1, page, page + 1].filter(p => p >= 1 && p <= pageCount))
  const sorted = [...pages].sort((a, b) => a - b)
  return sorted.flatMap((p, i) => (i > 0 && p - sorted[i - 1] > 1 ? ['gap' as const, p] : [p]))
}

export function DataPagination({ page, pageSize, total, onPageChange, onPageSizeChange, className }: DataPaginationProps) {
  const sizeId = useId()
  const pageCount = Math.max(1, Math.ceil(total / pageSize))
  const from = total === 0 ? 0 : (page - 1) * pageSize + 1
  const to = Math.min(total, page * pageSize)

  if (total <= PAGE_SIZE_OPTIONS[0] && page <= 1) return null
  const showPageButtons = pageCount > 1 || page > 1

  return (
    <nav
      aria-label="Phân trang"
      className={cn('flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between', className)}
    >
      <div className="flex items-center justify-between gap-3 text-sm text-muted-foreground sm:justify-start">
        <span className="tabular-nums">
          Hiển thị {from.toLocaleString('vi-VN')}–{to.toLocaleString('vi-VN')} / {total.toLocaleString('vi-VN')}
        </span>
        <label htmlFor={sizeId} className="flex items-center gap-2">
          <span className="sr-only sm:not-sr-only">Mỗi trang</span>
          <select
            id={sizeId}
            value={pageSize}
            onChange={e => onPageSizeChange(Number(e.target.value))}
            className="h-9 cursor-pointer rounded-md border border-input bg-background px-2 text-base text-foreground outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30 pointer-coarse:h-11 md:text-sm"
          >
            {PAGE_SIZE_OPTIONS.map(size => (
              <option key={size} value={size}>{size} / trang</option>
            ))}
          </select>
        </label>
      </div>

      {showPageButtons && (
        <div className="flex items-center justify-between gap-1 sm:justify-end">
          <Button
            variant="outline"
            size="icon"
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
            aria-label="Trang trước"
          >
            <ChevronLeft />
          </Button>
          <div className="flex items-center gap-1">
            {visiblePages(page, pageCount).map((p, i) =>
              p === 'gap' ? (
                <span key={`gap-${i}`} className="px-1 text-muted-foreground" aria-hidden>…</span>
              ) : (
                <Button
                  key={p}
                  variant={p === page ? 'secondary' : 'ghost'}
                  size="icon"
                  onClick={() => onPageChange(p)}
                  aria-label={`Trang ${p}`}
                  aria-current={p === page ? 'page' : undefined}
                  className="tabular-nums"
                >
                  {p}
                </Button>
              ),
            )}
          </div>
          <Button
            variant="outline"
            size="icon"
            onClick={() => onPageChange(page + 1)}
            disabled={page >= pageCount}
            aria-label="Trang sau"
          >
            <ChevronRight />
          </Button>
        </div>
      )}
    </nav>
  )
}
