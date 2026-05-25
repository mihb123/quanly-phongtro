import { Building } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { useHouseStore } from '@/data/houseData'

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
          <Card key={house.id} className="p-6 bg-white/80 border-slate-200/60 shadow-sm mb-6">
            <h2 className="text-xl font-bold flex items-center gap-2 mb-4 text-slate-800">
              <Building className="w-5 h-5 text-purple-600" />
              {house.name}
            </h2>
            
            <div className="overflow-x-auto rounded-lg border border-slate-200">
              <table className="w-full text-sm text-left whitespace-nowrap">
                <thead className="text-xs text-slate-500 bg-slate-50 uppercase border-b border-slate-200">
                  <tr>
                    <th className="px-4 py-3 font-semibold">Tên khách thuê</th>
                    <th className="px-4 py-3 font-semibold">Phòng đang ở</th>
                    <th className="px-4 py-3 font-semibold">Ngày bắt đầu</th>
                    <th className="px-4 py-3 font-semibold">Tình trạng</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  <tr>
                    <td colSpan={4} className="px-4 py-8 text-center text-slate-500 italic">
                      Dữ liệu khách thuê đang cập nhật (Chờ phân bổ dữ liệu)
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </Card>
        ))
      )}
    </div>
  )
}
