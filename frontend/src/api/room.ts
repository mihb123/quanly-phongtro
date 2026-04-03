import { apiClient } from './client'

export interface Room {
  id: string
  house_id: string
  name: string
  price: number
  max_tennants: number
  status: string
  electricity_price?: number
  water_price?: number
  wifi_price?: number
  parking_price?: number
  service_price?: number
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
