import { useEffect } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { applyTheme, readCachedTheme } from '@/lib/theme'
import { applyColorMode, readCachedColorMode, normalizeColorMode } from '@/lib/colorMode'

// ThemeApplier reflects the authenticated user's saved appearance onto the
// document so the choice stays consistent across every device the user logs in
// from. Renders nothing; it only owns the appearance side effects. Before the
// user has loaded (or when logged out) it falls back to the cached value the
// boot script already applied, keeping the two axes independent:
//   - color theme (red/sky) -> <html data-theme="...">
//   - appearance mode (system/light/dark) -> `.dark` class on <html>
export function ThemeApplier() {
  const { user } = useAuth()

  const theme = user?.theme ?? readCachedTheme()
  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  const colorMode = normalizeColorMode(user?.color_mode ?? readCachedColorMode())
  useEffect(() => {
    applyColorMode(colorMode)
    // In 'system' mode keep the UI in sync with live OS light/dark changes.
    if (colorMode !== 'system') return
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const handleOsChange = () => applyColorMode('system')
    media.addEventListener('change', handleOsChange)
    return () => media.removeEventListener('change', handleOsChange)
  }, [colorMode])

  return null
}
