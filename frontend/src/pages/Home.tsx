import { useNavigate, Link } from 'react-router-dom'
import { Home, LogOut, Settings, Users, LayoutDashboard, ChevronRight } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/contexts/AuthContext'

export default function HomePage() {
  const { logout } = useAuth()
  const navigate = useNavigate()

  return (
    <div className="min-h-screen bg-slate-950 text-white selection:bg-purple-500/30">
      {/* Sidebar background effect */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-0 -left-40 w-96 h-96 rounded-full bg-purple-500/10 blur-3xl" />
      </div>

      <div className="flex h-screen overflow-hidden relative">
        {/* Sidebar */}
        <aside className="w-64 border-r border-white/10 bg-black/20 backdrop-blur-3xl p-6 flex flex-col gap-8">
          <div className="flex items-center gap-3 px-2">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center shadow-lg shadow-purple-500/30">
              <Home className="w-5 h-5 text-white" />
            </div>
            <span className="font-bold text-xl tracking-tight">Trọ Pro</span>
          </div>

          <nav className="flex-1 flex flex-col gap-2">
            <SidebarItem icon={<LayoutDashboard />} label="Dashboard" active />
            <SidebarItem icon={<Users />} label="Khách thuê" />
            <SidebarItem icon={<Settings />} label="Cài đặt" />
          </nav>

          <Button 
            variant="ghost" 
            className="w-full justify-start gap-3 text-slate-300 hover:text-red-400 hover:bg-red-400/10"
            onClick={async () => {
              await logout()
              navigate('/login')
            }}
          >
            <LogOut className="w-5 h-5" />
            <span>Đăng xuất</span>
          </Button>
        </aside>

        {/* Main Content */}
        <main className="flex-1 overflow-auto p-8 relative">
          <div className="max-w-6xl mx-auto space-y-8">
            <div className="flex justify-between items-center">
              <div>
                <h1 className="text-4xl font-extrabold tracking-tight text-white">Chào mừng bạn trở lại! 👋</h1>
                <p className="text-slate-300 mt-2">Hôm nay mọi thứ đang diễn ra rất tốt tại các phòng trọ của bạn.</p>
              </div>
              <div className="flex items-center gap-4">
                <Button className="bg-purple-600 hover:bg-purple-700 shadow-lg shadow-purple-500/20 px-6 font-semibold">
                  Thêm phòng mới
                </Button>
              </div>
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <StatCard label="Tổng doanh thu" value="125.0M" subValue="+12% so với tháng trước" />
              <StatCard label="Phòng đã thuê" value="28/30" subValue="93.3% công suất" />
              <StatCard label="Hóa đơn chờ" value="5" subValue="Cần được xác nhận" />
            </div>

            {/* Recent Actions Placeholder */}
            <Card className="bg-white/5 border-white/10 backdrop-blur-xl p-6">
              <div className="flex items-center justify-between mb-6">
                <h3 className="text-xl font-bold text-white">Hoạt động gần đây</h3>
                <Link to="#" className="text-purple-300 hover:text-purple-200 text-sm font-medium flex items-center gap-1 transition-colors">
                  Xem tất cả <ChevronRight className="w-4 h-4" />
                </Link>
              </div>
              <div className="space-y-4">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="flex items-center gap-4 p-4 rounded-xl hover:bg-white/5 transition-all border border-transparent hover:border-white/10 group cursor-pointer">
                    <div className="w-10 h-10 rounded-full bg-slate-800 flex items-center justify-center text-slate-300 group-hover:text-purple-400 transition-colors">
                      <Users className="w-5 h-5" />
                    </div>
                    <div className="flex-1">
                      <p className="font-semibold text-slate-100">Nguyễn Văn A vừa thanh toán tiền phòng 10{i}</p>
                      <p className="text-sm text-slate-400 font-medium">2 giờ trước</p>
                    </div>
                    <span className="text-emerald-400 font-bold font-mono text-lg">+3.500.000đ</span>
                  </div>
                ))}
              </div>
            </Card>
          </div>
        </main>
      </div>
    </div>
  )
}

function SidebarItem({ icon, label, active = false }: { icon: React.ReactNode, label: string, active?: boolean }) {
  return (
    <button className={`flex items-center gap-3 w-full px-4 py-3 rounded-xl transition-all duration-300 font-bold ${
      active 
        ? 'bg-purple-600 text-white shadow-lg shadow-purple-500/25' 
        : 'text-slate-400 hover:text-slate-100 hover:bg-white/5'
    }`}>
      <span className="w-5 h-5 flex items-center justify-center">
        {React.cloneElement(icon as React.ReactElement<any>, { className: 'w-5 h-5' })}
      </span>
      <span>{label}</span>
    </button>
  )
}

import React from 'react'

function StatCard({ label, value, subValue }: { label: string, value: string, subValue: string }) {
  return (
    <Card className="bg-white/5 border-white/10 backdrop-blur-2xl p-6 hover:translate-y-[-4px] transition-all duration-300 group">
      <div className="flex flex-col gap-1">
        <span className="text-slate-300 text-sm font-semibold">{label}</span>
        <span className="text-3xl font-bold bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent group-hover:from-purple-400 group-hover:to-indigo-300">
          {value}
        </span>
        <span className="text-xs text-slate-400 mt-2 flex items-center gap-1 font-medium">
          <span className="text-emerald-400 font-bold">{subValue.split(' ')[0]}</span>
          {subValue.split(' ').slice(1).join(' ')}
        </span>
      </div>
    </Card>
  )
}
