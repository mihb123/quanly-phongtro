import React from 'react'

interface SidebarItemProps {
  icon: React.ReactNode
  label: string
  active?: boolean
  onClick?: () => void
  collapsed?: boolean
}

export function SidebarItem({ icon, label, active = false, onClick, collapsed = false }: SidebarItemProps) {
  return (
    <button 
      onClick={onClick}
      title={collapsed ? label : ''}
      className={`flex items-center ${collapsed ? 'justify-center' : 'justify-start'} gap-3 w-full px-4 py-3 rounded-xl transition-all duration-300 font-bold cursor-pointer ${
      active 
        ? 'bg-primary text-primary-foreground shadow-md shadow-primary/20' 
        : 'text-muted-foreground hover:text-foreground hover:bg-secondary'
    }`}>
      <span className="w-5 h-5 flex-shrink-0 flex items-center justify-center">
        {React.cloneElement(icon as React.ReactElement<{ className?: string }>, { className: 'w-5 h-5' })}
      </span>
      {!collapsed && <span className="whitespace-nowrap overflow-hidden text-ellipsis">{label}</span>}
    </button>
  )
}
