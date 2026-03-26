import axios from 'axios'

const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
})

export type RegisterPayload = {
  username: string
  password: string
  email: string
}

export type LoginPayload = {
  email: string
  password: string
}

export type AuthResponse = {
  access_token: string
  refresh_token: string
}

export type RefreshTokenPayload = {
  refresh_token: string
}

export const authApi = {
  register: (payload: RegisterPayload) =>
    apiClient.post<AuthResponse>('/auth/register', payload),

  login: (payload: LoginPayload) =>
    apiClient.post<AuthResponse>('/auth/login', payload),

  refreshToken: (payload: RefreshTokenPayload) =>
    apiClient.post<AuthResponse>('/auth/token', payload),
}
