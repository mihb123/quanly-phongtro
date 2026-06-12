import { create } from 'zustand'
import { type House } from '@/api/house'

export type TabType = 'dashboard' | 'house_rooms' | 'tenants' | 'invoices' | 'revenue' | 'settings'

interface SelectedDataState {
  selectedHouse: House | null
  activeTab: TabType
  isHouseListOpen: boolean
  isSidebarCollapsed: boolean
  selectHouse: (house: House | null) => void
  setActiveTab: (tab: TabType) => void
  setIsHouseListOpen: (open: boolean) => void
  setIsSidebarCollapsed: (collapsed: boolean) => void
  tabChangeInterceptor: ((nextTab: TabType) => boolean) | null
  setTabChangeInterceptor: (interceptor: ((nextTab: TabType) => boolean) | null) => void
}

const savedTab = localStorage.getItem('home_active_tab') as TabType | null
const savedHouseId = localStorage.getItem('home_selected_house_id')

export const useSelectedStore = create<SelectedDataState>((set, get) => ({
  selectedHouse: null, // this will be hydrated in Home.tsx after fetchHouses
  activeTab: savedTab || 'dashboard',
  isHouseListOpen: !!savedHouseId,
  isSidebarCollapsed: false,
  tabChangeInterceptor: null,
  
  setTabChangeInterceptor: (interceptor) => set({ tabChangeInterceptor: interceptor }),
  
  selectHouse: (house: House | null) => {
    set({ selectedHouse: house })
    if (house) {
      localStorage.setItem('home_selected_house_id', house.id)
    } else {
      localStorage.removeItem('home_selected_house_id')
    }
  },
  
  setActiveTab: (tab: TabType) => {
    const interceptor = get().tabChangeInterceptor;
    if (interceptor && !interceptor(tab)) {
      return; // Interceptor rejected the tab change
    }
    set({ activeTab: tab })
    localStorage.setItem('home_active_tab', tab)
  },
  
  setIsHouseListOpen: (open: boolean) => set({ isHouseListOpen: open }),
  setIsSidebarCollapsed: (collapsed: boolean) => set({ isSidebarCollapsed: collapsed }),
}))
