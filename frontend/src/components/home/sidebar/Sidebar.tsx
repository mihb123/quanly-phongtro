import { useState, useEffect } from 'react'
import { Home, Settings, Users, LayoutDashboard, ChevronRight, Building, ChevronDown, Trash2, Edit, Receipt } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/contexts/AuthContext'
import { UpdateProfileModal } from '@/components/home/modals/UpdateProfileModal'

import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'

import { SidebarItem } from './SidebarItem'
import { CreateHouseModal } from '@/components/home/modals/CreateHouseModal'
import { EditHouseModal } from '@/components/home/modals/EditHouseModal'
import { ConfirmModal } from '@/components/home/modals/ConfirmModal'
import type { House } from '@/api/house'
import { Plus } from 'lucide-react'

export function Sidebar() {
  const { user } = useAuth()

  const { houses, fetchHouses, deleteHouse } = useHouseStore()
  const { 
    selectedHouse, activeTab, isHouseListOpen, isSidebarCollapsed,
    selectHouse, setActiveTab, setIsHouseListOpen, setIsSidebarCollapsed 
  } = useSelectedStore()

  const [showCreateHouse, setShowCreateHouse] = useState(false)
  const [showUpdateProfile, setShowUpdateProfile] = useState(false)
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

  const handleTabClick = (tab: 'dashboard' | 'house_rooms' | 'tenants' | 'invoices' | 'settings') => {
    setActiveTab(tab)
    if (tab !== 'house_rooms') {
        selectHouse(null)
    }
  }

  // Auto collapse sidebar on smaller screens
  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth < 1000) {
        setIsSidebarCollapsed(true)
      } else {
        setIsSidebarCollapsed(false)
      }
    }
    
    // Initial check on mount
    handleResize()

    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [setIsSidebarCollapsed])

  return (
    <>
      {showUpdateProfile && (
        <UpdateProfileModal onClose={() => setShowUpdateProfile(false)} />
      )}
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
          className="fixed z-[100] bg-white border border-slate-200 shadow-2xl rounded-xl py-1 w-48 safe-fade-in"
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

      <aside className={`hidden md:flex flex-col gap-6 ${isSidebarCollapsed ? 'w-20' : 'w-72'} border-r border-border bg-card p-4 overflow-y-auto transition-all duration-300 ease-in-out z-20`}>
        <div className="flex flex-col gap-4">
          <div className={`flex items-center ${isSidebarCollapsed ? 'justify-center' : 'justify-between'} px-2 h-10`}>
            {!isSidebarCollapsed && (
              <div className="flex items-center gap-3 overflow-hidden">
                <div className="w-8 h-8 flex-shrink-0 rounded-lg bg-primary flex items-center justify-center shadow-md shadow-primary/20">
                  <Home className="w-5 h-5 text-primary-foreground" />
                </div>
                <span className="font-bold text-xl tracking-tight text-foreground whitespace-nowrap">Phòng trọ</span>
              </div>
            )}
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setIsSidebarCollapsed(!isSidebarCollapsed)}
              className={`h-8 w-8 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-lg flex-shrink-0 transition-all cursor-pointer ${isSidebarCollapsed ? 'bg-secondary' : ''}`}
            >
              <ChevronRight className={`w-5 h-5 transition-transform duration-300 ${isSidebarCollapsed ? '' : 'rotate-180'}`} />
            </Button>
          </div>
        </div>

        <nav className="flex-1 flex flex-col gap-2">
          <SidebarItem
            icon={<LayoutDashboard />}
            label="Tổng quan"
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
                  ? 'text-foreground'
                  : 'text-muted-foreground hover:text-foreground hover:bg-secondary'
                } group`}
            >
              <div className="flex items-center gap-3 min-w-0">
                <Building className={`w-5 h-5 flex-shrink-0 ${!isSidebarCollapsed && isHouseListOpen ? 'text-primary' : ''}`} />
                {!isSidebarCollapsed && <span className="whitespace-nowrap overflow-hidden text-ellipsis">Danh sách nhà trọ</span>}
              </div>
              {!isSidebarCollapsed && <ChevronDown className={`w-4 h-4 flex-shrink-0 transition-transform duration-300 ${isHouseListOpen ? 'rotate-180' : ''}`} />}
            </button>

            {isHouseListOpen && !isSidebarCollapsed && (
              <div className="flex flex-col gap-1 mt-1">
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
                        className={`flex items-center gap-3 w-full px-4 py-2.5 rounded-xl transition-all relative overflow-hidden group cursor-pointer ${
                          selectedHouse?.id === house.id && activeTab === 'house_rooms'
                            ? 'bg-primary/10 text-primary shadow-sm shadow-primary/10'
                            : 'text-muted-foreground hover:text-foreground hover:bg-secondary'
                          }`}
                      >
                        <div className="w-5 h-5 flex-shrink-0 flex items-center justify-center">
                          <div className={`w-1.5 h-1.5 rounded-full transition-all ${
                            selectedHouse?.id === house.id && activeTab === 'house_rooms'
                              ? 'bg-primary scale-100'
                              : 'bg-border scale-100 group-hover:bg-muted-foreground group-hover:scale-125'
                            }`} />
                        </div>
                        <span className="whitespace-nowrap overflow-hidden text-ellipsis text-sm font-bold">
                          {house.name}
                        </span>
                      </button>
                    ))}

                    <button
                      onClick={() => setShowCreateHouse(true)}
                      className="flex items-center gap-3 w-full px-4 py-2.5 rounded-xl transition-all cursor-pointer text-muted-foreground hover:text-foreground hover:bg-secondary group"
                    >
                      <div className="w-5 h-5 flex-shrink-0 flex items-center justify-center">
                        <Plus className="w-4 h-4 text-muted-foreground group-hover:text-foreground transition-colors" />
                      </div>
                      <span className="text-sm font-bold">Tạo thêm nhà</span>
                    </button>
                  </>
                ) : (
                  <button
                    onClick={() => setShowCreateHouse(true)}
                    className="flex items-center gap-3 w-full px-4 py-2.5 rounded-xl transition-all cursor-pointer text-muted-foreground hover:text-foreground hover:bg-secondary border border-dashed border-border mt-2 group"
                  >
                    <div className="w-5 h-5 flex-shrink-0 flex items-center justify-center">
                      <Plus className="w-4 h-4 text-muted-foreground group-hover:text-foreground transition-colors" />
                    </div>
                    <span className="text-sm font-bold">Thêm nhà trọ</span>
                  </button>
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
            icon={<Receipt />}
            label="Hóa đơn"
            active={activeTab === 'invoices'}
            onClick={() => handleTabClick('invoices')}
            collapsed={isSidebarCollapsed}
          />
          <SidebarItem
            icon={<Settings />}
            label="Cài đặt"
            active={activeTab === 'settings'}
            onClick={() => handleTabClick('settings')}
            collapsed={isSidebarCollapsed}
          />
        </nav>

        <div className="mt-auto flex flex-col gap-2 pt-4">
          {user && (
            <button
              onClick={() => setShowUpdateProfile(true)}
              className={`flex items-center gap-3 w-full p-2 rounded-xl hover:bg-secondary transition-colors cursor-pointer text-left ${isSidebarCollapsed ? 'justify-center' : ''}`}
            >
              <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0 border border-primary/20">
                <span className="text-primary font-bold text-lg">{user.full_name?.charAt(0).toUpperCase() || user.email.charAt(0).toUpperCase()}</span>
              </div>
              {!isSidebarCollapsed && (
                <div className="flex flex-col min-w-0 flex-1 overflow-hidden">
                  <span className="text-sm font-bold text-foreground truncate">{user.full_name || 'Tài khoản'}</span>
                  <span className="text-xs text-muted-foreground truncate">{user.email}</span>
                </div>
              )}
            </button>
          )}
        </div>
      </aside>
    </>
  )
}
