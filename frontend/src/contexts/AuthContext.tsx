import React, { createContext, useContext, useState, useEffect } from 'react'
import { getMe, logout as logoutApi, refreshToken } from '@/api/auth'
import type { AuthOutput } from '@/types/auth'

interface AuthContextType {
  user: AuthOutput | null
  isLoading: boolean
  login: (user: AuthOutput) => void
  logout: () => void
  updateUser: (user: AuthOutput) => void
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

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

  const login = (user: AuthOutput) => setUser(user)
  const updateUser = (user: AuthOutput) => setUser(user)
  const logout = async () => {
    try {
      await logoutApi()
    } catch (error) {
      console.error('Failed to logout:', error)
    } finally {
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
