import { create } from 'zustand'
import { getTenantsByRoom as apiGetTenantsByRoom, createTenant as apiCreateTenant, updateTenant as apiUpdateTenant, deleteTenant as apiDeleteTenant, type Tenant, type TenantPage } from '@/api/tenant'
import { useRoomStore } from './roomData'
import { invalidateQueries, updateQueries } from '@/lib/queryCache'

interface TenantDataState {
  createTenant: (payload: FormData) => Promise<{success: boolean, error?: string}>
  updateTenant: (id: string, payload: FormData) => Promise<{success: boolean, data?: Tenant, error?: string}>
  deleteTenant: (id: string) => Promise<boolean>
  getTenantsByRoom: (roomId: string) => Promise<Tenant[]>
}

const errorMessage = (error: unknown, fallback: string) => {
  const err = error as Error & { response?: { data?: { message?: string } } }
  return err?.response?.data?.message || err?.message || fallback
}

export const refreshTenantDependents = () => {
  invalidateQueries('tenants:')
  invalidateQueries('rooms:')
  void useRoomStore.getState().refreshCurrentRooms()
}

export const useTenantStore = create<TenantDataState>(() => ({
  createTenant: async (payload: FormData) => {
    try {
      await apiCreateTenant(payload)
      refreshTenantDependents()
      return { success: true }
    } catch (error) {
      console.error("Failed to create tenant", error)
      return { success: false, error: errorMessage(error, "Lỗi khi thêm người thuê!") }
    }
  },
  updateTenant: async (id: string, payload: FormData) => {
    let previousTenant: Tenant | undefined
    updateQueries<TenantPage>('tenants:list', page => ({
      ...page,
      items: page.items.map(t => {
        if (t.id !== id) return t
        previousTenant = t
        const next = { ...t }
        if (payload.has('full_name')) next.full_name = payload.get('full_name') as string
        if (payload.has('phone')) next.phone = payload.get('phone') as string
        if (payload.has('email')) next.email = payload.get('email') as string
        if (payload.has('identity_card')) next.identity_card = payload.get('identity_card') as string
        if (payload.has('start_date')) next.start_date = payload.get('start_date') as string
        return next
      }),
    }))

    try {
      const res = await apiUpdateTenant(id, payload)
      // Update response không kèm room_name (query không JOIN rooms); phòng không đổi khi sửa nên giữ lại từ tenant cũ.
      const updatedTenant: Tenant = {
        ...res.data,
        room_name: res.data.room_name || previousTenant?.room_name,
        house_id: res.data.house_id || previousTenant?.house_id,
        house_name: res.data.house_name || previousTenant?.house_name,
      }
      invalidateQueries('tenants:')
      return { success: true, data: updatedTenant }
    } catch (error) {
      console.error("Failed to update tenant", error)
      invalidateQueries('tenants:')
      return { success: false, error: errorMessage(error, "Lỗi khi sửa người thuê!") }
    }
  },
  deleteTenant: async (id: string) => {
    updateQueries<TenantPage>('tenants:list', page => ({
      ...page,
      items: page.items.filter(t => t.id !== id),
    }))
    try {
      await apiDeleteTenant(id)
      return true
    } catch (err) {
      console.error("Failed to delete tenant", err)
      return false
    } finally {
      refreshTenantDependents()
    }
  },
  getTenantsByRoom: async (roomId: string) => {
    try {
      const res = await apiGetTenantsByRoom(roomId)
      return res.data || []
    } catch (err) {
      console.error("Failed to fetch tenants", err)
      return []
    }
  }
}))
