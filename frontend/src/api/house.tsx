import { apiClient } from './client'

export interface House {
  id: string
  manager_id: string
  name: string
  house_code: string
  address: string
  default_electricity_price?: number
  default_water_price?: number
  default_wifi_price?: number
  default_parking_price?: number
  default_service_price?: number
  electricity_billing_type?: string
  water_billing_type?: string
  electricity_billing_unit?: string
  water_billing_unit?: string
  extra_person_threshold?: number
  extra_person_fee?: number
  extra_vehicle_threshold?: number
  extra_vehicle_fee?: number
  // Thông tin thuê nguyên căn từ chủ nhà
  owner_name?: string
  owner_phone?: string
  owner_rent_price?: number
  owner_deposit?: number
  rent_start_date?: string | null
  rent_end_date?: string | null
  // Nhiều file, ngăn cách bởi dấu phẩy
  owner_cccd_path?: string
  owner_contract_path?: string
  created_at?: string
}

export const getHouses = async () => {
  const { data } = await apiClient.get('/house/')
  return (data.data || []) as House[]
}

export const createHouse = async (payload: Partial<House>) => {
  const { data } = await apiClient.post('/house/create', payload)
  return data.data as House
}

export const updateHouse = async (id: string, payload: Partial<House>) => {
  const { data } = await apiClient.post(`/house/${id}`, payload)
  return data.data as House
}

// CCCD chủ nhà + hợp đồng thuê nguyên căn đi qua endpoint multipart riêng.
export const updateHouseDocuments = async (id: string, payload: FormData) => {
  const { data } = await apiClient.patch(`/house/${id}/documents`, payload, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data.data as House
}

export const deleteHouse = async (id: string) => {
  const { data } = await apiClient.delete(`/house/${id}`)
  return data
}

// checkHouseCode hỏi backend xem house_code còn dùng được không (duy nhất toàn hệ thống); excludeId là nhà đang sửa (bỏ qua chính nó).
export const checkHouseCode = async (code: string, excludeId?: string) => {
  const params = new URLSearchParams({ code })
  if (excludeId) params.set('exclude', excludeId)
  const { data } = await apiClient.get(`/house/check-code?${params.toString()}`)
  return data.data as { available: boolean }
}
