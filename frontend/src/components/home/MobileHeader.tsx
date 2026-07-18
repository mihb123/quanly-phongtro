import { useState } from 'react'
import { Home, Settings } from '@/components/icons'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/contexts/AuthContext'
import { UpdateProfileModal } from '@/components/home/modals/UpdateProfileModal'
import { useSelectedStore } from '@/data/selectedData'

export function MobileHeader() {
  const { user } = useAuth()
  const [showUpdateProfile, setShowUpdateProfile] = useState(false)
  const { setActiveTab } = useSelectedStore()

  const userInitial = (user?.full_name?.charAt(0) || user?.email.charAt(0) || '?').toUpperCase()

  return (
    <>
      {showUpdateProfile && (
        <UpdateProfileModal onClose={() => setShowUpdateProfile(false)} />
      )}

      <header className="fixed top-0 left-0 right-0 z-40 flex h-16 items-center justify-between border-b border-border bg-background/80 px-4 backdrop-blur-xl safe-top pt-[env(safe-area-inset-top)] md:hidden">
        {/* Left: Logo */}
        <div className="flex items-center gap-2.5 overflow-hidden">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Home className="size-4" />
          </div>
          <span className="whitespace-nowrap text-base font-semibold tracking-tight text-foreground">Phòng trọ</span>
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setActiveTab('settings')}
            className="text-muted-foreground"
          >
            <Settings />
            <span className="sr-only">Cài đặt</span>
          </Button>

          {user && (
            <button
              onClick={() => setShowUpdateProfile(true)}
              className="cursor-pointer rounded-full transition-opacity active:opacity-80"
            >
              <Avatar className="size-8">
                <AvatarFallback className="text-xs font-medium">{userInitial}</AvatarFallback>
              </Avatar>
              <span className="sr-only">Tài khoản</span>
            </button>
          )}
        </div>
      </header>
    </>
  )
}
