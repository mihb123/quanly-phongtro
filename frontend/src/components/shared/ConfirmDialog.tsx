import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { AlertTriangle } from '@/components/icons'

interface ConfirmDialogProps {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  onConfirm: () => void
  onCancel: () => void
  isLoading?: boolean
}

// Hộp thoại xác nhận dùng chung (thay ConfirmModal tự chế). Dựng trên Dialog (đóng bằng Esc + click nền),
// giữ nguyên API cũ để nơi gọi chỉ cần đổi import. Mặc định render khi mounted nên open={true}.
export function ConfirmDialog({
  title,
  message,
  confirmText = 'Xác nhận',
  cancelText = 'Hủy',
  onConfirm,
  onCancel,
  isLoading = false,
}: ConfirmDialogProps) {
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !isLoading) onCancel()
      }}
    >
      <DialogContent showCloseButton={false} className="max-w-sm">
        <div className="flex flex-col items-center gap-4 text-center">
          <span className="flex size-16 items-center justify-center rounded-full bg-destructive/10 text-destructive">
            <AlertTriangle className="size-8" />
          </span>
          <div className="space-y-2">
            <DialogTitle className="text-xl">{title}</DialogTitle>
            <DialogDescription className="text-sm">{message}</DialogDescription>
          </div>
          <div className="flex w-full gap-3 pt-2">
            <Button variant="outline" onClick={onCancel} disabled={isLoading} className="flex-1 font-bold">
              {cancelText}
            </Button>
            <Button variant="destructive" onClick={onConfirm} disabled={isLoading} className="flex-1 font-bold">
              {isLoading ? 'Đang thực hiện...' : confirmText}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
