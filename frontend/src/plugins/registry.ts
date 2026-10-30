import type { PluginInfo } from '@/api/plugins'
import type { FrontendPlugin } from './sdk'

const modules = import.meta.glob<{ default: FrontendPlugin }>('../../../plugins/*/web/index.tsx')

let builtins: Promise<(FrontendPlugin | null)[]> | null = null

function loadBuiltinPlugins() {
  if (!builtins) {
    builtins = Promise.all(
      Object.entries(modules).map(([path, load]) =>
        load()
          .then((mod) => mod.default)
          .catch((error) => {
            console.error(`Không tải được plugin ${path}`, error)
            return null
          }),
      ),
    )
  }
  return builtins
}

export async function loadInstalledFrontend(plugin: PluginInfo) {
  try {
    const { loadInstalledPlugin } = await import('./runtime')
    return await loadInstalledPlugin(plugin)
  } catch (error) {
    console.error(`Không tải được giao diện tiện ích ${plugin.id}`, error)
    return null
  }
}

export async function loadFrontendPlugins(plugins: PluginInfo[]) {
  const installed = plugins.filter((p) => p.source === 'installed' && p.web && p.enabled)
  const [builtin, external] = await Promise.all([loadBuiltinPlugins(), Promise.all(installed.map(loadInstalledFrontend))])
  const byId = new Map<string, FrontendPlugin>()
  for (const plugin of [...builtin, ...external]) {
    if (plugin?.id) byId.set(plugin.id, plugin)
  }
  return byId
}

export function unloadInstalledFrontend(id: string) {
  import('./runtime')
    .then(({ unloadInstalledPlugin }) => unloadInstalledPlugin(id))
    .catch(() => {})
}
