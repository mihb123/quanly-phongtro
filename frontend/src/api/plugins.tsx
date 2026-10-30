import { apiClient } from './client'

export interface PluginInfo {
  id: string
  name: string
  version: string
  description: string
  author: string
  default_enabled: boolean
  enabled: boolean
}

export const listPlugins = async () => {
  const { data } = await apiClient.get('/plugins/')
  return (data.data || []) as PluginInfo[]
}

export const setPluginEnabled = async (id: string, enabled: boolean) => {
  const { data } = await apiClient.put(`/plugins/${id}/enabled`, { enabled })
  return data.data as { enabled: boolean }
}

export const pluginRequest = async <T,>(pluginId: string, method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE', path: string, body?: unknown) => {
  const headers = body instanceof FormData ? { 'Content-Type': 'multipart/form-data' } : undefined
  const { data } = await apiClient.request({ method, url: `/plugins/${pluginId}${path}`, data: body, headers })
  return data.data as T
}

export const apiErrorMessage = (error: unknown, fallback: string) =>
  (error as { response?: { data?: { message?: string } } }).response?.data?.message || fallback
