import React, { useState } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { createRoom } from '@/api/room'

export function CreateRoomModal({ houseId, onClose, onSuccess }: { houseId: string, onClose: () => void, onSuccess: () => void }) {
  const [name, setName] = useState('')
  const [price, setPrice] = useState('0')
  const [maxTennants, setMaxTennants] = useState('2')
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    try {
      await createRoom({ 
        house_id: houseId, 
        name, 
        price: Number(price), 
        max_tennants: Number(maxTennants),
        status: 'AVAILABLE'
      })
      onSuccess()
    } catch(err) {
      alert("Lỗi khi thêm phòng, vui lòng kiểm tra lại thông tin!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4">
      <Card className="w-full max-w-sm p-6 bg-white shadow-xl border-0 animate-in zoom-in-95 duration-200">
        <h2 className="text-xl font-bold mb-4 text-slate-800">Thêm phòng mới</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label>Tên phòng</Label>
            <Input required value={name} onChange={e => setName(e.target.value)} placeholder="vd: Phòng 101" className="border-slate-200" />
          </div>
          <div className="space-y-2">
            <Label>Giá thuê (VNĐ)</Label>
            <Input required type="number" min="0" value={price} onChange={e => setPrice(e.target.value)} className="border-slate-200" />
          </div>
          <div className="space-y-2">
            <Label>Số khách thuê tối đa</Label>
            <Input required type="number" min="1" value={maxTennants} onChange={e => setMaxTennants(e.target.value)} className="border-slate-200" />
          </div>
          <div className="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={onClose} className="border-slate-200 text-slate-600">Hủy</Button>
            <Button type="submit" disabled={isLoading} className="bg-purple-600 text-white hover:bg-purple-700 shadow-md">
              {isLoading ? 'Đang tạo...' : 'Tạo mới'}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
