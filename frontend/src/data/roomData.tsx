import { create } from 'zustand'
import { getRoomsByHouseId, deleteRoom, type Room } from '@/api/room'
import { useSelectedStore } from './selectedData'

export const ROOMS_LIMIT = 25

interface RoomDataState {
  rooms: Room[]
  roomPage: number
  setRoomPage: (page: number) => void
  fetchRooms: (houseId: string, page: number) => Promise<void>
  refreshCurrentRooms: () => Promise<void>
  deleteRoom: (roomId: string, houseId: string) => Promise<boolean>
}

export const useRoomStore = create<RoomDataState>((set, get) => ({
  rooms: [],
  roomPage: 1,
  setRoomPage: (page: number) => set({ roomPage: page }),
  fetchRooms: async (houseId: string, page: number) => {
    try {
      const data = await getRoomsByHouseId(houseId, page, ROOMS_LIMIT)
      set({ rooms: data || [] })
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
    try {
      await deleteRoom(roomId, houseId)
      return true
    } catch (err) {
      console.error("Failed to delete room", err)
      return false
    }
  }
}))
