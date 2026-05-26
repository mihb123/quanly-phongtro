import { create } from 'zustand'
import { getHouses, deleteHouse, updateHouse as apiUpdateHouse, createHouse as apiCreateHouse, type House } from '@/api/house'

interface HouseDataState {
  houses: House[]
  fetchHouses: () => Promise<void>
  deleteHouse: (houseId: string) => Promise<boolean>
  updateHouse: (id: string, payload: Partial<House>) => Promise<House | null>
  createHouse: (payload: Partial<House>) => Promise<House | null>
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
    let previousHouse: House | undefined
    set(state => {
      previousHouse = state.houses.find(h => h.id === houseId)
      return { houses: state.houses.filter(h => h.id !== houseId) }
    })

    try {
      await deleteHouse(houseId)
      return true
    } catch (err) {
      console.error("Failed to delete house", err)
      if (previousHouse) {
        set(state => ({ houses: [...state.houses, previousHouse!] }))
      }
      return false
    }
  },
  updateHouse: async (id: string, payload: Partial<House>) => {
    let previousHouse: House | undefined
    set(state => {
      previousHouse = state.houses.find(h => h.id === id)
      return { houses: state.houses.map(h => h.id === id ? { ...h, ...payload } : h) }
    })

    try {
      const house = await apiUpdateHouse(id, payload)
      set(state => ({ houses: state.houses.map(h => h.id === id ? house : h) }))
      return house
    } catch (err) {
      console.error("Failed to update house", err)
      if (previousHouse) {
        set(state => ({ houses: state.houses.map(h => h.id === id ? previousHouse! : h) }))
      }
      return null
    }
  },
  createHouse: async (payload: Partial<House>) => {
    const tempId = `temp-${Date.now()}-${Math.random()}`
    const tempHouse: House = {
      id: tempId,
      manager_id: '',
      name: payload.name || '',
      address: payload.address || '',
      default_electricity_price: payload.default_electricity_price || 0,
      default_water_price: payload.default_water_price || 0,
      default_wifi_price: payload.default_wifi_price || 0,
      default_parking_price: payload.default_parking_price || 0,
      default_service_price: payload.default_service_price || 0,
    }

    set(state => ({ houses: [...state.houses, tempHouse] }))

    try {
      const house = await apiCreateHouse(payload)
      set(state => ({ houses: state.houses.map(h => h.id === tempId ? house : h) }))
      return house
    } catch (err) {
      console.error("Failed to create house", err)
      set(state => ({ houses: state.houses.filter(h => h.id !== tempId) }))
      return null
    }
  }
}))
