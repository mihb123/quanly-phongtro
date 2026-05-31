import { useState, useEffect } from 'react'
import type { Room } from '@/api/room'
import type { Tenant } from '@/api/tenant'
import { TenantListModal } from './TenantListModal'
import { TenantAddModal } from './TenantAddModal'
import { TenantEditModal } from './TenantEditModal'
import { useTenantList } from '@/hooks/useTenantList'

interface TenantRoomModalProps {
  room: Room
  initialView?: 'list' | 'add' | 'edit'
  initialEditingTenant?: Tenant | null
  onClose: () => void
}

export function TenantRoomModal({ room, initialView, initialEditingTenant, onClose }: TenantRoomModalProps) {
  const isOccupied = room.status === 'OCCUPIED'
  
  const {
    tenants,
    fetchTenants,
    isFetching
  } = useTenantList(room.id)

  const [view, setView] = useState<'list' | 'add' | 'edit'>(initialView || (!isOccupied ? 'add' : 'list'))
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(initialEditingTenant || null)

  useEffect(() => {
    if (isOccupied) {
      fetchTenants(room.id)
    }
  }, [isOccupied, room.id, fetchTenants])

  useEffect(() => {
     if (!initialView && isOccupied && !isFetching && tenants.length === 0 && view === 'list') {
         // eslint-disable-next-line react-hooks/set-state-in-effect
         setView('add')
     }
  }, [isOccupied, isFetching, tenants.length, view, initialView])

  if (view === 'list') {
    return (
      <TenantListModal 
        room={room} 
        onClose={onClose} 
        onAdd={() => setView('add')} 
        onEdit={(t) => {
          setEditingTenant(t)
          setView('edit')
        }} 
      />
    )
  }

  if (view === 'add') {
    return (
      <TenantAddModal 
        room={room}
        onClose={onClose}
        onSuccess={() => {
          fetchTenants(room.id)
          setView('list')
        }}
      />
    )
  }

  if (view === 'edit' && editingTenant) {
    return (
      <TenantEditModal 
        room={room}
        tenant={editingTenant}
        onClose={() => setView('list')}
        onSuccess={() => {
          fetchTenants(room.id)
          setView('list')
        }}
      />
    )
  }

  return null
}
