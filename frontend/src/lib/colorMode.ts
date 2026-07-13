// Appearance-mode registry: single source of truth for the light/dark/system
// preference. Orthogonal to the color theme (red/sky) in theme.ts — this axis
// only toggles the `.dark` class on <html>. Must stay in sync with the backend
// allow-list (internal/service/auth_service.go).

export type ColorMode = 'system' | 'light' | 'dark'

export const DEFAULT_COLOR_MODE: ColorMode = 'system'

// Storage key shared with the inline boot script in index.html (no-flash).
export const COLOR_MODE_STORAGE_KEY = 'color-mode'

export interface ColorModeOption {
  id: ColorMode
  label: string
}

export const COLOR_MODES: ColorModeOption[] = [
  { id: 'light', label: 'Sáng' },
  { id: 'dark', label: 'Tối' },
  { id: 'system', label: 'Hệ thống' },
]

// normalizeColorMode coerces an arbitrary stored value into a supported mode,
// falling back to the default so the UI never lands in an undefined state.
export function normalizeColorMode(mode?: string | null): ColorMode {
  return COLOR_MODES.some((m) => m.id === mode) ? (mode as ColorMode) : DEFAULT_COLOR_MODE
}

// prefersDark reads the OS-level dark preference (used to resolve 'system').
function prefersDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

// resolveIsDark turns a mode into the concrete dark/light decision.
export function resolveIsDark(mode: ColorMode): boolean {
  return mode === 'dark' || (mode === 'system' && prefersDark())
}

// applyColorMode toggles the `.dark` class on <html> and caches the choice so
// the inline boot script can restore it before React mounts (avoids a flash).
export function applyColorMode(mode?: string | null): void {
  const normalized = normalizeColorMode(mode)
  document.documentElement.classList.toggle('dark', resolveIsDark(normalized))
  try {
    localStorage.setItem(COLOR_MODE_STORAGE_KEY, normalized)
  } catch {
    // Ignore storage failures (private mode / quota); DB stays the source of truth.
  }
}

// readCachedColorMode returns the last-applied mode from storage, used as a
// fallback before the authenticated user (source of truth) has loaded.
export function readCachedColorMode(): ColorMode {
  try {
    return normalizeColorMode(localStorage.getItem(COLOR_MODE_STORAGE_KEY))
  } catch {
    return DEFAULT_COLOR_MODE
  }
}
