import { apiClient } from '@/api/client'
import type {
  AuthResponse,
  LoginPayload,
  RefreshTokenPayload,
  RegisterPayload,
  AuthOutput,
} from '@/types/auth'

export const register = (payload: RegisterPayload) =>
  apiClient.post<AuthOutput>('/auth/register', payload).then((res) => res.data)

export const login = (payload: LoginPayload) =>
  apiClient.post<AuthResponse>('/auth/login', payload).then((res) => res.data)

export const refreshToken = (payload: RefreshTokenPayload) =>
  apiClient.post<AuthResponse>('/auth/token', payload).then((res) => res.data)

export const getMe = () =>
  apiClient.get<AuthOutput>('/auth/me').then((res) => res.data)

export const logout = () =>
  apiClient.post<void>('/auth/logout').then((res) => res.data)
