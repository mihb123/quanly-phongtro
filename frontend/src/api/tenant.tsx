import { apiClient } from './client'

export interface TenantPayload {
  room_id: string
  full_name: string
  phone: string
  email?: string
  identity_card: string
  start_date: string
  cccd_file?: File
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
  room_name?: string
  cccd_path?: string
  house_id?: string
  house_name?: string
}

export interface TenantListSummary {
  tenants: number
  rooms: number
  verified: number
  houses: number
}

export interface TenantPage {
  items: Tenant[]
  total: number
  page: number
  limit: number
  summary: TenantListSummary
}

export interface TenantListParams {
  house_id?: string
  q?: string
  page: number
  limit: number
}

// Normalize a tenant response: backend returns `tenant_id`, frontend uses `id`.
const normalizeTenant = (t: Partial<Tenant> & { tenant_id?: string }) =>
  ({ ...t, id: t.tenant_id || t.id }) as Tenant

export const searchTenants = async (params: TenantListParams): Promise<TenantPage> => {
  const { data } = await apiClient.get('/tenant/', { params })
  const page = data.data as Omit<TenantPage, 'items'> & { items: (Partial<Tenant> & { tenant_id?: string })[] }
  return { ...page, items: (page.items || []).map(normalizeTenant) }
}

// Create a new tenant
export const createTenant = async (payload: FormData) => {
  const { data } = await apiClient.post('/tenant/', payload, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  })
  return { data: normalizeTenant(data.data) }
}

// Get all active tenants by room ID
export const getTenantsByRoom = async (roomId: string) => {
  const { data } = await apiClient.get(`/tenant/room/${roomId}`)
  const tenants = data.data
  if (!tenants || (Array.isArray(tenants) && tenants.length === 0)) {
    return { data: [] }
  }
  const tenantArray = Array.isArray(tenants) ? tenants : [tenants]
  return { data: tenantArray.map((t: Partial<Tenant> & { tenant_id?: string, room_name?: string }) => ({ ...t, id: t.tenant_id || t.id, room_name: t.room_name })) as Tenant[] }
}

export const updateTenant = async (id: string, payload: FormData) => {
  const { data } = await apiClient.patch(`/tenant/${id}`, payload, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  })
  return { data: normalizeTenant(data.data) }
}

export const deleteTenant = async (id: string) => {
  const { data } = await apiClient.delete(`/tenant/${id}`)
  return data
}

export const checkoutRoom = async (roomId: string) => {
  const { data } = await apiClient.post(`/tenant/room/${roomId}/checkout`)
  return data.data as { removed: number }
}
