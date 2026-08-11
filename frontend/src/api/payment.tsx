import { apiClient } from './client'

export interface PayOSConfigPayload {
  client_id: string
  api_key: string
  checksum_key: string
}

export interface PayOSConfigStatus {
  has_config: boolean
  is_active: boolean
  provider: string
  masked_client_id: string
  webhook_url: string
}

export const getPaymentPublicKey = async (): Promise<{ public_key: string }> => {
  const response = await apiClient.get('/payments/public-key')
  return response.data
}

export const getPayOSConfig = async (): Promise<PayOSConfigStatus> => {
  const response = await apiClient.get('/payments/providers/payos/config')
  return response.data
}

export const savePayOSConfig = async (payload: PayOSConfigPayload) => {
  const response = await apiClient.post('/payments/providers/payos/config', payload)
  return response.data
}

export const deletePayOSConfig = async () => {
  const response = await apiClient.delete('/payments/providers/payos/config')
  return response.data
}

export interface SePayConfigPayload {
  environment: 'production' | 'sandbox'
  bank_short_name: string
  account_number: string
  account_name: string
  code_prefix: string
  webhook_auth_method: string
  webhook_api_key?: string
  webhook_secret?: string
  api_token?: string
}

export interface SePayConfigStatus {
  has_config: boolean
  is_active: boolean
  provider: string
  environment: 'production' | 'sandbox'
  masked_account_number: string
  bank_short_name: string
  code_prefix: string
  webhook_auth_method: string
  webhook_url: string
}

export const getSePayConfig = async (): Promise<SePayConfigStatus> => {
  const response = await apiClient.get('/payments/providers/sepay/config')
  return response.data
}

export interface SePayConfigSaveResult {
  success: boolean
  bank_account_verified: boolean
  account_holder_name?: string
}

export const saveSePayConfig = async (payload: SePayConfigPayload): Promise<SePayConfigSaveResult> => {
  const response = await apiClient.post('/payments/providers/sepay/config', payload)
  return response.data
}

export const deleteSePayConfig = async () => {
  const response = await apiClient.delete('/payments/providers/sepay/config')
  return response.data
}

export interface SePayReconcileResult {
  pages_fetched: number
  scanned: number
  processed: number
  truncated: boolean
}

export const reconcileSePay = async (): Promise<SePayReconcileResult> => {
  const response = await apiClient.post('/payments/providers/sepay/reconcile', {})
  return response.data
}
