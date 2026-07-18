// Theme registry: single source of truth for the selectable UI color themes.
// Each theme id maps to a `[data-theme="..."]` block in src/index.css and must
// stay in sync with the backend allow-list (internal/service/auth_service.go).

export type ThemeId = 'neutral' | 'lime' | 'sky'

export interface ThemeOption {
  id: ThemeId
  label: string
  description: string
  // Representative brand color used for the swatch preview in the theme picker.
  swatch: string
}

export const DEFAULT_THEME: ThemeId = 'neutral'

// Storage key shared with the inline boot script in index.html (no-flash).
export const THEME_STORAGE_KEY = 'color-theme'

export const THEMES: ThemeOption[] = [
  {
    id: 'neutral',
    label: 'Neutral',
    description: 'Tối giản, tinh tế (mặc định)',
    swatch: 'oklch(0.205 0 0)',
  },
  {
    id: 'lime',
    label: 'Lime',
    description: 'Tông xanh lá mặc định',
    swatch: 'oklch(0.841 0.238 128.85)',
  },
  {
    id: 'sky',
    label: 'Sky',
    description: 'Tông xanh dương tươi',
    swatch: 'oklch(0.5 0.134 242.749)',
  },
]

// normalizeTheme coerces an arbitrary stored value into a supported theme id,
// falling back to the default so the UI never lands in an unstyled state.
export function normalizeTheme(theme?: string | null): ThemeId {
  return THEMES.some((t) => t.id === theme) ? (theme as ThemeId) : DEFAULT_THEME
}

// applyTheme reflects the chosen theme onto <html data-theme="..."> so the
// scoped CSS variables in index.css take effect immediately, and caches it so
// the inline boot script can restore it before React mounts (avoids a flash).
export function applyTheme(theme?: string | null): void {
  const normalized = normalizeTheme(theme)
  document.documentElement.dataset.theme = normalized
  try {
    localStorage.setItem(THEME_STORAGE_KEY, normalized)
  } catch {
    // Ignore storage failures (private mode / quota); DB stays the source of truth.
  }
}

// readCachedTheme returns the last-applied theme from storage, used as a
// fallback before the authenticated user (source of truth) has loaded.
export function readCachedTheme(): ThemeId {
  try {
    return normalizeTheme(localStorage.getItem(THEME_STORAGE_KEY))
  } catch {
    return DEFAULT_THEME
  }
}
