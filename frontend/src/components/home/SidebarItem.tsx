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
        ? 'bg-purple-600 text-white shadow-md shadow-purple-500/20' 
        : 'text-slate-500 hover:text-slate-800 hover:bg-slate-100/80'
    }`}>
      <span className="w-5 h-5 flex-shrink-0 flex items-center justify-center">
        {React.cloneElement(icon as React.ReactElement<{ className?: string }>, { className: 'w-5 h-5' })}
      </span>
      {!collapsed && <span className="whitespace-nowrap overflow-hidden text-ellipsis">{label}</span>}
    </button>
  )
}
