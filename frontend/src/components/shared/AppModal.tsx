import type { ReactNode } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'

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
}: AppModalProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && dismissible) onClose()
      }}
    >
      <DialogContent className={cn('flex max-h-[90vh] flex-col gap-0 overflow-hidden p-0', contentClassName)}>
        <DialogHeader className="border-b border-border/60 px-6 pb-4 pt-6">
          <DialogTitle className="text-lg">{title}</DialogTitle>
          {description ? <DialogDescription>{description}</DialogDescription> : null}
        </DialogHeader>
        <div className="flex-1 overflow-y-auto px-6 py-4">{children}</div>
        {footer ? (
          <DialogFooter className="border-t border-border/60 px-6 py-4">{footer}</DialogFooter>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}
