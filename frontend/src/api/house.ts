import { apiClient } from './client'

export interface House {
  id: string
  manager_id: string
  name: string
  address: string
  default_electricity_price?: number
  default_water_price?: number
  default_wifi_price?: number
  default_parking_price?: number
  default_service_price?: number
  created_at?: string
}

export const getHouses = async () => {
  const { data } = await apiClient.get('/house/')
  return (data.data?.data || data.data || []) as House[]
}

export const createHouse = async (payload: Partial<House>) => {
  const { data } = await apiClient.post('/house/create', payload)
  return data.data as House
}

export const deleteHouse = async (id: string) => {
  const { data } = await apiClient.delete(`/house/${id}`)
  return data
}
