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
  user?: AuthOutput
}

export type RefreshTokenPayload = {
  refresh_token: string
}

export type AuthOutput = {
  user_id: string
  email: string
  role: string
  full_name?: string
  phone: string
  is_activated: boolean
  theme?: string
  color_mode?: string
  access_token?: string
  zalo_bot_token?: string
  is_zalo_bot_active?: boolean
}

export interface UpdateProfilePayload {
  full_name?: string
  phone?: string
  theme?: string
  color_mode?: string
  old_password?: string
  password?: string
}
