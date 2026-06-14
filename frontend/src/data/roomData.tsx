import { create } from 'zustand'
import { getRoomsByHouseId, deleteRoom, createRoom as apiCreateRoom, updateRoom as apiUpdateRoom, type Room } from '@/api/room'
import { useSelectedStore } from './selectedData'
import { useInvoiceStore } from './invoiceData'

export const ROOMS_LIMIT = 25

interface RoomDataState {
  rooms: Room[]
  roomPage: number
  setRoomPage: (page: number) => void
  fetchRooms: (houseId: string, page: number) => Promise<void>
  refreshCurrentRooms: () => Promise<void>
  deleteRoom: (roomId: string, houseId: string) => Promise<{success: boolean, error?: string}>
  createRoom: (payload: Partial<Room>) => Promise<{success: boolean, error?: string}>
  updateRoom: (roomId: string, payload: Partial<Room>) => Promise<{success: boolean, error?: string}>
  getRoomsByHouse: (houseId: string) => Promise<Room[]>
}

export const useRoomStore = create<RoomDataState>((set, get) => ({
  rooms: [],
  roomPage: 1,
  setRoomPage: (page: number) => set({ roomPage: page }),
  fetchRooms: async (houseId: string, page: number) => {
    try {
      const data = await getRoomsByHouseId(houseId, page, ROOMS_LIMIT)
      const sortedData = [...(data || [])].sort((a, b) => 
        a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })
      )
      set({ rooms: sortedData })
    } catch (err) {
      console.error("Failed to fetch rooms", err)
    }
  },
  refreshCurrentRooms: async () => {
    const selectedHouse = useSelectedStore.getState().selectedHouse
    const { roomPage, fetchRooms } = get()
    if (selectedHouse) {
      await fetchRooms(selectedHouse.id, roomPage)
    }
  },
  deleteRoom: async (roomId: string, houseId: string) => {
    let previousRoom: Room | undefined
    set(state => {
      previousRoom = state.rooms.find(r => r.id === roomId)
      return { rooms: state.rooms.filter(r => r.id !== roomId) }
    })

    try {
      await deleteRoom(roomId, houseId)
      return { success: true }
    } catch (error) {
      const err = error as Error & { response?: { data?: { message?: string } } };
      console.error("Failed to delete room", err)
      if (previousRoom) {
        set(state => ({
          rooms: [...state.rooms, previousRoom!].sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' }))
        }))
      }
      return { success: false, error: err?.response?.data?.message || err?.message || "Lỗi khi xóa phòng!" }
    }
  },
  createRoom: async (payload: Partial<Room>) => {
    const tempId = `temp-${Date.now()}-${Math.random()}`
    const tempRoom: Room = {
      id: tempId,
      house_id: payload.house_id || '',
      name: payload.name || '',
      price: payload.price || 0,
      max_tenants: payload.max_tenants || 2,
      status: payload.status || 'AVAILABLE',
      electricity_price: payload.electricity_price,
      water_price: payload.water_price,
      wifi_price: payload.wifi_price,
      parking_price: payload.parking_price,
      service_price: payload.service_price,
    }

    set(state => ({
      rooms: [...state.rooms, tempRoom].sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' }))
    }))

    try {
      const createdRoom = await apiCreateRoom(payload)
      set(state => ({
        rooms: state.rooms.map(r => r.id === tempId ? createdRoom : r)
      }))
      return { success: true }
    } catch (error) {
      const err = error as Error & { response?: { data?: { message?: string } } };
      console.error("Failed to create room", err)
      set(state => ({ rooms: state.rooms.filter(r => r.id !== tempId) }))
      return { success: false, error: err?.response?.data?.message || err?.message || "Lỗi khi thêm phòng!" }
    }
  },
  updateRoom: async (roomId: string, payload: Partial<Room>) => {
    let previousRoom: Room | undefined
    set(state => {
      previousRoom = state.rooms.find(r => r.id === roomId)
      return {
        rooms: state.rooms.map(r => r.id === roomId ? { ...r, ...payload } : r).sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' }))
      }
    })

    try {
      const updatedRoom = await apiUpdateRoom(roomId, payload)
      set(state => ({
        rooms: state.rooms.map(r => r.id === roomId ? updatedRoom : r)
      }))
      // Refetch invoices to reflect the price change in unpaid invoices
      useInvoiceStore.getState().fetchInvoices()
      return { success: true }
    } catch (error) {
      const err = error as Error & { response?: { data?: { message?: string } } };
      console.error("Failed to update room", err)
      if (previousRoom) {
        set(state => ({
          rooms: state.rooms.map(r => r.id === roomId ? previousRoom! : r).sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' }))
        }))
      }
      return { success: false, error: err?.response?.data?.message || err?.message || "Lỗi khi sửa phòng!" }
    }
  },
  getRoomsByHouse: async (houseId: string) => {
    try {
      const data = await getRoomsByHouseId(houseId, 1, 100)
      return data || []
    } catch (err) {
      console.error("Failed to get rooms", err)
      return []
    }
  }
}))
