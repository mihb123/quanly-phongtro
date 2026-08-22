import { useState, useEffect } from 'react'
import {
  Building,
  ChevronDown,
  Edit,
  Home,
  LayoutDashboard,
  MoreHorizontal,
  Plus,
  Receipt,
  Settings,
  Trash2,
  Users,
  Wallet,
} from '@/components/icons'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
} from '@/components/ui/sidebar'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { cn } from '@/lib/utils'
import { useAuth } from '@/contexts/AuthContext'
import { UpdateProfileModal } from '@/components/home/modals/UpdateProfileModal'

import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore, type TabType } from '@/data/selectedData'

import { CreateHouseModal } from '@/components/home/modals/CreateHouseModal'
import { EditHouseModal } from '@/components/home/modals/EditHouseModal'
import { ConfirmModal } from '@/components/home/modals/ConfirmModal'
import type { House } from '@/api/house'

// Các mục điều hướng chính của sidebar (ngoài nhóm "Nhà trọ" xử lý riêng).
const NAV_ITEMS: { tab: TabType; label: string; icon: typeof Home }[] = [
  { tab: 'dashboard', label: 'Tổng quan', icon: LayoutDashboard },
  { tab: 'tenants', label: 'Khách thuê', icon: Users },
  { tab: 'invoices', label: 'Hóa đơn', icon: Receipt },
  { tab: 'revenue', label: 'Doanh thu', icon: Wallet },
  { tab: 'settings', label: 'Cài đặt', icon: Settings },
]

// AppSidebar: sidebar desktop dựng trên bộ Sidebar chuẩn shadcn (collapse icon-mode,
// rail, tooltip khi thu gọn) — giữ nguyên nghiệp vụ chọn nhà/tab + các modal nhà trọ.
export function AppSidebar() {
  const { user } = useAuth()

  const { houses, fetchHouses, deleteHouse } = useHouseStore()
  const {
    selectedHouse, activeTab, isHouseListOpen,
    selectHouse, setActiveTab, setIsHouseListOpen,
  } = useSelectedStore()

  const [showCreateHouse, setShowCreateHouse] = useState(false)
  const [showUpdateProfile, setShowUpdateProfile] = useState(false)
  const [houseToDelete, setHouseToDelete] = useState<House | null>(null)
  const [houseToEdit, setHouseToEdit] = useState<House | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

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

  const handleTabClick = (tab: TabType) => {
    setActiveTab(tab)
    if (tab !== 'house_rooms') {
      selectHouse(null)
    }
  }

  const userInitial = (user?.full_name?.charAt(0) || user?.email.charAt(0) || '?').toUpperCase()

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
          message="Toàn bộ phòng, khách thuê, hóa đơn, thanh toán và chi phí của nhà này sẽ bị xóa vĩnh viễn. Không có cách khôi phục."
          confirmPhrase={houseToDelete.name}
          confirmText="Xóa ngay"
          cancelText="Bỏ qua"
          isLoading={isDeleting}
          onCancel={() => setHouseToDelete(null)}
          onConfirm={async () => {
            setIsDeleting(true)
            const res = await deleteHouse(houseToDelete.id)
            if (res.success) {
              await fetchHouses()
              if (selectedHouse?.id === houseToDelete.id) {
                selectHouse(null)
                setActiveTab('dashboard')
              }
            } else {
              alert(res.error || "Lỗi khi xóa nhà trọ, vui lòng thử lại!")
            }
            setIsDeleting(false)
            setHouseToDelete(null)
          }}
        />
      )}

      <Sidebar collapsible="icon">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton size="lg" onClick={() => handleTabClick('dashboard')}>
                <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                  <Home className="size-4" />
                </div>
                <div className="flex flex-col gap-0.5 leading-none">
                  <span className="font-semibold">Phòng trọ</span>
                  <span className="text-xs text-muted-foreground">Quản lý cho thuê</span>
                </div>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    tooltip="Tổng quan"
                    isActive={activeTab === 'dashboard'}
                    onClick={() => handleTabClick('dashboard')}
                  >
                    <LayoutDashboard />
                    <span>Tổng quan</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>

                <Collapsible open={isHouseListOpen} onOpenChange={setIsHouseListOpen}>
                  <SidebarMenuItem>
                    <CollapsibleTrigger
                      render={
                        <SidebarMenuButton
                          tooltip="Nhà trọ"
                          isActive={activeTab === 'house_rooms'}
                        >
                          <Building />
                          <span>Nhà trọ</span>
                          <ChevronDown
                            className={cn(
                              'ml-auto transition-transform duration-200',
                              isHouseListOpen && 'rotate-180',
                            )}
                          />
                        </SidebarMenuButton>
                      }
                    />
                    <CollapsibleContent>
                      <SidebarMenuSub>
                        {houses.map(house => (
                          <SidebarMenuSubItem key={house.id}>
                            <SidebarMenuSubButton
                              isActive={selectedHouse?.id === house.id && activeTab === 'house_rooms'}
                              render={<button type="button" />}
                              onClick={() => handleHouseClick(house)}
                            >
                              <span>{house.name}</span>
                            </SidebarMenuSubButton>
                            <DropdownMenu>
                              <DropdownMenuTrigger
                                render={
                                  <SidebarMenuAction>
                                    <MoreHorizontal />
                                    <span className="sr-only">Tùy chọn nhà trọ</span>
                                  </SidebarMenuAction>
                                }
                              />
                              <DropdownMenuContent side="right" align="start">
                                <DropdownMenuGroup>
                                  <DropdownMenuItem onClick={() => setHouseToEdit(house)}>
                                    <Edit />
                                    Sửa thông tin
                                  </DropdownMenuItem>
                                  <DropdownMenuSeparator />
                                  <DropdownMenuItem
                                    variant="destructive"
                                    onClick={() => setHouseToDelete(house)}
                                  >
                                    <Trash2 />
                                    Xóa nhà trọ
                                  </DropdownMenuItem>
                                </DropdownMenuGroup>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </SidebarMenuSubItem>
                        ))}
                        <SidebarMenuSubItem>
                          <SidebarMenuSubButton
                            render={<button type="button" />}
                            onClick={() => setShowCreateHouse(true)}
                            className="text-muted-foreground"
                          >
                            <Plus />
                            <span>{houses.length > 0 ? 'Tạo thêm nhà' : 'Thêm nhà trọ'}</span>
                          </SidebarMenuSubButton>
                        </SidebarMenuSubItem>
                      </SidebarMenuSub>
                    </CollapsibleContent>
                  </SidebarMenuItem>
                </Collapsible>

                {NAV_ITEMS.filter(item => item.tab !== 'dashboard').map(item => (
                  <SidebarMenuItem key={item.tab}>
                    <SidebarMenuButton
                      tooltip={item.label}
                      isActive={activeTab === item.tab}
                      onClick={() => handleTabClick(item.tab)}
                    >
                      <item.icon />
                      <span>{item.label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>

        <SidebarFooter>
          {user && (
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  size="lg"
                  tooltip="Tài khoản"
                  onClick={() => setShowUpdateProfile(true)}
                >
                  <Avatar className="size-8 rounded-lg">
                    <AvatarFallback className="rounded-lg">{userInitial}</AvatarFallback>
                  </Avatar>
                  <div className="flex min-w-0 flex-col leading-tight">
                    <span className="truncate text-sm font-medium">{user.full_name || 'Tài khoản'}</span>
                    <span className="truncate text-xs text-muted-foreground">{user.email}</span>
                  </div>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          )}
        </SidebarFooter>

        <SidebarRail />
      </Sidebar>
    </>
  )
}
