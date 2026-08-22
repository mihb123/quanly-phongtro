import { apiClient } from './client'

export interface Room {
  id: string
  house_id: string
  name: string
  price: number
  max_tenants: number
  status: string
  electricity_price?: number
  water_price?: number
  wifi_price?: number
  parking_price?: number
  service_price?: number
  extra_person_threshold?: number
  extra_person_fee?: number
  extra_vehicle_threshold?: number
  extra_vehicle_fee?: number
  group_chat_id?: string
  contract_path?: string
}

export const getRoomsByHouseId = async (houseId: string, page: number = 1, limit: number = 25) => {
  const { data } = await apiClient.get(`/room?house_id=${houseId}&page=${page}&limit=${limit}`)
  return (data.data || []) as Room[]
}

export const createRoom = async (payload: Partial<Room>) => {
  const { data } = await apiClient.post('/room/', payload)
  return data.data as Room
}

export const updateRoom = async (id: string, payload: Partial<Room>) => {
  const { data } = await apiClient.patch(`/room/${id}`, payload)
  return data.data as Room
}

// Hợp đồng thuê gắn theo phòng nên đi qua endpoint multipart riêng.
export const updateRoomContract = async (id: string, payload: FormData) => {
  const { data } = await apiClient.patch(`/room/${id}/contract`, payload, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data.data as Room
}

export const deleteRoom = async (id: string, houseId: string) => {
  const { data } = await apiClient.delete(`/room/${id}?house_id=${houseId}`)
  return data
}
