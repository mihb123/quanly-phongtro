import { useState, useEffect } from 'react'
import { Building, Plus, CheckCircle2 } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useHouseStore } from '@/data/houseData'
import { useTenantStore } from '@/data/tenantData'
import { TenantRoomModal } from '@/components/home/modals/TenantRoomModal'
import { SelectRoomModal } from '@/components/home/modals/SelectRoomModal'
import { type House } from '@/api/house'
import { type Room } from '@/api/room'
import { type Tenant } from '@/api/tenant'
import { useRoomStore } from '@/data/roomData'
import { toast } from 'sonner'

function HouseTenantTable({ house }: { house: House }) {
  const { tenantsByHouse, loadingByHouse, fetchTenants } = useTenantStore()
  const getRoomsByHouse = useRoomStore(state => state.getRoomsByHouse)
  
  const tenants = tenantsByHouse[house.id] || []
  const loading = loadingByHouse[house.id] ?? true

  const [isSelectRoomModalOpen, setIsSelectRoomModalOpen] = useState(false)
  const [selectedRoom, setSelectedRoom] = useState<Room | null>(null)
  const [modalView, setModalView] = useState<'list' | 'add' | 'edit'>('list')
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null)
  const [isFetchingRoom, setIsFetchingRoom] = useState(false)
  
  useEffect(() => {
    fetchTenants(house.id)
  }, [fetchTenants, house.id])

  const handleRoomClick = async (roomId: string) => {
    if (!roomId) return;
    setIsFetchingRoom(true)
    try {
      const rooms = await getRoomsByHouse(house.id)
      const room = rooms.find(r => r.id === roomId)
      if (room) {
        setSelectedRoom(room)
        setModalView('list')
        setEditingTenant(null)
      }
    } catch (err) {
      console.error(err)
    } finally {
      setIsFetchingRoom(false)
    }
  }

  const handleTenantClick = async (tenant: Tenant) => {
    if (!tenant.room_id) return;
    setIsFetchingRoom(true)
    try {
      const rooms = await getRoomsByHouse(house.id)
      const room = rooms.find(r => r.id === tenant.room_id)
      if (room) {
        setSelectedRoom(room)
        setEditingTenant(tenant)
        setModalView('edit')
      }
    } catch (err) {
      console.error(err)
    } finally {
      setIsFetchingRoom(false)
    }
  }

  const handlePhoneClick = (phone: string) => {
    navigator.clipboard.writeText(phone).then(() => {
      toast.success('Sao chép thành công', { description: phone })
    })
  }

  return (
    <Card className="p-6 bg-white/80 border-slate-200/60 shadow-sm mb-6 relative">
      {isFetchingRoom && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-white/50 backdrop-blur-sm rounded-lg">
          <div className="text-slate-500 font-medium">Đang tải thông tin phòng...</div>
        </div>
      )}
      <div className="flex items-center justify-between mb-4 flex-wrap gap-4">
        <h2 className="text-xl font-bold flex items-center gap-2 text-slate-800">
          <Building className="w-5 h-5 text-purple-600" />
          {house.name}
        </h2>
        <Button onClick={() => setIsSelectRoomModalOpen(true)} variant="default" className="bg-blue-600 hover:bg-blue-700 font-bold gap-2 cursor-pointer">
          <Plus className="w-4 h-4" /> Thêm khách thuê
        </Button>
      </div>
      
      <div className="overflow-x-auto rounded-lg border border-slate-200">
        <table className="w-full text-sm text-left whitespace-nowrap">
          <thead className="text-xs text-slate-500 bg-slate-50 uppercase border-b border-slate-200">
            <tr>
              <th className="px-4 py-3 font-semibold">Tên khách thuê</th>
              <th className="px-4 py-3 font-semibold">Phòng đang ở</th>
              <th className="px-4 py-3 font-semibold">Số điện thoại</th>
              <th className="px-4 py-3 font-semibold">Ngày bắt đầu ở</th>
              <th className="px-4 py-3 font-semibold">Tình trạng</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {loading ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-slate-500 italic">
                  Đang tải dữ liệu...
                </td>
              </tr>
            ) : tenants.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-slate-500 italic">
                  Chưa có khách thuê nào trong nhà này.
                </td>
              </tr>
            ) : (
              tenants.map(t => (
                <tr 
                  key={t.id} 
                  className="hover:bg-slate-50/50 transition-colors"
                >
                  <td className="px-4 py-3 font-semibold text-slate-800">
                    <span 
                      onClick={() => handleTenantClick(t)} 
                      className="cursor-pointer hover:text-blue-600 hover:underline transition-colors"
                      title="Sửa người thuê"
                    >
                      {t.full_name}
                    </span>
                  </td>
                  <td className="px-4 py-3 font-medium text-slate-700">
                    <span 
                      onClick={() => handleRoomClick(t.room_id)} 
                      className="cursor-pointer hover:text-blue-600 hover:underline transition-colors"
                      title="Xem danh sách người thuê phòng"
                    >
                      {t.room_name || 'N/A'}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-slate-600">
                    <span 
                      onClick={() => handlePhoneClick(t.phone)} 
                      className="cursor-pointer hover:text-blue-600 hover:underline transition-colors"
                      title="Sao chép số điện thoại"
                    >
                      {t.phone}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-slate-600">{new Date(t.start_date).toLocaleDateString('vi-VN')}</td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[10px] font-bold bg-green-100 text-green-700 uppercase tracking-wider">
                      <CheckCircle2 className="w-3 h-3" /> Đang ở
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {isSelectRoomModalOpen && (
        <SelectRoomModal
          houseId={house.id}
          tenants={tenants}
          onSelect={(r) => {
            setSelectedRoom(r)
            setModalView('add')
            setIsSelectRoomModalOpen(false)
          }}
          onClose={() => setIsSelectRoomModalOpen(false)}
        />
      )}

      {selectedRoom && (
        <TenantRoomModal
          room={selectedRoom}
          initialView={modalView}
          initialEditingTenant={editingTenant}
          onClose={() => {
            setSelectedRoom(null)
            setEditingTenant(null)
            fetchTenants(house.id) // refresh list after closing modal
          }}
        />
      )}
    </Card>
  )
}

export function TenantsView() {
  const { houses } = useHouseStore()

  return (
    <div className="space-y-8 animate-in fade-in duration-300">
      <div>
        <h1 className="text-3xl font-extrabold tracking-tight text-slate-800">Danh sách Khách thuê</h1>
        <p className="text-slate-500 mt-2">Thông tin khách thuê được nhóm theo từng nhà trọ.</p>
      </div>

      {houses.length === 0 ? (
        <Card className="p-8 text-center bg-white/80 border-slate-200/60 shadow-sm">
          <p className="text-slate-500">Bạn chưa có nhà trọ nào để quản lý khách thuê.</p>
        </Card>
      ) : (
        houses.map(house => (
          <HouseTenantTable key={house.id} house={house} />
        ))
      )}
    </div>
  )
}
