import { useState } from 'react'
import { Home, Settings } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useSelectedStore } from '@/data/selectedData'
import { UpdateProfileModal } from '@/components/home/modals/UpdateProfileModal'
import { Button } from '@/components/ui/button'

export function MobileHeader() {
  const { user } = useAuth()
  const { setActiveTab } = useSelectedStore()
  const [showUpdateProfile, setShowUpdateProfile] = useState(false)

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
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setActiveTab('settings')}
            className="w-10 h-10 rounded-full text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
          >
            <Settings className="w-5 h-5" />
          </Button>

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
