import { useState } from 'react'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'

interface EditNoteModalProps {
  initialNote: string
  onSave: (note: string) => void
  onClose: () => void
}

// Modal chỉnh sửa ghi chú. Vỏ dùng AppModal (Esc/click nền/X tự xử lý), giữ nội dung note ở state cục bộ.
export function EditNoteModal({ initialNote, onSave, onClose }: EditNoteModalProps) {
  const [note, setNote] = useState(initialNote)

  return (
    <AppModal
      open
      onClose={onClose}
      title="Chỉnh sửa ghi chú"
      footer={
        <>
          <Button variant="outline" onClick={onClose} className="flex-1 sm:flex-none">
            Hủy
          </Button>
          <Button onClick={() => onSave(note)} className="flex-1 sm:flex-none">
            Hoàn tất
          </Button>
        </>
      }
    >
      <Textarea
        autoFocus
        value={note}
        onChange={(e) => setNote(e.target.value)}
        placeholder="Nhập nội dung ghi chú chi tiết..."
        className="h-40"
      />
    </AppModal>
  )
}
