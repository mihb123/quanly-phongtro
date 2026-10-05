import { useCallback, useEffect, useRef, useState } from 'react'
import { create } from 'zustand'

interface Entry {
  data?: unknown
  error?: unknown
  updatedAt: number
}

interface QueryCacheState {
  entries: Record<string, Entry>
}

const MAX_ENTRIES = 60

const useQueryCacheStore = create<QueryCacheState>(() => ({ entries: {} }))

const inflight = new Map<string, { promise: Promise<unknown>; generation: number }>()
const generations = new Map<string, number>()
const fetchers = new Map<string, () => Promise<unknown>>()
const subscribers = new Map<string, number>()
const lastUsed = new Map<string, number>()

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Object.prototype.toString.call(value) === '[object Object]'
}

// Giữ nguyên tham chiếu cũ cho mọi nhánh không đổi để memo dòng/bảng không render lại.
export function replaceEqualDeep<T>(prev: unknown, next: T): T {
  if (prev === next) return prev as T
  const bothArrays = Array.isArray(prev) && Array.isArray(next)
  if (!bothArrays && !(isPlainObject(prev) && isPlainObject(next))) return next

  const prevItems = prev as Record<string, unknown>
  const nextItems = next as Record<string, unknown>
  const keys = bothArrays ? (next as unknown[]).map((_, i) => String(i)) : Object.keys(nextItems)
  const prevSize = bothArrays ? (prev as unknown[]).length : Object.keys(prevItems).length
  const copy: Record<string, unknown> | unknown[] = bothArrays ? [] : {}
  let equalItems = 0

  for (const key of keys) {
    const value = replaceEqualDeep(prevItems[key], nextItems[key])
    ;(copy as Record<string, unknown>)[key] = value
    if (value === prevItems[key] && (key in prevItems)) equalItems++
  }

  return (prevSize === keys.length && equalItems === prevSize ? prev : copy) as T
}

export function queryKey(scope: string, params: Record<string, unknown> = {}): string {
  const sorted = Object.keys(params)
    .sort()
    .reduce<Record<string, unknown>>((acc, key) => {
      const value = params[key]
      if (value !== undefined && value !== null && value !== '') acc[key] = value
      return acc
    }, {})
  return `${scope}:${JSON.stringify(sorted)}`
}

function prune(entries: Record<string, Entry>) {
  const keys = Object.keys(entries)
  if (keys.length <= MAX_ENTRIES) return entries
  const removable = keys
    .filter(key => !subscribers.get(key))
    .sort((a, b) => (lastUsed.get(a) ?? 0) - (lastUsed.get(b) ?? 0))
  const next = { ...entries }
  for (const key of removable.slice(0, keys.length - MAX_ENTRIES)) {
    delete next[key]
    lastUsed.delete(key)
    fetchers.delete(key)
  }
  return next
}

function writeEntry(key: string, patch: Partial<Entry>) {
  useQueryCacheStore.setState(state => ({
    entries: prune({ ...state.entries, [key]: { ...state.entries[key], ...patch, updatedAt: Date.now() } }),
  }))
}

export function getQueryData<T>(key: string): T | undefined {
  return useQueryCacheStore.getState().entries[key]?.data as T | undefined
}

export function setQueryData<T>(key: string, updater: T | ((prev: T | undefined) => T)) {
  const prev = getQueryData<T>(key)
  const next = typeof updater === 'function' ? (updater as (p: T | undefined) => T)(prev) : updater
  const merged = replaceEqualDeep(prev, next)
  if (merged === prev) return
  writeEntry(key, { data: merged, error: undefined })
}

export function updateQueries<T>(prefix: string, updater: (prev: T) => T) {
  const { entries } = useQueryCacheStore.getState()
  for (const key of Object.keys(entries)) {
    if (key.startsWith(prefix) && entries[key].data !== undefined) {
      setQueryData<T>(key, prev => (prev === undefined ? prev as unknown as T : updater(prev)))
    }
  }
}

export function fetchQuery<T>(key: string, fn: () => Promise<T>): Promise<T> {
  const generation = generations.get(key) ?? 0
  const running = inflight.get(key)
  if (running && running.generation === generation) return running.promise as Promise<T>

  const promise = fn().then(
    data => {
      if ((generations.get(key) ?? 0) === generation) {
        const prev = useQueryCacheStore.getState().entries[key]
        const merged = replaceEqualDeep(prev?.data, data)
        if (merged !== prev?.data || prev?.error !== undefined) writeEntry(key, { data: merged, error: undefined })
      }
      return data
    },
    error => {
      if ((generations.get(key) ?? 0) === generation) writeEntry(key, { error })
      throw error
    },
  ).finally(() => {
    if (inflight.get(key)?.promise === promise) inflight.delete(key)
  })

  inflight.set(key, { promise, generation })
  return promise
}

// Đánh dấu dữ liệu cũ (không xóa) rồi tải lại ngầm các key đang hiển thị; response của request cũ bị bỏ qua.
export function invalidateQueries(prefix = '') {
  const keys = new Set([...Object.keys(useQueryCacheStore.getState().entries), ...fetchers.keys(), ...inflight.keys()])
  for (const key of keys) {
    if (!key.startsWith(prefix)) continue
    generations.set(key, (generations.get(key) ?? 0) + 1)
    inflight.delete(key)
    const fetcher = fetchers.get(key)
    if (fetcher && subscribers.get(key)) fetchQuery(key, fetcher).catch(() => {})
  }
}

export function clearQueryCache() {
  const keys = new Set([...fetchers.keys(), ...inflight.keys()])
  for (const key of keys) generations.set(key, (generations.get(key) ?? 0) + 1)
  inflight.clear()
  useQueryCacheStore.setState({ entries: {} })
}

interface UseQueryOptions {
  enabled?: boolean
  keepPrevious?: boolean
}

export function useQuery<T>(key: string, fn: () => Promise<T>, options: UseQueryOptions = {}) {
  const { enabled = true, keepPrevious = true } = options
  const entry = useQueryCacheStore(state => state.entries[key])
  const fnRef = useRef(fn)
  const [shown, setShown] = useState<{ key: string; data: T } | null>(null)

  useEffect(() => {
    fnRef.current = fn
  })

  useEffect(() => {
    if (!enabled) return
    const fetcher = () => fnRef.current()
    fetchers.set(key, fetcher)
    subscribers.set(key, (subscribers.get(key) ?? 0) + 1)
    lastUsed.set(key, Date.now())
    fetchQuery(key, fetcher).catch(() => {})
    return () => {
      subscribers.set(key, Math.max(0, (subscribers.get(key) ?? 1) - 1))
      lastUsed.set(key, Date.now())
    }
  }, [key, enabled])

  const entryData = entry?.data as T | undefined
  if (entryData !== undefined && (shown?.key !== key || shown.data !== entryData)) {
    setShown({ key, data: entryData })
  }

  const data = entryData ?? (keepPrevious ? shown?.data : undefined)
  const isPlaceholder = entryData === undefined && data !== undefined

  const refetch = useCallback(() => {
    const fetcher = fetchers.get(key)
    if (!fetcher) return Promise.resolve(undefined)
    generations.set(key, (generations.get(key) ?? 0) + 1)
    return fetchQuery(key, fetcher).catch(() => undefined)
  }, [key])

  return {
    data,
    error: entry?.error,
    isLoading: enabled && data === undefined && entry?.error === undefined,
    isPlaceholder,
    refetch,
  }
}
