import { apiClient } from '@/api/client'
import type {
  AuthResponse,
  LoginPayload,
  RefreshTokenPayload,
  RegisterPayload,
  AuthOutput,
} from '@/types/auth'

export const register = (payload: RegisterPayload) =>
  apiClient.post<{ data: AuthOutput }>('/auth/register', payload).then((res) => res.data.data)

export const login = (payload: LoginPayload) =>
  apiClient.post<{ data: AuthResponse }>('/auth/login', payload).then((res) => res.data.data)

export const refreshToken = (payload: RefreshTokenPayload) =>
  apiClient.post<{ data: AuthResponse }>('/auth/token', payload).then((res) => res.data.data)

export const getMe = () =>
  apiClient.get<{ data: AuthOutput }>('/auth/me').then((res) => res.data.data)

export const logout = () =>
  apiClient.post<{ data: void }>('/auth/logout').then((res) => res.data.data)
