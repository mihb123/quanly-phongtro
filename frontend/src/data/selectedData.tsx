import { create } from 'zustand'
import { type House } from '@/api/house'
import { readStorageRaw, writeStorageRaw } from '@/lib/storage'

export const TABS = ['dashboard', 'house_rooms', 'tenants', 'invoices', 'revenue', 'settings'] as const
export type TabType = (typeof TABS)[number]

const TAB_KEY = 'home_active_tab'
const HOUSE_KEY = 'home_selected_house_id'

export function readSavedHouseId(): string | null {
  return readStorageRaw(HOUSE_KEY)
}

function isTab(value: string | null): value is TabType {
  return TABS.includes(value as TabType)
}

interface SelectedDataState {
  selectedHouse: House | null
  activeTab: TabType
  isHouseListOpen: boolean
  selectHouse: (house: House | null) => void
  setActiveTab: (tab: TabType) => void
  setIsHouseListOpen: (open: boolean) => void
  tabChangeInterceptor: ((nextTab: TabType) => boolean) | null
  setTabChangeInterceptor: (interceptor: ((nextTab: TabType) => boolean) | null) => void
}

const rawTab = readStorageRaw(TAB_KEY)
const savedTab: TabType = isTab(rawTab) ? rawTab : 'dashboard'
const savedHouseId = readStorageRaw(HOUSE_KEY)

export const useSelectedStore = create<SelectedDataState>((set, get) => ({
  selectedHouse: null, // this will be hydrated in Home.tsx after fetchHouses
  activeTab: savedTab,
  isHouseListOpen: !!savedHouseId,
  tabChangeInterceptor: null,
  
  setTabChangeInterceptor: (interceptor) => set({ tabChangeInterceptor: interceptor }),
  
  selectHouse: (house: House | null) => {
    set({ selectedHouse: house })
    writeStorageRaw(HOUSE_KEY, house ? house.id : null)
  },
  
  setActiveTab: (tab: TabType) => {
    const interceptor = get().tabChangeInterceptor;
    if (interceptor && !interceptor(tab)) {
      return; // Interceptor rejected the tab change
    }
    if (!isTab(tab)) return
    set({ activeTab: tab })
    writeStorageRaw(TAB_KEY, tab)
  },
  
  setIsHouseListOpen: (open: boolean) => set({ isHouseListOpen: open }),
}))
