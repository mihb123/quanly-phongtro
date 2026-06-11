export type RegisterPayload = {
  password: string
  email: string
  Latitude?: number
  Longitude?: number
}

export type LoginPayload = {
  email: string
  password: string
  Latitude?: number
  Longitude?: number
}

export type AuthResponse = {
  access_token: string
  refresh_token: string
}

export type RefreshTokenPayload = {
  refresh_token: string
}

export type AuthOutput = {
  user_id: string
  email: string
  role: string
  full_name?: string
  phone?: string
  is_activated: boolean
  access_token?: string
}
