import React, { useState } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { updateRoom, type Room } from '@/api/room'
import type { House } from '@/api/house'
import { formatNumber, parseNumber } from '@/utils/format'

export function EditRoomModal({ house, room, onClose, onSuccess }: { house: House, room: Room, onClose: () => void, onSuccess: () => void }) {
  const [name, setName] = useState(room.name || '')
  const [price, setPrice] = useState(room.price?.toString() || '0')
  const [maxTennants, setMaxTennants] = useState(room.max_tennants?.toString() || '2')
  
  const [electricity, setElectricity] = useState(room.electricity_price?.toString() || house.default_electricity_price?.toString() || '')
  const [water, setWater] = useState(room.water_price?.toString() || house.default_water_price?.toString() || '')
  const [wifi, setWifi] = useState(room.wifi_price?.toString() || house.default_wifi_price?.toString() || '')
  const [parking, setParking] = useState(room.parking_price?.toString() || house.default_parking_price?.toString() || '')
  const [service, setService] = useState(room.service_price?.toString() || house.default_service_price?.toString() || '')
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    try {
      await updateRoom(room.id, { 
        house_id: house.id, 
        name, 
        price: parseNumber(price), 
        max_tennants: Number(maxTennants),
        electricity_price: electricity !== '' ? parseNumber(electricity) : undefined,
        water_price: water !== '' ? parseNumber(water) : undefined,
        wifi_price: wifi !== '' ? parseNumber(wifi) : undefined,
        parking_price: parking !== '' ? parseNumber(parking) : undefined,
        service_price: service !== '' ? parseNumber(service) : undefined,
      })
      onSuccess()
    } catch(err) {
      alert("Lỗi khi cập nhật phòng, vui lòng kiểm tra lại thông tin!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 overflow-y-auto pt-20 pb-20">
      <Card className="w-full max-w-xl p-6 bg-white shadow-xl border-0 animate-in zoom-in-95 duration-200">
        <h2 className="text-xl font-bold mb-4 text-slate-800">Sửa thông tin phòng</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 col-span-2">
              <Label>Tên phòng</Label>
              <Input required value={name} onChange={e => setName(e.target.value)} className="border-slate-200" />
            </div>
            <div className="space-y-2">
              <Label>Giá thuê hàng tháng (VNĐ)</Label>
              <Input required type="text" value={formatNumber(price)} onChange={e => setPrice(e.target.value.replace(/\D/g, ''))} className="border-slate-200" />
            </div>
            <div className="space-y-2">
              <Label>Số người ở tối đa</Label>
              <Input required type="number" min="1" value={maxTennants} onChange={e => setMaxTennants(e.target.value)} className="border-slate-200" />
            </div>
          </div>
          
          <hr className="my-2 border-slate-100" />
          <div className="bg-slate-50 p-4 rounded-xl border border-slate-100">
             <h3 className="font-bold text-slate-700 text-sm">Tuỳ chỉnh giá phát sinh riêng</h3>
             <p className="text-xs text-slate-500 mb-3">Nếu để trống, hệ thống sẽ tự động dùng giá mặc định của nhà trọ.</p>
             <div className="grid grid-cols-2 md:grid-cols-3 gap-3 text-sm">
                <div className="space-y-1">
                  <Label className="text-xs">Giá điện / số</Label>
                  <Input type="text" placeholder="Mặc định..." value={formatNumber(electricity)} onChange={e => setElectricity(e.target.value.replace(/\D/g, ''))} className="h-8 border-slate-200 bg-white" />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Giá nước</Label>
                  <Input type="text" placeholder="Mặc định..." value={formatNumber(water)} onChange={e => setWater(e.target.value.replace(/\D/g, ''))} className="h-8 border-slate-200 bg-white" />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Giá Wifi</Label>
                  <Input type="text" placeholder="Mặc định..." value={formatNumber(wifi)} onChange={e => setWifi(e.target.value.replace(/\D/g, ''))} className="h-8 border-slate-200 bg-white" />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Giá gửi xe</Label>
                  <Input type="text" placeholder="Mặc định..." value={formatNumber(parking)} onChange={e => setParking(e.target.value.replace(/\D/g, ''))} className="h-8 border-slate-200 bg-white" />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Giá dịch vụ chung</Label>
                  <Input type="text" placeholder="Mặc định..." value={formatNumber(service)} onChange={e => setService(e.target.value.replace(/\D/g, ''))} className="h-8 border-slate-200 bg-white" />
                </div>
             </div>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={onClose} className="border-slate-200 text-slate-600">Hủy</Button>
            <Button type="submit" disabled={isLoading} className="bg-purple-600 text-white hover:bg-purple-700 shadow-md">
              {isLoading ? 'Đang lưu...' : 'Lưu thay đổi'}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
