import ThuChiApp from '@thuchi/App'
import type { Transport } from '@thuchi/types'
import { refreshToken } from '@/api/auth'
import { apiClient } from '@/api/client'
import { useAuth } from '@/contexts/AuthContext'

const isAuthError = (err: unknown) => (err as { response?: { status?: number } })?.response?.status === 401

const transport: Transport = async <T,>(method: 'GET' | 'POST' | 'PUT' | 'DELETE', url: string, data?: unknown) => {
  const send = () => apiClient.request<T>({ method, url, data }).then((r) => r.data)
  try {
    return await send()
  } catch (err) {
    if (!isAuthError(err)) throw err
    await refreshToken()
    return send()
  }
}

export function ThuChiView() {
  const { user } = useAuth()
  if (!user) return null
  return <ThuChiApp key={user.user_id} ownerId={user.user_id} transport={transport} userEmail={user.email} embedded />
}
