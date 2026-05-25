import { useState, useEffect } from 'react'
import { getHouses, type House } from '@/api/house'
import { getRoomsByHouseId, type Room } from '@/api/room'

export const ROOMS_LIMIT = 25

export function useHomeData() {
  const [houses, setHouses] = useState<House[]>([])
  const [rooms, setRooms] = useState<Room[]>([])
  const [selectedHouse, setSelectedHouse] = useState<House | null>(null)
  
  const savedTab = localStorage.getItem('home_active_tab') as 'dashboard' | 'house_rooms' | 'tenants' | null
  const savedHouseId = localStorage.getItem('home_selected_house_id')

  const [activeTab, setActiveTab] = useState<'dashboard' | 'house_rooms' | 'tenants'>(savedTab || 'dashboard')
  const [isHouseListOpen, setIsHouseListOpen] = useState(!!savedHouseId)
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false)
  const [roomPage, setRoomPage] = useState(1)

  const fetchHouses = async () => {
    try {
      const data = await getHouses()
      setHouses(data || [])
      
      if (savedHouseId && data) {
        const house = data.find((h: House) => h.id === savedHouseId)
        if (house) setSelectedHouse(house)
      }
    } catch (err) {
      console.error("Failed to fetch houses", err)
    }
  }

  const fetchRooms = async (houseId: string, page: number) => {
    try {
      const data = await getRoomsByHouseId(houseId, page, ROOMS_LIMIT)
      setRooms(data || [])
    } catch (err) {
      console.error("Failed to fetch rooms", err)
    }
  }

  useEffect(() => {
    fetchHouses()
  }, [])

  useEffect(() => {
    localStorage.setItem('home_active_tab', activeTab)
    if (selectedHouse) {
      localStorage.setItem('home_selected_house_id', selectedHouse.id)
      fetchRooms(selectedHouse.id, roomPage)
    } else {
      localStorage.removeItem('home_selected_house_id')
    }
  }, [selectedHouse, roomPage, activeTab])

  const handleHouseClick = (house: House) => {
    setSelectedHouse(house)
    setRoomPage(1)
    setActiveTab('house_rooms')
  }

  const handleTabClick = (tab: 'dashboard' | 'house_rooms' | 'tenants') => {
    setActiveTab(tab)
    if (tab !== 'house_rooms') {
        setSelectedHouse(null)
    }
  }

  const handleHouseDelete = async (houseId: string) => {
    const { deleteHouse } = await import('@/api/house')
    try {
      await deleteHouse(houseId)
      fetchHouses()
      if (selectedHouse?.id === houseId) {
        setSelectedHouse(null)
        setActiveTab('dashboard')
      }
      return true
    } catch (err) {
      alert("Lỗi khi xóa nhà trọ, vui lòng thử lại!")
      return false
    }
  }

  return {
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
  }
}
