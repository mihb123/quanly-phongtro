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
