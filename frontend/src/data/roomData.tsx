import { create } from 'zustand'
import { getAllRoomsByHouseId, getRoomsPage, deleteRoom, createRoom as apiCreateRoom, updateRoom as apiUpdateRoom, updateRoomContract as apiUpdateRoomContract, type Room } from '@/api/room'
import { useSelectedStore } from './selectedData'
import { useInvoiceStore } from './invoiceData'
import { fetchQuery, getQueryData, invalidateQueries, queryKey } from '@/lib/queryCache'
import { readPageSize, writeStorage } from '@/lib/storage'

export const ROOM_PAGE_SIZE_KEY = 'rooms:pageSize'

type RoomPage = { items: Room[]; total: number }

const byName = (a: Room, b: Room) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })

export const roomListKey = (houseId: string, page: number, limit: number, q: string) =>
  queryKey('rooms:list', { house_id: houseId, page, limit, q })

let latestRequest = 0

interface RoomDataState {
  rooms: Room[]
  roomsHouseId: string | null
  roomTotal: number
  roomPage: number
  roomPageSize: number
  roomSearch: string
  roomsLoading: boolean
  roomsError: string | null
  setRoomPage: (page: number) => void
  setRoomPageSize: (size: number) => void
  setRoomSearch: (q: string) => void
  fetchRooms: (houseId: string, page: number) => Promise<void>
  refreshCurrentRooms: () => Promise<void>
  deleteRoom: (roomId: string, houseId: string) => Promise<{success: boolean, error?: string}>
  createRoom: (payload: Partial<Room>) => Promise<{success: boolean, error?: string}>
  updateRoom: (roomId: string, payload: Partial<Room>) => Promise<{success: boolean, error?: string}>
  updateRoomContract: (roomId: string, payload: FormData) => Promise<{success: boolean, data?: Room, error?: string}>
  getRoomsByHouse: (houseId: string) => Promise<Room[]>
}

const errorMessage = (error: unknown, fallback: string) => {
  const err = error as Error & { response?: { data?: { message?: string } } }
  return err?.response?.data?.message || err?.message || fallback
}

const afterRoomMutation = () => {
  invalidateQueries('rooms:')
  invalidateQueries('tenants:')
  void useRoomStore.getState().refreshCurrentRooms()
}

export const useRoomStore = create<RoomDataState>((set, get) => ({
  rooms: [],
  roomsHouseId: null,
  roomTotal: 0,
  roomPage: 1,
  roomPageSize: readPageSize(ROOM_PAGE_SIZE_KEY),
  roomSearch: '',
  roomsLoading: false,
  roomsError: null,
  setRoomPage: (page: number) => set({ roomPage: page }),
  setRoomPageSize: (size: number) => {
    writeStorage(ROOM_PAGE_SIZE_KEY, size)
    set({ roomPageSize: size, roomPage: 1 })
  },
  setRoomSearch: (q: string) => set({ roomSearch: q, roomPage: 1 }),
  fetchRooms: async (houseId: string, page: number) => {
    const { roomPageSize: limit, roomSearch: q } = get()
    const key = roomListKey(houseId, page, limit, q)
    const request = ++latestRequest
    const cached = getQueryData<RoomPage>(key)

    if (cached) {
      set({ rooms: cached.items, roomTotal: cached.total, roomsHouseId: houseId, roomsLoading: false, roomsError: null })
    } else if (get().roomsHouseId !== houseId) {
      set({ rooms: [], roomTotal: 0, roomsHouseId: houseId, roomsLoading: true, roomsError: null })
    } else {
      set({ roomsLoading: true, roomsError: null })
    }

    try {
      await fetchQuery<RoomPage>(key, async () => {
        const result = await getRoomsPage({ house_id: houseId, page, limit, q: q || undefined })
        return { items: [...result.items].sort(byName), total: result.total }
      })
      if (request !== latestRequest) return
      const shared = getQueryData<RoomPage>(key)
      if (!shared) return
      const lastPage = Math.max(1, Math.ceil(shared.total / limit))
      if (page > lastPage) {
        set({ roomPage: lastPage })
        await get().fetchRooms(houseId, lastPage)
        return
      }
      set({ rooms: shared.items, roomTotal: shared.total, roomsHouseId: houseId, roomsLoading: false })
    } catch (err) {
      if (request !== latestRequest) return
      set({ roomsLoading: false, roomsError: errorMessage(err, 'Không tải được danh sách phòng') })
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
      return { rooms: state.rooms.filter(r => r.id !== roomId), roomTotal: Math.max(0, state.roomTotal - 1) }
    })

    try {
      await deleteRoom(roomId, houseId)
      afterRoomMutation()
      return { success: true }
    } catch (error) {
      console.error("Failed to delete room", error)
      if (previousRoom) {
        set(state => ({
          rooms: [...state.rooms, previousRoom!].sort(byName),
          roomTotal: state.roomTotal + 1,
        }))
      }
      return { success: false, error: errorMessage(error, "Lỗi khi xóa phòng!") }
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
      tenant_count: 0,
    }

    const showsHouse = get().roomsHouseId === tempRoom.house_id
    if (showsHouse) {
      set(state => ({ rooms: [...state.rooms, tempRoom].sort(byName) }))
    }

    try {
      const createdRoom = await apiCreateRoom(payload)
      set(state => ({
        rooms: state.rooms.map(r => r.id === tempId ? { ...createdRoom, tenant_count: 0 } : r)
      }))
      afterRoomMutation()
      return { success: true }
    } catch (error) {
      console.error("Failed to create room", error)
      set(state => ({ rooms: state.rooms.filter(r => r.id !== tempId) }))
      return { success: false, error: errorMessage(error, "Lỗi khi thêm phòng!") }
    }
  },
  updateRoom: async (roomId: string, payload: Partial<Room>) => {
    let previousRoom: Room | undefined
    set(state => {
      previousRoom = state.rooms.find(r => r.id === roomId)
      return {
        rooms: state.rooms.map(r => r.id === roomId ? { ...r, ...payload } : r).sort(byName)
      }
    })

    try {
      const updatedRoom = await apiUpdateRoom(roomId, payload)
      set(state => ({
        rooms: state.rooms.map(r => r.id === roomId ? { ...updatedRoom, tenant_count: r.tenant_count } : r)
      }))
      afterRoomMutation()
      useInvoiceStore.getState().fetchInvoices()
      return { success: true }
    } catch (error) {
      console.error("Failed to update room", error)
      if (previousRoom) {
        set(state => ({
          rooms: state.rooms.map(r => r.id === roomId ? previousRoom! : r).sort(byName)
        }))
      }
      return { success: false, error: errorMessage(error, "Lỗi khi sửa phòng!") }
    }
  },
  updateRoomContract: async (roomId: string, payload: FormData) => {
    try {
      const updatedRoom = await apiUpdateRoomContract(roomId, payload)
      set(state => ({
        rooms: state.rooms.map(r => r.id === roomId ? { ...updatedRoom, tenant_count: r.tenant_count } : r)
      }))
      invalidateQueries('rooms:')
      return { success: true, data: updatedRoom }
    } catch (error) {
      console.error("Failed to update room contract", error)
      return { success: false, error: errorMessage(error, "Lỗi khi lưu hợp đồng!") }
    }
  },
  getRoomsByHouse: async (houseId: string) => {
    try {
      return await getAllRoomsByHouseId(houseId)
    } catch (err) {
      console.error("Failed to get rooms", err)
      return []
    }
  }
}))
