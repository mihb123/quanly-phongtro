import { createPortal } from 'react-dom'
import { useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { AlertTriangle } from 'lucide-react'

interface ConfirmModalProps {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  onConfirm: () => void
  onCancel: () => void
  isLoading?: boolean
}

export function ConfirmModal({ 
  title, 
  message, 
  confirmText = "Xác nhận", 
  cancelText = "Hủy", 
  onConfirm, 
  onCancel,
  isLoading = false
}: ConfirmModalProps) {
  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isLoading) onCancel()
    }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [onCancel, isLoading])

  return createPortal(
    <div 
      className="fixed inset-0 z-[110] flex items-center justify-center bg-background/80 backdrop-blur-sm px-4 p-4 sm:p-0"
      onMouseDown={e => {
        if (e.target === e.currentTarget && !isLoading) onCancel()
      }}
    >
      <Card className="w-full max-w-md p-6 bg-card text-card-foreground shadow-2xl border border-border/40 safe-fade-in max-h-[90vh] overflow-y-auto">
        <div className="flex flex-col items-center text-center space-y-4">
          <div className="w-16 h-16 rounded-full bg-red-50 flex items-center justify-center">
            <AlertTriangle className="w-8 h-8 text-red-500" />
          </div>
          
          <div className="space-y-2">
            <h2 className="text-xl font-bold text-foreground">{title}</h2>
            <p className="text-muted-foreground text-sm">{message}</p>
          </div>

          <div className="flex gap-3 w-full pt-4">
            <Button
              variant="outline"
              onClick={onCancel}
              className="flex-1 font-bold h-11 rounded-xl"
              disabled={isLoading}
            >
              {cancelText}
            </Button>
            <Button
              onClick={onConfirm}
              variant="destructive"
              className="flex-1 font-bold h-11 rounded-xl shadow-lg transition-all active:scale-95"
              disabled={isLoading}
            >
              {isLoading ? 'Đang thực hiện...' : confirmText}
            </Button>
          </div>
        </div>
      </Card>
    </div>
  , document.body)
}
