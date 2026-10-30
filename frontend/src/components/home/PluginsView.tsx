import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { ChevronLeft, ExternalLink, Loader2, Puzzle } from '@/components/icons'
import { PageHeader } from '@/components/shared/PageHeader'
import { EmptyState } from '@/components/shared/EmptyState'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { apiErrorMessage, listPlugins, setPluginEnabled, type PluginInfo } from '@/api/plugins'
import { loadFrontendPlugins } from '@/plugins/registry'
import type { FrontendPlugin } from '@/plugins/sdk'
import { readStorageRaw, writeStorageRaw } from '@/lib/storage'

const OPEN_PLUGIN_KEY = 'plugins:open'

let cachedPlugins: PluginInfo[] | null = null

export function PluginsView() {
  const [plugins, setPlugins] = useState<PluginInfo[] | null>(cachedPlugins)
  const [frontends, setFrontends] = useState<Map<string, FrontendPlugin>>(() => new Map())
  const [openId, setOpenId] = useState<string | null>(() => readStorageRaw(OPEN_PLUGIN_KEY))
  const [togglingId, setTogglingId] = useState<string | null>(null)
  const [loadError, setLoadError] = useState('')

  useEffect(() => {
    let active = true
    Promise.all([listPlugins(), loadFrontendPlugins()])
      .then(([list, modules]) => {
        if (!active) return
        cachedPlugins = list
        setPlugins(list)
        setFrontends(modules)
        setLoadError('')
      })
      .catch((error) => {
        if (active) setLoadError(apiErrorMessage(error, 'Không tải được danh sách tiện ích'))
      })
    return () => {
      active = false
    }
  }, [])

  const openPlugin = (id: string | null) => {
    setOpenId(id)
    writeStorageRaw(OPEN_PLUGIN_KEY, id)
  }

  const toggle = async (plugin: PluginInfo, enabled: boolean) => {
    setTogglingId(plugin.id)
    try {
      await setPluginEnabled(plugin.id, enabled)
      setPlugins((prev) => {
        const next = (prev || []).map((p) => (p.id === plugin.id ? { ...p, enabled } : p))
        cachedPlugins = next
        return next
      })
      toast.success(enabled ? `Đã bật ${plugin.name}` : `Đã tắt ${plugin.name}`)
    } catch (error) {
      toast.error(apiErrorMessage(error, 'Không đổi được trạng thái tiện ích'))
    } finally {
      setTogglingId(null)
    }
  }

  const opened = plugins?.find((p) => p.id === openId && p.enabled)
  const OpenedView = opened ? frontends.get(opened.id)?.View : undefined

  if (opened && OpenedView) {
    return (
      <div className="space-y-6">
        <PageHeader
          title={opened.name}
          description={opened.description}
          action={
            <Button variant="outline" size="sm" onClick={() => openPlugin(null)}>
              <ChevronLeft data-icon="inline-start" />
              Tiện ích
            </Button>
          }
        />
        <OpenedView plugin={opened} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Tiện ích" description="Bật/tắt và mở các plugin mở rộng của ứng dụng" />

      {loadError && !plugins ? (
        <Card><EmptyState icon={Puzzle} title="Không tải được tiện ích" description={loadError} /></Card>
      ) : !plugins ? (
        <div className="flex justify-center py-10 text-muted-foreground">
          <Loader2 className="size-5 animate-spin" />
        </div>
      ) : plugins.length === 0 ? (
        <Card><EmptyState icon={Puzzle} title="Chưa cài plugin nào" /></Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {plugins.map((plugin) => {
            const frontend = frontends.get(plugin.id)
            const Icon = frontend?.icon ?? Puzzle
            return (
              <Card key={plugin.id} className="gap-4 p-5">
                <div className="flex items-start gap-3">
                  <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Icon className="size-5" />
                  </span>
                  <div className="min-w-0 flex-1 space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="font-semibold text-foreground">{plugin.name}</h3>
                      <Badge variant="secondary">v{plugin.version}</Badge>
                    </div>
                    <p className="text-sm text-muted-foreground">{plugin.description}</p>
                    {plugin.author ? <p className="text-xs text-muted-foreground">Tác giả: {plugin.author}</p> : null}
                  </div>
                  <Switch
                    checked={plugin.enabled}
                    disabled={togglingId === plugin.id}
                    onCheckedChange={(checked) => toggle(plugin, checked)}
                    aria-label={plugin.enabled ? `Tắt ${plugin.name}` : `Bật ${plugin.name}`}
                  />
                </div>
                <div className="flex justify-end">
                  <Button size="sm" disabled={!plugin.enabled || !frontend} onClick={() => openPlugin(plugin.id)}>
                    <ExternalLink data-icon="inline-start" />
                    Mở
                  </Button>
                </div>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
