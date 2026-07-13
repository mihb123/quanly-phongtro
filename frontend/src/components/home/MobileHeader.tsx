import { useState } from 'react'
import { Home, Settings } from '@/components/icons'
import { useAuth } from '@/contexts/AuthContext'
import { UpdateProfileModal } from '@/components/home/modals/UpdateProfileModal'
import { useSelectedStore } from '@/data/selectedData'

export function MobileHeader() {
  const { user } = useAuth()
  const [showUpdateProfile, setShowUpdateProfile] = useState(false)
  const { setActiveTab } = useSelectedStore()

  return (
    <>
      {showUpdateProfile && (
        <UpdateProfileModal onClose={() => setShowUpdateProfile(false)} />
      )}
      
      <header className="md:hidden fixed top-0 left-0 right-0 z-40 bg-card/80 backdrop-blur-xl border-b border-border shadow-sm h-16 safe-top pt-[env(safe-area-inset-top)] flex items-center justify-between px-4">
        {/* Left: Logo */}
        <div className="flex items-center gap-3 overflow-hidden">
          <div className="w-8 h-8 flex-shrink-0 rounded-lg bg-primary flex items-center justify-center shadow-md shadow-primary/20">
            <Home className="w-5 h-5 text-primary-foreground" />
          </div>
          <span className="font-bold text-xl tracking-tight text-foreground whitespace-nowrap">Phòng trọ</span>
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => setActiveTab('settings')}
            className="w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors active:scale-95 cursor-pointer"
          >
            <Settings className="w-5 h-5" />
          </button>

          {user && (
            <button
              onClick={() => setShowUpdateProfile(true)}
              className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0 border border-primary/20 hover:bg-primary/20 transition-colors active:scale-95 cursor-pointer"
            >
              <span className="text-primary font-bold text-lg">
                {user.full_name?.charAt(0).toUpperCase() || user.email.charAt(0).toUpperCase()}
              </span>
            </button>
          )}
        </div>
      </header>
    </>
  )
}
