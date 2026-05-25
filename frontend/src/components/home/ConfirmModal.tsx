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
  return (
    <div className="fixed inset-0 z-[110] flex items-center justify-center bg-black/40 backdrop-blur-sm px-4">
      <Card className="w-full max-w-md p-6 bg-white shadow-2xl border-0 animate-in zoom-in-95 duration-200">
        <div className="flex flex-col items-center text-center space-y-4">
          <div className="w-16 h-16 rounded-full bg-red-50 flex items-center justify-center">
            <AlertTriangle className="w-8 h-8 text-red-500" />
          </div>
          
          <div className="space-y-2">
            <h2 className="text-xl font-bold text-slate-800">{title}</h2>
            <p className="text-slate-500 text-sm">{message}</p>
          </div>

          <div className="flex gap-3 w-full pt-4">
            <Button
              variant="outline"
              onClick={onCancel}
              className="flex-1 border-slate-200 text-slate-600 font-bold h-11 rounded-xl"
              disabled={isLoading}
            >
              {cancelText}
            </Button>
            <Button
              onClick={onConfirm}
              className="flex-1 bg-red-500 hover:bg-red-600 text-white font-bold h-11 rounded-xl shadow-lg shadow-red-200 transition-all active:scale-95"
              disabled={isLoading}
            >
              {isLoading ? 'Đang thực hiện...' : confirmText}
            </Button>
          </div>
        </div>
      </Card>
    </div>
  )
}
