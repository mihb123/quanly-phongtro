import { apiClient, setAccessToken } from '@/api/client'
import { ensureDPoPKeyPair } from '@/utils/dpop'
import type {
  AuthResponse,
  LoginPayload,
  RegisterPayload,
  AuthOutput,
} from '@/types/auth'

export const register = async (payload: RegisterPayload) => {
  await ensureDPoPKeyPair()
  return apiClient.post<{ data: AuthOutput }>('/auth/register', payload).then((res) => {
    if (res.data.data.access_token) {
      setAccessToken(res.data.data.access_token)
    }
    return res.data.data
  })
}

export const login = async (payload: LoginPayload) => {
  await ensureDPoPKeyPair()
  return apiClient.post<{ data: AuthResponse }>('/auth/login', payload).then((res) => {
    setAccessToken(res.data.data.access_token)
    return res.data.data
  })
}

let refreshTokenPromise: Promise<AuthResponse> | null = null

export const refreshToken = async () => {
  if (refreshTokenPromise) {
    return refreshTokenPromise
  }
  
  refreshTokenPromise = (async () => {
    try {
      await ensureDPoPKeyPair()
      const res = await apiClient.post<{ data: AuthResponse }>('/auth/token')
      setAccessToken(res.data.data.access_token)
      refreshTokenPromise = null
      return res.data.data
    } catch (err) {
      refreshTokenPromise = null
      throw err
    }
  })()
  
  return refreshTokenPromise
}

export const getMe = () =>
  apiClient.get<{ data: AuthOutput }>('/auth/me').then((res) => res.data.data)

export const logout = () =>
  apiClient.post<{ data: void }>('/auth/logout').then((res) => res.data.data)

export const createOTP = () =>
  apiClient.get('/auth/verify-email').then((res) => res.data)

export const verifyEmailOTP = async (otp: string) => {
  return apiClient.post<{ data: { access_token: string } }>('/auth/verify-email/otp', { otp }).then((res) => {
    if (res.data.data?.access_token) {
      setAccessToken(res.data.data.access_token)
    }
    return res.data.data
  })
}
