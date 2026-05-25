import { create } from 'zustand'
import { getHouses, deleteHouse, updateHouse as apiUpdateHouse, type House } from '@/api/house'

interface HouseDataState {
  houses: House[]
  fetchHouses: () => Promise<void>
  deleteHouse: (houseId: string) => Promise<boolean>
  updateHouse: (id: string, payload: Partial<House>) => Promise<House | null>
}

export const useHouseStore = create<HouseDataState>((set) => ({
  houses: [],
  fetchHouses: async () => {
    try {
      const data = await getHouses()
      set({ houses: data || [] })
    } catch (err) {
      console.error("Failed to fetch houses", err)
    }
  },
  deleteHouse: async (houseId: string) => {
    try {
      await deleteHouse(houseId)
      // fetchHouses will be called after this to refresh the list, or we could filter here
      return true
    } catch (err) {
      console.error("Failed to delete house", err)
      return false
    }
  },
  updateHouse: async (id: string, payload: Partial<House>) => {
    try {
      const house = await apiUpdateHouse(id, payload)
      return house
    } catch (err) {
      console.error("Failed to update house", err)
      return null
    }
  }
}))
