import { useState } from 'react'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { checkoutRoom } from '@/api/tenant'
import type { Room } from '@/api/room'

interface CheckoutRoomDialogProps {
  room: Room
  onCancel: () => void
  onDone: () => void
}

export function CheckoutRoomDialog({ room, onCancel, onDone }: CheckoutRoomDialogProps) {
  const [isLoading, setIsLoading] = useState(false)

  const handleConfirm = async () => {
    setIsLoading(true)
    try {
      const { removed } = await checkoutRoom(room.id)
      toast.success(`Đã trả phòng ${room.name}`, {
        description: `Đã xoá ${removed} người thuê. Hóa đơn tạo sau đây sẽ không tính tiền phòng.`,
      })
      onDone()
    } catch (error: unknown) {
      const err = error as { response?: { data?: { message?: string } } }
      toast.error(err.response?.data?.message || 'Không thể trả phòng')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <ConfirmDialog
      title={`Trả phòng ${room.name}`}
      message="Xoá toàn bộ người thuê đang ở và chuyển phòng về trạng thái trống. Hóa đơn ghi điện nước sau khi trả phòng sẽ không tính tiền phòng."
      confirmText="Trả phòng"
      cancelText="Bỏ qua"
      isLoading={isLoading}
      onConfirm={handleConfirm}
      onCancel={onCancel}
    />
  )
}
