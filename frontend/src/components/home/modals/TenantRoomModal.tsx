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

  // Đóng modal; cho phép override "changed" để tránh đọc state hasChanged còn cũ ngay sau khi thêm/sửa thành công.
  const handleClose = (changed?: boolean) => {
    onClose(changed ?? hasChanged)
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
          if (initialView === 'add') handleClose(true)
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
          if (initialView === 'edit') handleClose(true)
          else setView('list')
        }}
        onDataChange={() => setHasChanged(true)}
      />
    )
  }

  return null
}
