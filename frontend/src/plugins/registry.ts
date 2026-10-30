import type { FrontendPlugin } from './sdk'

const modules = import.meta.glob<{ default: FrontendPlugin }>('../../../plugins/*/web/index.tsx')

let loaded: Promise<Map<string, FrontendPlugin>> | null = null

export function loadFrontendPlugins() {
  if (!loaded) {
    loaded = Promise.all(
      Object.entries(modules).map(([path, load]) =>
        load()
          .then((mod) => mod.default)
          .catch((error) => {
            console.error(`Không tải được plugin ${path}`, error)
            return null
          }),
      ),
    ).then((plugins) => {
      const byId = new Map<string, FrontendPlugin>()
      for (const plugin of plugins) {
        if (plugin?.id) byId.set(plugin.id, plugin)
      }
      return byId
    })
  }
  return loaded
}
