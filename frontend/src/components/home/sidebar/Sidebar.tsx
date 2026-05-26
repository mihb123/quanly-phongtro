import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Home, LogOut, Settings, Users, LayoutDashboard, ChevronRight, Building, ChevronDown, Trash2, Edit } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/contexts/AuthContext'

import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'

import { SidebarItem } from './SidebarItem'
import { CreateHouseModal } from '@/components/home/modals/CreateHouseModal'
import { EditHouseModal } from '@/components/home/modals/EditHouseModal'
import { ConfirmModal } from '@/components/home/modals/ConfirmModal'
import type { House } from '@/api/house'

export function Sidebar() {
  const { logout } = useAuth()
  const navigate = useNavigate()

  const { houses, fetchHouses, deleteHouse } = useHouseStore()
  const { 
    selectedHouse, activeTab, isHouseListOpen, isSidebarCollapsed,
    selectHouse, setActiveTab, setIsHouseListOpen, setIsSidebarCollapsed 
  } = useSelectedStore()

  const [showCreateHouse, setShowCreateHouse] = useState(false)
  const [contextMenu, setContextMenu] = useState<{ x: number, y: number, house: House } | null>(null)
  const [houseToDelete, setHouseToDelete] = useState<House | null>(null)
  const [houseToEdit, setHouseToEdit] = useState<House | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    const handleClick = () => setContextMenu(null)
    window.addEventListener('click', handleClick)
    return () => window.removeEventListener('click', handleClick)
  }, [])

  // Initialize data
  useEffect(() => {
    fetchHouses()
  }, [fetchHouses])

  // Hydrate selected house on load
  useEffect(() => {
    const savedHouseId = localStorage.getItem('home_selected_house_id')
    if (savedHouseId && houses.length > 0 && !selectedHouse) {
      const house = houses.find(h => h.id === savedHouseId)
      if (house) {
        selectHouse(house)
      }
    }
  }, [houses, selectedHouse, selectHouse])

  const handleHouseClick = (house: House) => {
    selectHouse(house)
    useRoomStore.getState().setRoomPage(1)
    setActiveTab('house_rooms')
  }

  const handleTabClick = (tab: 'dashboard' | 'house_rooms' | 'tenants') => {
    setActiveTab(tab)
    if (tab !== 'house_rooms') {
        selectHouse(null)
    }
  }

  return (
    <>
      {showCreateHouse && (
        <CreateHouseModal onClose={() => setShowCreateHouse(false)} />
      )}
      
      {houseToEdit && (
        <EditHouseModal house={houseToEdit} onClose={() => setHouseToEdit(null)} />
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
            const success = await deleteHouse(houseToDelete.id)
            if (success) {
              await fetchHouses()
              if (selectedHouse?.id === houseToDelete.id) {
                selectHouse(null)
                setActiveTab('dashboard')
              }
            } else {
              alert("Lỗi khi xóa nhà trọ, vui lòng thử lại!")
            }
            setIsDeleting(false)
            setHouseToDelete(null)
          }}
        />
      )}

      {contextMenu && (
        <div
          className="fixed z-[100] bg-white border border-slate-200 shadow-2xl rounded-xl py-1 w-48 animate-in fade-in zoom-in duration-150"
          style={{ top: contextMenu.y, left: contextMenu.x }}
          onClick={(e) => e.stopPropagation()}
        >
          <button
            onClick={() => {
              setHouseToEdit(contextMenu.house)
              setContextMenu(null)
            }}
            className="flex items-center gap-3 w-full px-4 py-2 text-sm font-bold text-slate-700 hover:bg-slate-50 transition-colors cursor-pointer"
          >
            <Edit className="w-4 h-4" />
            <span>Sửa thông tin</span>
          </button>
          <div className="h-px bg-slate-100 my-1 mx-2" />
          <button
            onClick={() => {
              setHouseToDelete(contextMenu.house)
              setContextMenu(null)
            }}
            className="flex items-center gap-3 w-full px-4 py-2 text-sm font-bold text-red-500 hover:bg-red-50 transition-colors cursor-pointer"
          >
            <Trash2 className="w-4 h-4" />
            <span>Xóa nhà trọ</span>
          </button>
        </div>
      )}

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
    </>
  )
}
