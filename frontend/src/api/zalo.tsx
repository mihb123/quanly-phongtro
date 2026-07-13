import { apiClient } from './client'

export interface ZaloConfigPayload {
  bot_token: string;
}

export const getZaloPublicKey = async (): Promise<{ public_key: string }> => {
  const response = await apiClient.get('/zalo/public-key')
  return response.data
}

export const saveZaloConfig = async (payload: ZaloConfigPayload) => {
  const response = await apiClient.post('/zalo/config', payload)
  return response.data
}

export const getZaloConfigStatus = async (): Promise<{ has_config: boolean, webhook_url: string, is_linked: boolean, bot_id: string, manager_id: string }> => {
  const response = await apiClient.get('/zalo/config')
  return response.data
}

// sendInvoiceViaZalo asks the backend to deliver an invoice image to the linked Zalo recipients.
export const sendInvoiceViaZalo = async (id: string) => {
  const response = await apiClient.post(`/zalo/invoices/${id}/send`)
  return response.data
}
