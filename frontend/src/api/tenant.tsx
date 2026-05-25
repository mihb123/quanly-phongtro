import { apiClient } from './client'

export interface TenantPayload {
  room_id: string
  full_name: string
  phone: string
  email?: string
  identity_card: string
  start_date: string
  cccd_file?: File
  contract_file?: File
}

export interface Tenant {
  id: string
  room_id: string
  full_name: string
  phone: string
  email?: string
  identity_card: string
  start_date: string
  status: string
  cccd_path?: string
  contract_path?: string
}

// Create a new tenant
export const createTenant = async (payload: FormData) => {
  const { data } = await apiClient.post('/tenant/', payload, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  })
  return { data: (data.data?.data || data.data || data) as Tenant }
}

// Get all active tenants by room ID
export const getTenantsByRoom = async (roomId: string) => {
  const { data } = await apiClient.get(`/tenant/room/${roomId}`)
  const tenants = (data.data?.data || data.data || data)
  const tenantArray = Array.isArray(tenants) ? tenants : [tenants]
  return { data: tenantArray.map((t: Partial<Tenant> & { tenant_id?: string }) => ({ ...t, id: t.tenant_id || t.id })) as Tenant[] }
}

export const updateTenant = async (id: string, payload: FormData) => {
  const { data } = await apiClient.patch(`/tenant/${id}`, payload, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  })
  return { data: (data.data?.data || data.data || data) as Tenant }
}

export const deleteTenant = async (id: string) => {
  const { data } = await apiClient.delete(`/tenant/${id}`)
  return data
}
