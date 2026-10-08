import React, { createContext, useContext, useState, useEffect } from 'react'
import { getMe, logout as logoutApi, refreshToken } from '@/api/auth'
import { clearProtectedFileCache } from '@/api/files'
import { useHouseStore } from '@/data/houseData'
import { useHouseCostStore } from '@/data/houseCostData'
import { useInvoiceStore } from '@/data/invoiceData'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'
import { clearQueryCache } from '@/lib/queryCache'
import { writeStorageRaw } from '@/lib/storage'
import type { AuthOutput } from '@/types/auth'

interface AuthContextType {
  user: AuthOutput | null
  isLoading: boolean
  login: (user: AuthOutput) => void
  logout: () => void
  updateUser: (user: AuthOutput) => void
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

function clearUserData() {
  clearQueryCache()
  clearProtectedFileCache()
  useHouseStore.setState({ houses: [] })
  useRoomStore.setState({ rooms: [], roomsHouseId: null, roomTotal: 0, roomPage: 1, roomSearch: '', roomsLoading: false, roomsError: null })
  useSelectedStore.getState().selectHouse(null)
  useSelectedStore.setState({ isHouseListOpen: false })
  useInvoiceStore.getState().clearInvoiceFilter()
  useHouseCostStore.setState({ costs: {}, summaries: [], selectedHouseIds: [] })
  writeStorageRaw('lastSelectedHouseId_Invoice', null)
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthOutput | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    async function loadUser() {
      try {
        await refreshToken()
        const data = await getMe()
        setUser(data)
      } catch {
        setUser(null)
      } finally {
        setIsLoading(false)
      }
    }
    loadUser()
  }, [])

  const login = (user: AuthOutput) => {
    clearUserData()
    setUser(user)
  }
  const updateUser = (user: AuthOutput) => setUser(user)
  const logout = async () => {
    try {
      await logoutApi()
    } catch (error) {
      console.error('Failed to logout:', error)
    } finally {
      clearUserData()
      setUser(null)
    }
  }

  return (
    <AuthContext.Provider value={{ user, isLoading, login, logout, updateUser }}>
      {children}
    </AuthContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
