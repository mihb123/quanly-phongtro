import type { ComponentProps, ReactNode } from 'react'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { X } from '@/components/icons'
import { cn } from '@/lib/utils'
import { useBackToClose } from './useBackToClose'

interface AppModalProps {
  open: boolean
  onClose: () => void
  title: ReactNode
  description?: ReactNode
  children: ReactNode
  footer?: ReactNode
  /** Override độ rộng (vd: 'sm:max-w-lg', 'sm:max-w-2xl'). */
  contentClassName?: string
  /** Cho phép đóng khi đang submit (mặc định cho phép). */
  dismissible?: boolean
  /** Quản lý phần tử được focus khi mở modal (truyền false để không tự động focus vào input đầu tiên). */
  initialFocus?: ComponentProps<typeof DialogContent>['initialFocus']
}

// Vỏ modal form dùng chung cho 20 modal: chuẩn hóa header + body cuộn + footer dính,
// đóng bằng Esc/click nền/nút X (do Dialog base-ui lo). Form giữ nguyên logic RHF/Zod bên trong children.
export function AppModal({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  contentClassName,
  dismissible = true,
  initialFocus,
}: AppModalProps) {
  useBackToClose(open, onClose, dismissible)

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && dismissible) onClose()
      }}
    >
      <DialogContent
        initialFocus={initialFocus}
        showCloseButton={false}
        className={cn(
          'flex flex-col gap-0 overflow-hidden p-0 sm:max-h-[90dvh]',
          'max-sm:inset-0 max-sm:h-dvh max-sm:max-h-dvh max-sm:max-w-none max-sm:translate-x-0 max-sm:translate-y-0 max-sm:rounded-none max-sm:ring-0 max-sm:data-open:zoom-in-100 max-sm:data-closed:zoom-out-100',
          contentClassName,
        )}
      >
        <DialogHeader className="shrink-0 flex-row items-start gap-3 border-b border-border/60 px-4 pb-3 pt-[calc(0.75rem+env(safe-area-inset-top))] sm:px-6 sm:pb-4 sm:pt-6">
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <DialogTitle className="text-lg">{title}</DialogTitle>
            {description ? <DialogDescription>{description}</DialogDescription> : null}
          </div>
          <DialogClose
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                className="-mr-1 -mt-1 shrink-0 bg-secondary sm:-mr-2 sm:-mt-2"
                aria-label="Đóng"
              />
            }
          >
            <X />
          </DialogClose>
        </DialogHeader>
        <div className="flex-1 overflow-y-auto overscroll-contain px-4 py-4 sm:px-6">{children}</div>
        {footer ? (
          <DialogFooter className="shrink-0 border-t border-border/60 px-4 pt-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] sm:px-6 sm:py-4">
            {footer}
          </DialogFooter>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
