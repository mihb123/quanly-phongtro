import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, ChevronRight, Users } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { StatCard } from './StatCard'
import { CreateHouseModal } from './CreateHouseModal'

export function DashboardView() {
  const [showCreateHouse, setShowCreateHouse] = useState(false)

  return (
    <div className="space-y-8 animate-in fade-in duration-300">
      {showCreateHouse && (
        <CreateHouseModal onClose={() => setShowCreateHouse(false)} />
      )}
      
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-4xl font-extrabold tracking-tight text-slate-800">Chào mừng bạn trở lại! 👋</h1>
          <p className="text-slate-500 mt-2">Hôm nay mọi thứ đang diễn ra rất tốt tại các phòng trọ của bạn.</p>
        </div>
        <div className="flex items-center gap-4">
          <Button onClick={() => setShowCreateHouse(true)} className="bg-purple-600 hover:bg-purple-700 shadow-md shadow-purple-500/20 px-6 font-semibold text-white">
            <Plus className="w-4 h-4 mr-2" /> Tạo nhà trọ
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <StatCard label="Tổng doanh thu" value="125.0M" subValue="+12% so với tháng trước" />
        <StatCard label="Phòng đã thuê" value="28/30" subValue="93.3% công suất" />
        <StatCard label="Hóa đơn chờ" value="5" subValue="Cần được xác nhận" />
      </div>

      <Card className="bg-white/80 border-slate-200/60 shadow-lg shadow-slate-200/40 backdrop-blur-xl p-6 hover:shadow-xl transition-shadow duration-300">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-xl font-bold text-slate-800">Hoạt động gần đây</h3>
          <Link to="#" className="text-purple-600 hover:text-purple-700 text-sm font-medium flex items-center gap-1 transition-colors">
            Xem tất cả <ChevronRight className="w-4 h-4" />
          </Link>
        </div>
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="flex items-center gap-4 p-4 rounded-xl hover:bg-slate-50 transition-all border border-transparent hover:border-slate-200 group cursor-pointer">
              <div className="w-10 h-10 rounded-full bg-slate-100 flex items-center justify-center text-slate-500 group-hover:text-purple-600 group-hover:bg-purple-50 transition-colors">
                <Users className="w-5 h-5" />
              </div>
              <div className="flex-1">
                <p className="font-semibold text-slate-700">Nguyễn Văn A vừa thanh toán tiền phòng 10{i}</p>
                <p className="text-sm text-slate-500 font-medium">2 giờ trước</p>
              </div>
              <span className="text-emerald-500 font-bold font-mono text-lg">+3.500.000đ</span>
            </div>
          ))}
        </div>
      </Card>
    </div>
  )
}
