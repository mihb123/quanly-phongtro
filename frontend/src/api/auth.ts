import { apiClient } from '@/api/client'
import {
  AuthResponse,
  LoginPayload,
  RefreshTokenPayload,
  RegisterPayload,
} from '@/types/auth'

export const register = (payload: RegisterPayload) =>
  apiClient.post<AuthResponse>('/auth/register', payload).then((res) => res.data)

export const login = (payload: LoginPayload) =>
  apiClient.post<AuthResponse>('/auth/login', payload).then((res) => res.data)

export const refreshToken = (payload: RefreshTokenPayload) =>
  apiClient.post<AuthResponse>('/auth/token', payload).then((res) => res.data)
