import { useState } from 'react'
import type { Room } from '@/api/room'
import type { Tenant } from '@/api/tenant'
import { TenantListModal } from './TenantListModal'
import { TenantAddModal } from './TenantAddModal'
import { TenantEditModal } from './TenantEditModal'

interface TenantRoomModalProps {
  room: Room
  initialView?: 'list' | 'add' | 'edit'
  initialEditingTenant?: Tenant | null
  onClose: (changed?: boolean) => void
}

export function TenantRoomModal({ room, initialView, initialEditingTenant, onClose }: TenantRoomModalProps) {
  const isOccupied = room.status === 'OCCUPIED'
  
  const [view, setView] = useState<'list' | 'add' | 'edit'>(initialView || (!isOccupied ? 'add' : 'list'))
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(initialEditingTenant || null)
  const [hasChanged, setHasChanged] = useState(false)

  const handleClose = () => {
    onClose(hasChanged)
  }

  if (view === 'list') {
    return (
      <TenantListModal 
        room={room} 
        onClose={handleClose} 
        onAdd={() => setView('add')} 
        onEdit={(t) => {
          setEditingTenant(t)
          setView('edit')
        }}
        onDataChange={() => setHasChanged(true)}
      />
    )
  }

  if (view === 'add') {
    return (
      <TenantAddModal 
        room={room}
        onClose={() => {
          if (initialView === 'add') handleClose()
          else setView('list')
        }}
        onSuccess={() => {
          setHasChanged(true)
          if (initialView === 'add') handleClose()
          else setView('list')
        }}
      />
    )
  }

  if (view === 'edit' && editingTenant) {
    return (
      <TenantEditModal 
        room={room}
        tenant={editingTenant}
        onClose={() => {
          if (initialView === 'edit') handleClose()
          else setView('list')
        }}
        onSuccess={() => {
          setHasChanged(true)
          if (initialView === 'edit') handleClose()
          else setView('list')
        }}
      />
    )
  }

  return null
}
