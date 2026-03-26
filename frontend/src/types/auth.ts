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
