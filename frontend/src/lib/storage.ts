export const FILTER_TTL = 24 * 60 * 60 * 1000

interface StoredValue<T> {
  value: T
  savedAt: number
}

export function readStorage<T>(key: string, options: { ttl?: number } = {}): T | null {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return null
    const parsed = JSON.parse(raw) as StoredValue<T> | null
    if (!parsed || typeof parsed !== 'object' || !('value' in parsed)) return null
    if (options.ttl && Date.now() - (parsed.savedAt || 0) > options.ttl) {
      localStorage.removeItem(key)
      return null
    }
    return parsed.value
  } catch {
    return null
  }
}

export function writeStorage<T>(key: string, value: T) {
  try {
    localStorage.setItem(key, JSON.stringify({ value, savedAt: Date.now() } satisfies StoredValue<T>))
  } catch {
    // Chế độ riêng tư hoặc hết dung lượng: bỏ qua, chỉ mất khả năng ghi nhớ.
  }
}

export function removeStorage(key: string) {
  try {
    localStorage.removeItem(key)
  } catch {
    // ignore
  }
}

export function readSessionNumber(key: string): number | null {
  try {
    const raw = sessionStorage.getItem(key)
    const value = raw === null ? NaN : Number(raw)
    return Number.isFinite(value) ? value : null
  } catch {
    return null
  }
}

export function writeSessionNumber(key: string, value: number) {
  try {
    sessionStorage.setItem(key, String(value))
  } catch {
    // ignore
  }
}

export const PAGE_SIZE_OPTIONS = [25, 50, 100] as const
export const DEFAULT_PAGE_SIZE = 25

export function readPageSize(key: string): number {
  const saved = readStorage<number>(key)
  return PAGE_SIZE_OPTIONS.includes(saved as (typeof PAGE_SIZE_OPTIONS)[number]) ? (saved as number) : DEFAULT_PAGE_SIZE
}

export function readStorageRaw(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

export function writeStorageRaw(key: string, value: string | null) {
  try {
    if (value === null) localStorage.removeItem(key)
    else localStorage.setItem(key, value)
  } catch {
    // ignore
  }
}
