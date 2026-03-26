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

export type AuthOutput = {
  user_id: string
  email: string
  role: string
  full_name?: string
  phone?: string
  is_activated: boolean
}
