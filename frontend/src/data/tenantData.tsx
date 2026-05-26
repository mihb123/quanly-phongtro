import { create } from 'zustand'
import { getTenantsByHouse, getTenantsByRoom as apiGetTenantsByRoom, createTenant as apiCreateTenant, updateTenant as apiUpdateTenant, deleteTenant as apiDeleteTenant, type Tenant } from '@/api/tenant'
import { useRoomStore } from './roomData'

interface TenantDataState {
  tenantsByHouse: Record<string, Tenant[]>
  loadingByHouse: Record<string, boolean>
  fetchTenants: (houseId: string) => Promise<void>
  createTenant: (payload: FormData) => Promise<boolean>
  updateTenant: (id: string, payload: FormData) => Promise<boolean>
  deleteTenant: (id: string) => Promise<boolean>
  getTenantsByRoom: (roomId: string) => Promise<Tenant[]>
}

export const useTenantStore = create<TenantDataState>((set) => ({
  tenantsByHouse: {},
  loadingByHouse: {},
  fetchTenants: async (houseId: string) => {
    set((state) => ({ loadingByHouse: { ...state.loadingByHouse, [houseId]: true } }))
    try {
      const res = await getTenantsByHouse(houseId)
      set((state) => ({
        tenantsByHouse: { ...state.tenantsByHouse, [houseId]: res.data }
      }))
    } catch (err) {
      console.error(err)
    } finally {
      set((state) => ({ loadingByHouse: { ...state.loadingByHouse, [houseId]: false } }))
    }
  },
  createTenant: async (payload: FormData) => {
    const roomId = payload.get('room_id') as string;
    const room = useRoomStore.getState().rooms.find(r => r.id === roomId);
    const houseId = room?.house_id;
    
    const tempId = `temp-${Date.now()}`;
    const tempTenant: Tenant = {
      id: tempId,
      room_id: roomId,
      full_name: payload.get('full_name') as string,
      phone: payload.get('phone') as string,
      email: payload.get('email') as string,
      identity_card: payload.get('identity_card') as string,
      start_date: payload.get('start_date') as string || new Date().toISOString(),
      status: 'ACTIVE',
      room_name: room?.name,
    };

    if (houseId) {
      set(state => ({
        tenantsByHouse: {
          ...state.tenantsByHouse,
          [houseId]: [...(state.tenantsByHouse[houseId] || []), tempTenant]
        }
      }));
    }

    try {
      const res = await apiCreateTenant(payload)
      if (houseId) {
        set(state => ({
          tenantsByHouse: {
            ...state.tenantsByHouse,
            [houseId]: (state.tenantsByHouse[houseId] || []).map(t => t.id === tempId ? res.data : t)
          }
        }))
      }
      return true
    } catch (err) {
      console.error("Failed to create tenant", err)
      if (houseId) {
        set(state => ({
          tenantsByHouse: {
            ...state.tenantsByHouse,
            [houseId]: (state.tenantsByHouse[houseId] || []).filter(t => t.id !== tempId)
          }
        }))
      }
      return false
    }
  },
  updateTenant: async (id: string, payload: FormData) => {
    let foundHouseId: string | undefined;
    let previousTenant: Tenant | undefined;
    
    set(state => {
      const newTenantsByHouse = { ...state.tenantsByHouse };
      for (const [hId, tenants] of Object.entries(newTenantsByHouse)) {
        const tIndex = tenants.findIndex(t => t.id === id);
        if (tIndex !== -1) {
          foundHouseId = hId;
          previousTenant = tenants[tIndex];
          
          const updatedTenant = { ...previousTenant };
          if (payload.has('full_name')) updatedTenant.full_name = payload.get('full_name') as string;
          if (payload.has('phone')) updatedTenant.phone = payload.get('phone') as string;
          if (payload.has('email')) updatedTenant.email = payload.get('email') as string;
          if (payload.has('identity_card')) updatedTenant.identity_card = payload.get('identity_card') as string;
          if (payload.has('start_date')) updatedTenant.start_date = payload.get('start_date') as string;
          
          newTenantsByHouse[hId] = tenants.map(t => t.id === id ? updatedTenant : t);
          break;
        }
      }
      return { tenantsByHouse: newTenantsByHouse };
    });

    try {
      const res = await apiUpdateTenant(id, payload)
      if (foundHouseId) {
        const hId = foundHouseId;
        set(state => ({
          tenantsByHouse: {
            ...state.tenantsByHouse,
            [hId]: (state.tenantsByHouse[hId] || []).map((t: Tenant) => t.id === id ? res.data : t)
          }
        }))
      }
      return true
    } catch (err) {
      console.error("Failed to update tenant", err)
      if (foundHouseId && previousTenant) {
        const hId = foundHouseId;
        set(state => ({
          tenantsByHouse: {
            ...state.tenantsByHouse,
            [hId]: (state.tenantsByHouse[hId] || []).map((t: Tenant) => t.id === id ? previousTenant! : t)
          }
        }))
      }
      return false
    }
  },
  deleteTenant: async (id: string) => {
    let foundHouseId: string | undefined;
    let previousTenant: Tenant | undefined;
    
    set(state => {
      const newTenantsByHouse = { ...state.tenantsByHouse };
      for (const [hId, tenants] of Object.entries(newTenantsByHouse)) {
        const tIndex = tenants.findIndex(t => t.id === id);
        if (tIndex !== -1) {
          foundHouseId = hId;
          previousTenant = tenants[tIndex];
          newTenantsByHouse[hId] = tenants.filter(t => t.id !== id);
          break;
        }
      }
      return { tenantsByHouse: newTenantsByHouse };
    });

    try {
      await apiDeleteTenant(id)
      return true
    } catch (err) {
      console.error("Failed to delete tenant", err)
      if (foundHouseId && previousTenant) {
        const hId = foundHouseId;
        set(state => ({
          tenantsByHouse: {
            ...state.tenantsByHouse,
            [hId]: [...(state.tenantsByHouse[hId] || []), previousTenant!]
          }
        }))
      }
      return false
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
