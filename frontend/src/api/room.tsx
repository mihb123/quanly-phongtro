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
  tenant_count?: number
  house_name?: string
}

export interface RoomStats {
  total: number
  occupied: number
  available: number
  maintenance: number
  tenants: number
  capacity: number
}

export interface RoomListParams {
  house_id: string
  page: number
  limit: number
  q?: string
  status?: string
}

const totalFromHeader = (value: unknown, fallback: number) => {
  const total = Number(value)
  return Number.isFinite(total) ? total : fallback
}

export const getRoomsByHouseId = async (houseId: string, page: number = 1, limit: number = 25) => {
  const { data } = await apiClient.get(`/room?house_id=${houseId}&page=${page}&limit=${limit}`)
  return (data.data || []) as Room[]
}

export const getRoomsPage = async (params: RoomListParams) => {
  const response = await apiClient.get('/room', { params })
  const items = (response.data.data || []) as Room[]
  return { items, total: totalFromHeader(response.headers['x-total-count'], items.length) }
}

const ALL_ROOMS_PAGE = 200

// Lấy toàn bộ phòng của nhà (dùng cho select/modal) bằng cách đi hết các trang.
export const getAllRoomsByHouseId = async (houseId: string) => {
  const first = await getRoomsPage({ house_id: houseId, page: 1, limit: ALL_ROOMS_PAGE })
  const pages = Math.ceil(first.total / ALL_ROOMS_PAGE)
  if (pages <= 1) return first.items
  const rest = await Promise.all(
    Array.from({ length: pages - 1 }, (_, i) => getRoomsPage({ house_id: houseId, page: i + 2, limit: ALL_ROOMS_PAGE })),
  )
  return [...first.items, ...rest.flatMap(page => page.items)]
}

export const getRoomStats = async (houseId?: string) => {
  const { data } = await apiClient.get('/room/stats', { params: houseId ? { house_id: houseId } : undefined })
  return data.data as RoomStats
}

export const getAvailableRooms = async (limit = 50) => {
  const response = await apiClient.get('/room/available', { params: { limit } })
  const items = (response.data.data || []) as Room[]
  return { items, total: totalFromHeader(response.headers['x-total-count'], items.length) }
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
