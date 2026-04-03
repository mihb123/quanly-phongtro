import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Home, LogOut, Settings, Users, LayoutDashboard, ChevronRight, Building, ChevronDown, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/contexts/AuthContext'
import { useHomeData } from '@/hooks/useHomeData'

// Import extracted components
import { SidebarItem } from '@/components/home/SidebarItem'
import { DashboardView } from '@/components/home/DashboardView'
import { HouseRoomsView } from '@/components/home/HouseRoomsView'
import { TenantsView } from '@/components/home/TenantsView'
import { CreateHouseModal } from '@/components/home/CreateHouseModal'
import { CreateRoomModal } from '@/components/home/CreateRoomModal'
import { EditRoomModal } from '@/components/home/EditRoomModal'
import { ConfirmModal } from '@/components/home/ConfirmModal'
import { ROOMS_LIMIT } from '@/hooks/useHomeData'
import type { Room } from '@/api/room'

export default function HomePage() {
  const { logout } = useAuth()
  const navigate = useNavigate()

  const {
    houses,
    rooms,
    selectedHouse,
    activeTab,
    isHouseListOpen,
    setIsHouseListOpen,
    isSidebarCollapsed,
    setIsSidebarCollapsed,
    roomPage,
    setRoomPage,
    fetchHouses,
    fetchRooms,
    handleHouseClick,
    handleTabClick,
    handleHouseDelete
  } = useHomeData()

  const [showCreateHouse, setShowCreateHouse] = useState(false)
  const [showCreateRoom, setShowCreateRoom] = useState(false)
  const [showEditRoom, setShowEditRoom] = useState<Room | null>(null)
  const [contextMenu, setContextMenu] = useState<{ x: number, y: number, house: any } | null>(null)
  const [houseToDelete, setHouseToDelete] = useState<any>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    const handleClick = () => setContextMenu(null)
    window.addEventListener('click', handleClick)
    return () => window.removeEventListener('click', handleClick)
  }, [])

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 selection:bg-purple-200/50">
      {/* Modals */}
      {showCreateHouse && (
        <CreateHouseModal
          onClose={() => setShowCreateHouse(false)}
          onSuccess={() => { setShowCreateHouse(false); fetchHouses() }}
        />
      )}
      {showCreateRoom && selectedHouse && (
        <CreateRoomModal
          houseId={selectedHouse.id}
          onClose={() => setShowCreateRoom(false)}
          onSuccess={() => { setShowCreateRoom(false); fetchRooms(selectedHouse.id, roomPage) }}
        />
      )}
      {showEditRoom && selectedHouse && (
        <EditRoomModal
          house={selectedHouse}
          room={showEditRoom}
          onClose={() => setShowEditRoom(null)}
          onSuccess={() => { setShowEditRoom(null); fetchRooms(selectedHouse.id, roomPage) }}
        />
      )}
      {houseToDelete && (
        <ConfirmModal
          title={`Xóa nhà: ${houseToDelete.name}`}
          message="Bạn có chắc chắn muốn xóa nhà trọ này không? Tất cả phòng trọ và dữ liệu liên quan sẽ bị xóa vĩnh viễn và không thể khôi phục."
          confirmText="Xóa ngay"
          cancelText="Bỏ qua"
          isLoading={isDeleting}
          onCancel={() => setHouseToDelete(null)}
          onConfirm={async () => {
            setIsDeleting(true)
            await handleHouseDelete(houseToDelete.id)
            setIsDeleting(false)
            setHouseToDelete(null)
          }}
        />
      )}

      {/* Sidebar background effect */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-0 -left-40 w-96 h-96 rounded-full bg-purple-200/50 blur-3xl" />
      </div>

      <div className="flex h-screen overflow-hidden relative">
        {/* Sidebar */}
        <aside className={`${isSidebarCollapsed ? 'w-20' : 'w-72'} border-r border-slate-200 bg-white/60 backdrop-blur-3xl p-4 flex flex-col gap-6 overflow-y-auto transition-all duration-300 ease-in-out z-20`}>
          <div className="flex flex-col gap-4">
            <div className={`flex items-center ${isSidebarCollapsed ? 'justify-center' : 'justify-between'} px-2 h-10`}>
              {!isSidebarCollapsed && (
                <div className="flex items-center gap-3 overflow-hidden">
                  <div className="w-8 h-8 flex-shrink-0 rounded-lg bg-gradient-to-br from-purple-500 to-indigo-600 flex items-center justify-center shadow-md shadow-purple-500/20">
                    <Home className="w-5 h-5 text-white" />
                  </div>
                  <span className="font-bold text-xl tracking-tight text-slate-800 whitespace-nowrap">Trọ Pro</span>
                </div>
              )}
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
                className={`h-8 w-8 text-slate-400 hover:text-purple-600 hover:bg-purple-50 rounded-lg flex-shrink-0 transition-all cursor-pointer ${isSidebarCollapsed ? 'bg-slate-100/50' : ''}`}
              >
                <ChevronRight className={`w-5 h-5 transition-transform duration-300 ${isSidebarCollapsed ? '' : 'rotate-180'}`} />
              </Button>
            </div>
          </div>

          <nav className="flex-1 flex flex-col gap-2">
            <SidebarItem
              icon={<LayoutDashboard />}
              label="Dashboard"
              active={activeTab === 'dashboard'}
              onClick={() => handleTabClick('dashboard')}
              collapsed={isSidebarCollapsed}
            />

            <div className={`mt-2 ${!isSidebarCollapsed && isHouseListOpen ? 'bg-slate-50/50 rounded-2xl pb-2' : ''} transition-all duration-300`}>
              <button
                onClick={() => {
                  if (isSidebarCollapsed) setIsSidebarCollapsed(false);
                  setIsHouseListOpen(!isHouseListOpen);
                }}
                className={`flex items-center ${isSidebarCollapsed ? 'justify-center' : 'justify-between'} w-full px-4 py-3 rounded-xl transition-all duration-300 font-bold cursor-pointer ${!isSidebarCollapsed && isHouseListOpen
                    ? 'text-slate-800'
                    : 'text-slate-500 hover:text-slate-800 hover:bg-slate-100/80'
                  } group`}
              >
                <div className="flex items-center gap-3 min-w-0">
                  <Building className={`w-5 h-5 flex-shrink-0 ${!isSidebarCollapsed && isHouseListOpen ? 'text-purple-600' : ''}`} />
                  {!isSidebarCollapsed && <span className="whitespace-nowrap overflow-hidden text-ellipsis">Danh sách nhà trọ</span>}
                </div>
                {!isSidebarCollapsed && <ChevronDown className={`w-4 h-4 flex-shrink-0 transition-transform duration-300 ${isHouseListOpen ? 'rotate-180' : ''}`} />}
              </button>

              {isHouseListOpen && !isSidebarCollapsed && (
                <div className="pl-4 pr-2 flex flex-col gap-1 mt-1">
                  {houses.length > 0 ? (
                    <>
                      {houses.map(house => (
                        <button
                          key={house.id}
                          onClick={() => handleHouseClick(house)}
                          onContextMenu={(e) => {
                            e.preventDefault()
                            setContextMenu({ x: e.clientX, y: e.clientY, house })
                          }}
                          className={`text-left text-sm font-bold py-2.5 px-10 rounded-xl transition-all relative overflow-hidden group cursor-pointer ${selectedHouse?.id === house.id && activeTab === 'house_rooms'
                              ? 'text-purple-700 bg-purple-50/80'
                              : 'text-slate-500 hover:text-slate-800 hover:bg-slate-100'
                            }`}
                        >
                          <div className={`absolute left-4 top-1/2 -translate-y-1/2 w-1.5 h-1.5 rounded-full transition-all ${selectedHouse?.id === house.id && activeTab === 'house_rooms'
                              ? 'bg-purple-500 scale-100'
                              : 'bg-slate-300 scale-0 group-hover:scale-100'
                            }`} />
                          <span className="whitespace-nowrap overflow-hidden text-ellipsis block italic">
                            {house.name}
                          </span>
                        </button>
                      ))}

                      {/* Context Menu */}
                      {contextMenu && (
                        <div
                          className="fixed z-[100] bg-white border border-slate-200 shadow-2xl rounded-xl py-1 w-48 animate-in fade-in zoom-in duration-150"
                          style={{ top: contextMenu.y, left: contextMenu.x }}
                          onClick={(e) => e.stopPropagation()}
                        >
                          <button
                            onClick={() => {
                              setHouseToDelete(contextMenu.house)
                              setContextMenu(null)
                            }}
                            className="flex items-center gap-3 w-full px-4 py-2 text-sm font-bold text-red-500 hover:bg-red-50 transition-colors"
                          >
                            <Trash2 className="w-4 h-4" />
                            <span>Xóa nhà trọ</span>
                          </button>
                        </div>
                      )}
                      <Button
                        variant="ghost"
                        onClick={() => setShowCreateHouse(true)}
                        className="flex items-center justify-start gap-3 w-full px-10 py-2.5 rounded-xl text-sm font-bold text-purple-600 hover:text-purple-700 hover:bg-purple-100/50 transition-all cursor-pointer h-auto"
                      >
                        Tạo thêm nhà
                      </Button>
                    </>
                  ) : (
                    <Button
                      variant="outline"
                      onClick={() => setShowCreateHouse(true)}
                      className="flex items-center justify-start gap-3 w-full px-10 py-2.5 rounded-xl text-sm font-bold text-purple-600 hover:bg-purple-100/50 transition-all border border-dashed border-purple-200 cursor-pointer h-auto"
                    >
                      Thêm nhà trọ
                    </Button>
                  )}
                </div>
              )}
            </div>

            <SidebarItem
              icon={<Users />}
              label="Khách thuê"
              active={activeTab === 'tenants'}
              onClick={() => handleTabClick('tenants')}
              collapsed={isSidebarCollapsed}
            />
            <SidebarItem
              icon={<Settings />}
              label="Cài đặt"
              onClick={() => { }}
              collapsed={isSidebarCollapsed}
            />
          </nav>

          <Button
            variant="ghost"
            className={`w-full ${isSidebarCollapsed ? 'justify-center' : 'justify-start'} gap-3 text-slate-500 hover:text-red-500 hover:bg-red-50 mt-10 p-4 transition-all duration-300 cursor-pointer`}
            onClick={async () => {
              await logout()
              navigate('/login')
            }}
          >
            <LogOut className="w-5 h-5 flex-shrink-0" />
            {!isSidebarCollapsed && <span className="font-bold">Đăng xuất</span>}
          </Button>
        </aside>

        {/* Main Content */}
        <main className="flex-1 overflow-auto p-4 md:p-8 relative transition-all duration-300">
          <div className="max-w-6xl mx-auto space-y-8">
            {activeTab === 'dashboard' && (
              <DashboardView onOpenCreateHouse={() => setShowCreateHouse(true)} />
            )}

            {activeTab === 'house_rooms' && selectedHouse && (
              <HouseRoomsView
                house={selectedHouse}
                rooms={rooms}
                onOpenCreateRoom={() => setShowCreateRoom(true)}
                onEditRoom={(room) => setShowEditRoom(room)}
                page={roomPage}
                onPageChange={setRoomPage}
                limit={ROOMS_LIMIT}
              />
            )}

            {activeTab === 'tenants' && (
              <TenantsView houses={houses} />
            )}
          </div>
        </main>
      </div>
    </div>
  )
}
