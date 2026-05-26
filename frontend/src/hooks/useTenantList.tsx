import { useState, useCallback } from 'react'
import type { Tenant } from '@/api/tenant'
import { useTenantStore } from '@/data/tenantData'

interface UseTenantListReturn {
  tenants: Tenant[]
  isLoading: boolean
  isFetching: boolean
  showAddForm: boolean
  editingTenantId: string | null
  fetchTenants: (roomId: string) => Promise<void>
  handleDeleteTenant: (tenantId: string, roomId: string) => Promise<void>
  setShowAddForm: (show: boolean) => void
  setEditingTenantId: (id: string | null) => void
  setIsFetching: (isFetching: boolean) => void
}

export function useTenantList(_roomId: string, initialShowAddForm: boolean = false): UseTenantListReturn {
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [isFetching, setIsFetching] = useState(false)
  const [showAddForm, setShowAddForm] = useState(initialShowAddForm)
  const [editingTenantId, setEditingTenantId] = useState<string | null>(null)
  
  const getTenantsByRoom = useTenantStore(state => state.getTenantsByRoom)
  const deleteTenant = useTenantStore(state => state.deleteTenant)

  const fetchTenants = useCallback(async (rId: string) => {
    setIsFetching(true)
    try {
      const data = await getTenantsByRoom(rId)
      setTenants(data)
      if (data.length === 0) {
        setShowAddForm(true)
      }
    } catch (err) {
      console.error("Error fetching tenants", err)
    } finally {
      setIsFetching(false)
    }
  }, [getTenantsByRoom])

  const handleDeleteTenant = async (tenantId: string, rId: string) => {
    setIsLoading(true)
    try {
      await deleteTenant(tenantId)
      await fetchTenants(rId)
    } catch (error) {
      alert("Lỗi khi xóa người thuê!")
      throw error // re-throw to allow caller to handle if needed
    } finally {
      setIsLoading(false)
    }
  }

  return {
    tenants,
    isLoading,
    isFetching,
    showAddForm,
    editingTenantId,
    fetchTenants,
    handleDeleteTenant,
    setShowAddForm,
    setEditingTenantId,
    setIsFetching
  }
}
