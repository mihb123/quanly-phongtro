import { cn } from '@/lib/utils'

interface VerifiedBadgeProps {
  /** Nội dung tooltip khi hover. */
  title?: string
  className?: string
}

// Tick xác thực kiểu Facebook (viền răng cưa + dấu check khoét rỗng), dùng token `success` nên đổi theme là tự đổi màu.
export function VerifiedBadge({ title = 'Đã xác thực', className }: VerifiedBadgeProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      role="img"
      aria-label={title}
      className={cn('size-4 shrink-0 text-success', className)}
    >
      <title>{title}</title>
      <path
        fill="currentColor"
        d="M23 12l-2.44-2.79.34-3.69-3.61-.82-1.89-3.2L12 2.96 8.6 1.5 6.71 4.69 3.1 5.5l.34 3.7L1 12l2.44 2.79-.34 3.7 3.61.82L8.6 22.5l3.4-1.47 3.4 1.46 1.89-3.19 3.61-.82-.34-3.69L23 12z"
      />
      <path
        className="fill-background"
        d="M10.09 16.72l-3.8-3.81 1.48-1.48 2.32 2.33 5.34-5.34 1.48 1.48-6.82 6.82z"
      />
    </svg>
  )
}
