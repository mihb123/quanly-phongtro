import { create } from 'zustand'
import { getHouses, deleteHouse, updateHouse as apiUpdateHouse, createHouse as apiCreateHouse, type House } from '@/api/house'
import { useInvoiceStore } from './invoiceData'

interface HouseDataState {
  houses: House[]
  fetchHouses: () => Promise<void>
  deleteHouse: (houseId: string) => Promise<{success: boolean, error?: string}>
  updateHouse: (id: string, payload: Partial<House>) => Promise<{success: boolean, house?: House, error?: string}>
  createHouse: (payload: Partial<House>) => Promise<{success: boolean, house?: House, error?: string}>
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
      return { success: true }
    } catch (error) {
      const err = error as Error & { response?: { data?: { message?: string } } };
      console.error("Failed to delete house", err)
      if (previousHouse) {
        set(state => ({ houses: [...state.houses, previousHouse!] }))
      }
      return { success: false, error: err?.response?.data?.message || err?.message || "Lỗi khi xóa nhà!" }
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
      useInvoiceStore.getState().fetchInvoices()
      return { success: true, house }
    } catch (error) {
      const err = error as Error & { response?: { data?: { message?: string } } };
      console.error("Failed to update house", err)
      if (previousHouse) {
        set(state => ({ houses: state.houses.map(h => h.id === id ? previousHouse! : h) }))
      }
      return { success: false, error: err?.response?.data?.message || err?.message || "Lỗi khi sửa nhà!" }
    }
  },
  createHouse: async (payload: Partial<House>) => {
    const tempId = `temp-${Date.now()}-${Math.random()}`
    const tempHouse: House = {
      id: tempId,
      manager_id: '',
      name: payload.name || '',
      house_code: payload.house_code || '',
      address: payload.address || '',
      default_electricity_price: payload.default_electricity_price || 0,
      default_water_price: payload.default_water_price || 0,
      default_wifi_price: payload.default_wifi_price || 0,
      default_parking_price: payload.default_parking_price || 0,
      default_service_price: payload.default_service_price || 0,
      electricity_billing_type: payload.electricity_billing_type || 'USAGE',
      water_billing_type: payload.water_billing_type || 'USAGE',
      electricity_billing_unit: payload.electricity_billing_unit || 'ROOM',
      water_billing_unit: payload.water_billing_unit || 'ROOM',
      extra_person_threshold: payload.extra_person_threshold || 0,
      extra_person_fee: payload.extra_person_fee || 0,
      extra_vehicle_threshold: payload.extra_vehicle_threshold || 0,
      extra_vehicle_fee: payload.extra_vehicle_fee || 0,
    }

    set(state => ({ houses: [...state.houses, tempHouse] }))

    try {
      const house = await apiCreateHouse(payload)
      set(state => ({ houses: state.houses.map(h => h.id === tempId ? house : h) }))
      return { success: true, house }
    } catch (error) {
      const err = error as Error & { response?: { data?: { message?: string } } };
      console.error("Failed to create house", err)
      set(state => ({ houses: state.houses.filter(h => h.id !== tempId) }))
      return { success: false, error: err?.response?.data?.message || err?.message || "Lỗi khi thêm nhà!" }
    }
  }
}))
