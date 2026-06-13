import { createPortal } from 'react-dom'
import { useEffect, useState } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { X, FileText } from 'lucide-react'

interface EditNoteModalProps {
  initialNote: string
  onSave: (note: string) => void
  onClose: () => void
}

export function EditNoteModal({ 
  initialNote, 
  onSave, 
  onClose 
}: EditNoteModalProps) {
  const [note, setNote] = useState(initialNote)

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [onClose])

  return createPortal(
    <div 
      className="fixed inset-0 z-[110] flex items-center justify-center bg-background/80 backdrop-blur-sm px-4 p-4 sm:p-0"
      onMouseDown={e => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <Card className="w-full max-w-md p-6 bg-card text-card-foreground shadow-2xl border border-border/40 safe-fade-in max-h-[90vh] flex flex-col gap-4">
        <div className="flex justify-between items-center pb-2 border-b">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
              <FileText className="w-4 h-4 text-primary" />
            </div>
            <h2 className="text-xl font-bold text-foreground">Chỉnh sửa ghi chú</h2>
          </div>
          <Button variant="ghost" size="icon" onClick={onClose} className="h-8 w-8 rounded-full hover:bg-secondary">
            <X className="w-4 h-4" />
          </Button>
        </div>

        <div className="flex-1 py-2">
          <textarea
            autoFocus
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="Nhập nội dung ghi chú chi tiết..."
            className="w-full h-40 p-3 font-medium rounded-xl border border-input bg-background text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary resize-none"
          />
        </div>

        <div className="flex gap-3 pt-2">
          <Button
            variant="outline"
            onClick={onClose}
            className="flex-1 font-bold h-11 rounded-xl"
          >
            Hủy
          </Button>
          <Button
            onClick={() => onSave(note)}
            className="flex-1 font-bold h-11 rounded-xl shadow-lg transition-all active:scale-95"
          >
            Hoàn tất
          </Button>
        </div>
      </Card>
    </div>
  , document.body)
}
