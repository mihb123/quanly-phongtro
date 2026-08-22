import { useId, useState } from 'react'
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { AlertTriangle } from '@/components/icons'

interface ConfirmDialogProps {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  onConfirm: () => void
  onCancel: () => void
  isLoading?: boolean
  /** Bắt gõ đúng chuỗi này mới mở khóa nút xác nhận — dùng cho thao tác xóa vĩnh viễn, chống bấm/chạm nhầm. */
  confirmPhrase?: string
}

// So khớp bỏ qua hoa/thường và khoảng trắng thừa: đủ chặn thao tác nhầm mà không bắt gõ lại y hệt dấu.
const normalize = (value: string) => value.trim().toLocaleLowerCase('vi')

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
  confirmPhrase,
}: ConfirmDialogProps) {
  const inputId = useId()
  const [typed, setTyped] = useState('')

  const isUnlocked = !confirmPhrase || normalize(typed) === normalize(confirmPhrase)

  const actions = (
    <div className="flex w-full gap-3 pt-2">
      <Button type="button" variant="outline" onClick={onCancel} disabled={isLoading} className="flex-1">
        {cancelText}
      </Button>
      <Button
        type={confirmPhrase ? 'submit' : 'button'}
        variant="destructive"
        onClick={confirmPhrase ? undefined : onConfirm}
        disabled={isLoading || !isUnlocked}
        className="flex-1"
      >
        {isLoading ? 'Đang thực hiện...' : confirmText}
      </Button>
    </div>
  )

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

          {confirmPhrase ? (
            // Bọc form để Enter cũng xác nhận được; chỉ bật khi có confirmPhrase để các nơi gọi khác giữ nguyên hành vi.
            <form
              className="flex w-full flex-col gap-2"
              onSubmit={(e) => {
                e.preventDefault()
                if (isUnlocked && !isLoading) onConfirm()
              }}
            >
              <div className="space-y-1.5 text-left">
                <Label htmlFor={inputId} className="text-xs font-normal text-muted-foreground">
                  Gõ <span className="font-semibold text-foreground">{confirmPhrase}</span> để xác nhận
                </Label>
                <Input
                  id={inputId}
                  value={typed}
                  onChange={(e) => setTyped(e.target.value)}
                  disabled={isLoading}
                  placeholder={confirmPhrase}
                  autoComplete="off"
                  autoCapitalize="off"
                  spellCheck={false}
                  className="border-input"
                />
              </div>
              {actions}
            </form>
          ) : (
            actions
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
