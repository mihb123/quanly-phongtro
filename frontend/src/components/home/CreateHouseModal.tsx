import React, { useState } from 'react'
import { Building } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { createHouse } from '@/api/house'
import { createRoom } from '@/api/room'
import { formatNumber, parseNumber } from '@/utils/format'

export function CreateHouseModal({ onClose, onSuccess }: { onClose: () => void, onSuccess: () => void }) {
  const [name, setName] = useState('')
  const [address, setAddress] = useState('')
  const [electricity, setElectricity] = useState('0')
  const [water, setWater] = useState('0')
  const [wifi, setWifi] = useState('0')
  const [parking, setParking] = useState('0')
  const [service, setService] = useState('0')
  
  const [floorCountStr, setFloorCountStr] = useState('0')
  const [roomsPerFloor, setRoomsPerFloor] = useState<Record<number, number>>({})
  const [isLoading, setIsLoading] = useState(false)

  const handleFloorCountChange = (val: string) => {
    setFloorCountStr(val);
    const count = parseInt(val) || 0;
    const newRooms = { ...roomsPerFloor };
    for (let i = 1; i <= count; i++) {
        if (newRooms[i] === undefined) newRooms[i] = 1;
    }
    setRoomsPerFloor(newRooms);
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    try {
      const house = await createHouse({ 
         name, 
         address,
         default_electricity_price: parseNumber(electricity),
         default_water_price: parseNumber(water),
         default_wifi_price: parseNumber(wifi),
         default_parking_price: parseNumber(parking),
         default_service_price: parseNumber(service)
      })

      const promises = []
      const floors = parseInt(floorCountStr) || 0
      if (floors > 0) {
        for (let i = 1; i <= floors; i++) {
           const count = roomsPerFloor[i] || 0;
           for (let j = 1; j <= count; j++) {
              const roomName = `P${i}${j < 10 ? '0' + j : j}`;
              promises.push(createRoom({
                 house_id: house.id,
                 name: roomName,
                 price: 0,
                 max_tenants: 2,
                 status: 'AVAILABLE'
              }))
           }
        }
      }
      if (promises.length > 0) {
         await Promise.all(promises);
      }
      onSuccess()
    } catch(err) {
      alert("Lỗi khi tạo nhà trọ, vui lòng kiểm tra lại!")
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 overflow-y-auto pt-20 pb-20">
      <Card className="w-full max-w-2xl p-6 bg-white shadow-xl border-0 animate-in zoom-in-95 duration-200">
        <h2 className="text-xl font-bold mb-4 text-slate-800">Tạo nhà trọ mới & Cấu hình tầng</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2 col-span-2">
              <Label>Tên nhà trọ</Label>
              <Input required value={name} onChange={e => setName(e.target.value)} placeholder="vd: Trọ Cầu Giấy" className="border-slate-200" />
            </div>
            <div className="space-y-2 col-span-2">
              <Label>Địa chỉ</Label>
              <Input required value={address} onChange={e => setAddress(e.target.value)} placeholder="Nhập địa chỉ đầy đủ" className="border-slate-200" />
            </div>
            {/* Phí mặc định */}
            <div className="space-y-2">
              <Label className="text-xs">Giá điện mặc định / số (VNĐ)</Label>
              <Input type="text" required value={formatNumber(electricity)} onChange={e => setElectricity(e.target.value.replace(/\D/g, ''))} className="border-slate-200 h-8" />
            </div>
            <div className="space-y-2">
              <Label className="text-xs">Giá nước mặc định (theo khối hoặc người)</Label>
              <Input type="text" required value={formatNumber(water)} onChange={e => setWater(e.target.value.replace(/\D/g, ''))} className="border-slate-200 h-8" />
            </div>
            <div className="space-y-2">
              <Label className="text-xs">Giá Wifi / phòng (VNĐ)</Label>
              <Input type="text" required value={formatNumber(wifi)} onChange={e => setWifi(e.target.value.replace(/\D/g, ''))} className="border-slate-200 h-8" />
            </div>
            <div className="space-y-2">
              <Label className="text-xs">Giá gửi xe / xe (VNĐ)</Label>
              <Input type="text" required value={formatNumber(parking)} onChange={e => setParking(e.target.value.replace(/\D/g, ''))} className="border-slate-200 h-8" />
            </div>
            <div className="space-y-2 col-span-2">
              <Label className="text-xs">Giá dịch vụ chung / người (VNĐ)</Label>
              <Input type="text" required value={formatNumber(service)} onChange={e => setService(e.target.value.replace(/\D/g, ''))} className="border-slate-200 h-8" />
            </div>
          </div>
          
          <hr className="my-4 border-slate-100" />
          <div className="space-y-4 bg-slate-50 rounded-xl p-4 border border-slate-100">
            <h3 className="font-bold text-slate-700 text-sm flex items-center gap-2">
              <Building className="w-4 h-4 text-purple-600"/>
              Cấu trúc số phòng theo tầng
            </h3>
            <p className="text-xs text-slate-500">Hệ thống sẽ tự động khởi tạo danh sách phòng dựa vào số lượng bạn cấu hình bên dưới. Nhập 0 nếu không muốn auto-generate.</p>
            <div className="space-y-2">
              <Label>Số tầng của toà nhà (bao gồm cả trệt/thượng)</Label>
              <Input type="number" min="0" max="20" value={floorCountStr} onChange={e => handleFloorCountChange(e.target.value)} className="border-slate-200 max-w-[200px]" />
            </div>
            
            {(parseInt(floorCountStr) || 0) > 0 && (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4 bg-white p-3 rounded border border-slate-200 max-h-48 overflow-y-auto">
                {Array.from({ length: parseInt(floorCountStr) || 0 }).map((_, i) => {
                   const floorNo = i + 1;
                   return (
                     <div key={floorNo} className="space-y-1">
                       <Label className="text-xs text-slate-600">Số phòng Tầng {floorNo}</Label>
                       <Input 
                         type="number" min="0" 
                         value={roomsPerFloor[floorNo] ?? 1} 
                         onChange={e => setRoomsPerFloor(prev => ({...prev, [floorNo]: parseInt(e.target.value) || 0}))} 
                         className="h-8 text-sm border-slate-200" 
                       />
                     </div>
                   )
                })}
              </div>
            )}
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <Button type="button" variant="outline" onClick={onClose} className="border-slate-200 text-slate-600">Hủy</Button>
            <Button type="submit" disabled={isLoading} className="bg-purple-600 text-white hover:bg-purple-700 shadow-md">
              {isLoading ? 'Đang khởi tạo...' : 'Xác nhận tạo'}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  )
}
