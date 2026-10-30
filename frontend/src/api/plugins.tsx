import { apiClient } from './client'

export type PluginSettingType = 'toggle' | 'text' | 'textarea' | 'password' | 'number' | 'select'

export interface PluginSettingField {
  key: string
  type: PluginSettingType
  label: string
  description?: string
  group?: string
  placeholder?: string
  default?: unknown
  options?: { value: string; label: string }[]
  min?: number
  max?: number
}

export type PluginSource = 'builtin' | 'installed'

export interface PluginInfo {
  id: string
  name: string
  version: string
  description: string
  author: string
  homepage?: string
  default_enabled: boolean
  enabled: boolean
  permissions?: string[]
  settings?: PluginSettingField[]
  source: PluginSource
  web: boolean
  style: boolean
}

export interface PluginInstaller {
  allowed: boolean
  api_version: number
  max_size: number
  permissions: string[]
}

export interface PluginConfig {
  values: Record<string, unknown>
  secrets: Record<string, boolean>
}

export const listPlugins = async () => {
  const { data } = await apiClient.get('/plugins/')
  return (data.data || []) as PluginInfo[]
}

export const setPluginEnabled = async (id: string, enabled: boolean) => {
  const { data } = await apiClient.put(`/plugins/${id}/enabled`, { enabled })
  return data.data as { enabled: boolean }
}

export const getPluginConfig = async (id: string) => {
  const { data } = await apiClient.get(`/plugins/${id}/config`)
  return data.data as PluginConfig
}

export const updatePluginConfig = async (id: string, values: Record<string, unknown>) => {
  const { data } = await apiClient.put(`/plugins/${id}/config`, { values })
  return data.data as PluginConfig
}

export const getPluginInstaller = async () => {
  const { data } = await apiClient.get('/plugins/install')
  return data.data as PluginInstaller
}

export const installPlugin = async (file: File) => {
  const form = new FormData()
  form.append('file', file)
  const { data } = await apiClient.post('/plugins/install', form, { headers: { 'Content-Type': 'multipart/form-data' } })
  return data.data as PluginInfo
}

export const uninstallPlugin = async (id: string) => {
  await apiClient.delete(`/plugins/${id}`)
}

export const getPluginAsset = async (id: string, file: 'index.js' | 'index.css') => {
  const { data } = await apiClient.get(`/plugins/${id}/assets/${file}`, { responseType: 'text' })
  return data as string
}

export const pluginRequest = async <T,>(pluginId: string, method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE', path: string, body?: unknown) => {
  const headers = body instanceof FormData ? { 'Content-Type': 'multipart/form-data' } : undefined
  const { data } = await apiClient.request({ method, url: `/plugins/${pluginId}${path}`, data: body, headers })
  return data.data as T
}

export const apiErrorMessage = (error: unknown, fallback: string) =>
  (error as { response?: { data?: { message?: string } } }).response?.data?.message || fallback
