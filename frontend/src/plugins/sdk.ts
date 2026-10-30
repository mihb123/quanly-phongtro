import type { ComponentType } from 'react'
import { apiErrorMessage, pluginRequest, type PluginInfo } from '@/api/plugins'
import { invalidateQueries } from '@/lib/queryCache'

export type { PluginInfo }
export { toast } from 'sonner'
export { usePluginConfig } from './configStore'

export interface PluginViewProps {
  plugin: PluginInfo
  openSettings: () => void
}

export interface FrontendPlugin {
  id: string
  icon?: ComponentType<{ className?: string }>
  View: ComponentType<PluginViewProps>
}

export function definePlugin(plugin: FrontendPlugin): FrontendPlugin {
  return plugin
}

export function createPluginApi(pluginId: string) {
  return {
    get: <T,>(path: string) => pluginRequest<T>(pluginId, 'GET', path),
    post: <T,>(path: string, body?: unknown) => pluginRequest<T>(pluginId, 'POST', path, body),
    put: <T,>(path: string, body?: unknown) => pluginRequest<T>(pluginId, 'PUT', path, body),
    delete: <T,>(path: string) => pluginRequest<T>(pluginId, 'DELETE', path),
    errorMessage: apiErrorMessage,
  }
}

export function refreshAppData() {
  invalidateQueries()
}
