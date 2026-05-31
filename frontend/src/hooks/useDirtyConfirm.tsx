import { useState, useCallback, useEffect } from 'react'
import { ConfirmModal } from '@/components/home/modals/ConfirmModal'

export function useDirtyConfirm(isDirty: boolean, onClose: () => void, isLoading: boolean = false) {
  const [showConfirmClose, setShowConfirmClose] = useState(false)

  const handleClose = useCallback(() => {
    if (isDirty) {
      setShowConfirmClose(true)
    } else {
      onClose()
    }
  }, [isDirty, onClose])

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !isLoading && !showConfirmClose) handleClose()
    }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [handleClose, isLoading, showConfirmClose])

  const confirmModal = showConfirmClose ? (
    <ConfirmModal
      title="Bạn có chắc chắn muốn thoát?"
      message="Các thay đổi của bạn sẽ không được lưu lại."
      confirmText="Thoát không lưu"
      cancelText="Tiếp tục chỉnh sửa"
      onConfirm={onClose}
      onCancel={() => setShowConfirmClose(false)}
    />
  ) : null;

  return { handleClose, confirmModal }
}
