import { useCallback, useEffect, useSyncExternalStore } from 'react'
import { apiErrorMessage, getPluginConfig, updatePluginConfig, type PluginConfig } from '@/api/plugins'

interface ConfigEntry {
  data: PluginConfig | null
  error: string
}

const EMPTY: ConfigEntry = { data: null, error: '' }
const entries = new Map<string, ConfigEntry>()
const inflight = new Map<string, Promise<PluginConfig>>()
const listeners = new Set<() => void>()

function setEntry(id: string, entry: ConfigEntry) {
  entries.set(id, entry)
  listeners.forEach((listener) => listener())
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function loadPluginConfig(id: string) {
  const running = inflight.get(id)
  if (running) return running
  const request = getPluginConfig(id)
    .then((data) => {
      setEntry(id, { data, error: '' })
      return data
    })
    .catch((error) => {
      setEntry(id, { data: entries.get(id)?.data ?? null, error: apiErrorMessage(error, 'Không tải được cài đặt tiện ích') })
      throw error
    })
    .finally(() => inflight.delete(id))
  inflight.set(id, request)
  return request
}

export async function savePluginConfig(id: string, values: Record<string, unknown>) {
  const data = await updatePluginConfig(id, values)
  setEntry(id, { data, error: '' })
  return data
}

export function clearPluginConfigCache() {
  entries.clear()
  listeners.forEach((listener) => listener())
}

export function usePluginConfig(id: string) {
  const entry = useSyncExternalStore(subscribe, () => entries.get(id) ?? EMPTY)
  useEffect(() => {
    loadPluginConfig(id).catch(() => {})
  }, [id])
  const save = useCallback((values: Record<string, unknown>) => savePluginConfig(id, values), [id])
  const reload = useCallback(() => loadPluginConfig(id), [id])
  return { config: entry.data, error: entry.error, save, reload }
}
