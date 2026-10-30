import { memo, useCallback, useEffect, useMemo, useState, type ComponentType } from 'react'
import { toast } from 'sonner'
import { ChevronLeft, Loader2, Puzzle, Settings } from '@/components/icons'
import { PageHeader } from '@/components/shared/PageHeader'
import { EmptyState } from '@/components/shared/EmptyState'
import { SearchInput } from '@/components/shared/SearchInput'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { PluginSettingsPanel } from '@/components/home/plugins/PluginSettingsPanel'
import { apiErrorMessage, listPlugins, setPluginEnabled, type PluginInfo } from '@/api/plugins'
import { useIsMobile } from '@/hooks/use-mobile'
import { readStorageRaw, writeStorageRaw } from '@/lib/storage'
import { cn } from '@/lib/utils'
import { loadFrontendPlugins } from '@/plugins/registry'
import type { FrontendPlugin } from '@/plugins/sdk'

const OPEN_PLUGIN_KEY = 'plugins:open'
const pageKey = (id: string) => `plugins:${id}:tab`
const SEARCH_MIN_PLUGINS = 5

type PluginPageTab = 'use' | 'settings'

let cachedPlugins: PluginInfo[] | null = null
let cachedFrontends = new Map<string, FrontendPlugin>()

type IconType = ComponentType<{ className?: string }>

function PluginIcon({ icon: Icon, className }: { icon: IconType; className?: string }) {
  return (
    <span className={cn('flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground', className)}>
      <Icon className="size-4.5" />
    </span>
  )
}

const hasSettings = (plugin: PluginInfo) => (plugin.settings?.length ?? 0) > 0

function readPageTab(plugin: PluginInfo, hasView: boolean): PluginPageTab {
  const saved = readStorageRaw(pageKey(plugin.id))
  if (saved === 'settings' && hasSettings(plugin)) return 'settings'
  return hasView ? 'use' : 'settings'
}

interface PluginRowProps {
  plugin: PluginInfo
  icon: IconType
  canOpen: boolean
  toggling: boolean
  onToggle: (plugin: PluginInfo, enabled: boolean) => void
  onOpen: (id: string, tab?: PluginPageTab) => void
}

const PluginRow = memo(function PluginRow({ plugin, icon, canOpen, toggling, onToggle, onOpen }: PluginRowProps) {
  const openable = plugin.enabled && canOpen
  return (
    <div className="flex items-start gap-3 px-4 py-4 sm:px-5">
      <PluginIcon icon={icon} className={cn(!plugin.enabled && 'opacity-60')} />
      <div className="min-w-0 flex-1 space-y-1">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          {openable ? (
            <button type="button" className="-my-2 py-2 text-left font-semibold text-foreground hover:underline" onClick={() => onOpen(plugin.id)}>
              {plugin.name}
            </button>
          ) : (
            <span className={cn('font-semibold', plugin.enabled ? 'text-foreground' : 'text-muted-foreground')}>{plugin.name}</span>
          )}
          <Badge variant="secondary" className="font-normal">v{plugin.version}</Badge>
        </div>
        <p className="text-sm text-muted-foreground">{plugin.description}</p>
        {plugin.author ? <p className="text-xs text-muted-foreground">Tác giả: {plugin.author}</p> : null}
      </div>
      <div className="flex shrink-0 items-center gap-1 sm:gap-2">
        {plugin.enabled && hasSettings(plugin) ? (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button variant="ghost" size="icon-sm" aria-label={`Cài đặt ${plugin.name}`} onClick={() => onOpen(plugin.id, 'settings')}>
                  <Settings />
                </Button>
              }
            />
            <TooltipContent>Cài đặt</TooltipContent>
          </Tooltip>
        ) : null}
        <label className="-m-2 flex cursor-pointer p-2.5">
          <Switch
            checked={plugin.enabled}
            disabled={toggling}
            onCheckedChange={(checked) => onToggle(plugin, checked)}
            aria-label={plugin.enabled ? `Tắt ${plugin.name}` : `Bật ${plugin.name}`}
          />
        </label>
      </div>
    </div>
  )
})

function PluginManager({
  plugins,
  frontends,
  togglingId,
  onToggle,
  onOpen,
}: {
  plugins: PluginInfo[]
  frontends: Map<string, FrontendPlugin>
  togglingId: string | null
  onToggle: (plugin: PluginInfo, enabled: boolean) => void
  onOpen: (id: string, tab?: PluginPageTab) => void
}) {
  const [search, setSearch] = useState('')
  const enabledCount = plugins.filter((p) => p.enabled).length
  const visible = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return plugins
    return plugins.filter((p) => `${p.name} ${p.description} ${p.author}`.toLowerCase().includes(q))
  }, [plugins, search])

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-muted-foreground">
          Đang bật <span className="font-medium text-foreground tabular-nums">{enabledCount}</span> / {plugins.length} tiện ích. Tắt tiện ích sẽ ẩn nó và chặn mọi thao tác của nó; cài đặt vẫn được giữ.
        </p>
        {plugins.length >= SEARCH_MIN_PLUGINS ? (
          <SearchInput value={search} onSearch={setSearch} label="Tìm tiện ích" placeholder="Tìm tiện ích..." className="sm:w-64" />
        ) : null}
      </div>
      <Card className="gap-0 divide-y p-0">
        {visible.length === 0 ? (
          <EmptyState icon={Puzzle} title="Không có tiện ích phù hợp" description="Thử từ khóa khác." />
        ) : (
          visible.map((plugin) => {
            const frontend = frontends.get(plugin.id)
            return (
              <PluginRow
                key={plugin.id}
                plugin={plugin}
                icon={frontend?.icon ?? Puzzle}
                canOpen={!!frontend || hasSettings(plugin)}
                toggling={togglingId === plugin.id}
                onToggle={onToggle}
                onOpen={onOpen}
              />
            )
          })
        )}
      </Card>
    </div>
  )
}

function PluginPage({
  plugin,
  frontend,
  toggling,
  onToggle,
  onBack,
}: {
  plugin: PluginInfo
  frontend: FrontendPlugin | undefined
  toggling: boolean
  onToggle: (plugin: PluginInfo, enabled: boolean) => void
  onBack?: () => void
}) {
  const View = frontend?.View
  const settingsAvailable = hasSettings(plugin)
  const [tab, setTab] = useState<PluginPageTab>(() => readPageTab(plugin, !!View))
  const activeTab: PluginPageTab = !View ? 'settings' : tab === 'settings' && !settingsAvailable ? 'use' : tab

  const changeTab = useCallback((next: PluginPageTab) => {
    setTab(next)
    writeStorageRaw(pageKey(plugin.id), next)
  }, [plugin.id])
  const openSettings = useCallback(() => changeTab('settings'), [changeTab])

  return (
    <div className="min-w-0 space-y-5">
      {onBack ? (
        <Button variant="ghost" size="sm" className="-ml-2" onClick={onBack}>
          <ChevronLeft data-icon="inline-start" />Tiện ích
        </Button>
      ) : null}
      <div className="flex items-start gap-3">
        <PluginIcon icon={frontend?.icon ?? Puzzle} className="size-11" />
        <div className="min-w-0 flex-1 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="font-heading text-lg font-semibold text-foreground sm:text-xl">{plugin.name}</h2>
            <Badge variant="secondary" className="font-normal">v{plugin.version}</Badge>
          </div>
          <p className="text-sm text-muted-foreground">{plugin.description}</p>
        </div>
        <label className="flex shrink-0 cursor-pointer items-center gap-2 text-sm text-muted-foreground">
          <span className="hidden sm:inline">{plugin.enabled ? 'Đang bật' : 'Đang tắt'}</span>
          <Switch checked={plugin.enabled} disabled={toggling} onCheckedChange={(checked) => onToggle(plugin, checked)} aria-label={plugin.enabled ? `Tắt ${plugin.name}` : `Bật ${plugin.name}`} />
        </label>
      </div>

      {!plugin.enabled ? (
        <Card>
          <EmptyState
            icon={Puzzle}
            title="Tiện ích đang tắt"
            description="Bật lại để dùng tiện ích và chỉnh cài đặt. Cài đặt cũ vẫn được giữ."
            action={<Button size="sm" disabled={toggling} onClick={() => onToggle(plugin, true)}>Bật tiện ích</Button>}
          />
        </Card>
      ) : (
        <>
          {View && settingsAvailable ? (
            <Tabs value={activeTab} onValueChange={(value) => changeTab(value as PluginPageTab)}>
              <TabsList>
                <TabsTrigger value="use" className="px-4"><Puzzle data-icon="inline-start" />Sử dụng</TabsTrigger>
                <TabsTrigger value="settings" className="px-4"><Settings data-icon="inline-start" />Cài đặt</TabsTrigger>
              </TabsList>
            </Tabs>
          ) : null}
          {activeTab === 'use' && View ? <View plugin={plugin} openSettings={openSettings} /> : <PluginSettingsPanel plugin={plugin} />}
        </>
      )}
    </div>
  )
}

function PluginNav({
  plugins,
  frontends,
  openId,
  onSelect,
}: {
  plugins: PluginInfo[]
  frontends: Map<string, FrontendPlugin>
  openId: string | null
  onSelect: (id: string | null) => void
}) {
  const enabled = plugins.filter((p) => p.enabled && (frontends.has(p.id) || hasSettings(p)))
  const item = (active: boolean) =>
    cn(
      'flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-left text-sm transition-colors',
      active ? 'bg-muted font-medium text-foreground' : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
    )
  return (
    <nav aria-label="Tiện ích" className="sticky top-0 space-y-4">
      <button type="button" className={item(openId === null)} onClick={() => onSelect(null)} aria-current={openId === null ? 'page' : undefined}>
        <Puzzle className="size-4 shrink-0" />
        <span className="flex-1 truncate">Quản lý tiện ích</span>
        <span className="text-xs tabular-nums">{plugins.length}</span>
      </button>
      <div className="space-y-1">
        <p className="px-2.5 text-xs font-semibold tracking-wide text-muted-foreground uppercase">Đang bật</p>
        {enabled.length === 0 ? (
          <p className="px-2.5 py-1 text-sm text-muted-foreground">Chưa bật tiện ích nào</p>
        ) : (
          enabled.map((plugin) => {
            const Icon = frontends.get(plugin.id)?.icon ?? Puzzle
            return (
              <button
                key={plugin.id}
                type="button"
                className={item(openId === plugin.id)}
                onClick={() => onSelect(plugin.id)}
                aria-current={openId === plugin.id ? 'page' : undefined}
                title={plugin.name}
              >
                <Icon className="size-4 shrink-0" />
                <span className="line-clamp-2 flex-1">{plugin.name}</span>
              </button>
            )
          })
        )}
      </div>
    </nav>
  )
}

export function PluginsView() {
  const isMobile = useIsMobile()
  const [plugins, setPlugins] = useState<PluginInfo[] | null>(cachedPlugins)
  const [frontends, setFrontends] = useState<Map<string, FrontendPlugin>>(cachedFrontends)
  const [openId, setOpenId] = useState<string | null>(() => readStorageRaw(OPEN_PLUGIN_KEY))
  const [keepDisabledId, setKeepDisabledId] = useState<string | null>(null)
  const [togglingId, setTogglingId] = useState<string | null>(null)
  const [loadError, setLoadError] = useState('')
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    let active = true
    Promise.all([listPlugins(), loadFrontendPlugins()])
      .then(([list, modules]) => {
        if (!active) return
        cachedPlugins = list
        cachedFrontends = modules
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
  }, [reloadKey])

  const openPlugin = useCallback((id: string | null, tab?: PluginPageTab) => {
    setOpenId(id)
    if (id && tab) writeStorageRaw(pageKey(id), tab)
    writeStorageRaw(OPEN_PLUGIN_KEY, id)
  }, [])

  const toggle = useCallback(async (plugin: PluginInfo, enabled: boolean) => {
    setTogglingId(plugin.id)
    try {
      await setPluginEnabled(plugin.id, enabled)
      setPlugins((prev) => {
        const next = (prev || []).map((p) => (p.id === plugin.id ? { ...p, enabled } : p))
        cachedPlugins = next
        return next
      })
      setKeepDisabledId(enabled ? null : plugin.id)
      if (enabled) {
        toast.success(`Đã bật ${plugin.name}`, { action: { label: 'Mở', onClick: () => openPlugin(plugin.id) } })
      } else {
        toast.success(`Đã tắt ${plugin.name}`)
      }
    } catch (error) {
      toast.error(apiErrorMessage(error, 'Không đổi được trạng thái tiện ích'))
    } finally {
      setTogglingId(null)
    }
  }, [openPlugin])

  if (!plugins) {
    return (
      <div className="space-y-6">
        <PageHeader title="Tiện ích" description="Bật, tắt và cài đặt các tiện ích mở rộng" />
        {loadError ? (
          <Card>
            <EmptyState
              icon={Puzzle}
              title="Không tải được tiện ích"
              description={loadError}
              action={<Button variant="outline" size="sm" onClick={() => setReloadKey((k) => k + 1)}>Thử lại</Button>}
            />
          </Card>
        ) : (
          <div className="flex justify-center py-10 text-muted-foreground"><Loader2 className="size-5 animate-spin" /></div>
        )}
      </div>
    )
  }

  const opened = openId ? plugins.find((p) => p.id === openId) : undefined
  const openedFrontend = opened ? frontends.get(opened.id) : undefined
  const showPage = !!opened && (!!openedFrontend || hasSettings(opened)) && (opened.enabled || keepDisabledId === opened.id)
  const page = showPage && opened ? (
    <PluginPage
      key={opened.id}
      plugin={opened}
      frontend={openedFrontend}
      toggling={togglingId === opened.id}
      onToggle={toggle}
      onBack={isMobile ? () => openPlugin(null) : undefined}
    />
  ) : null

  const manager = plugins.length === 0 ? (
    <Card><EmptyState icon={Puzzle} title="Chưa cài tiện ích nào" /></Card>
  ) : (
    <PluginManager plugins={plugins} frontends={frontends} togglingId={togglingId} onToggle={toggle} onOpen={openPlugin} />
  )

  if (isMobile) {
    return page ?? (
      <div className="space-y-6">
        <PageHeader title="Tiện ích" description="Bật, tắt và cài đặt các tiện ích mở rộng" />
        {manager}
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Tiện ích" description="Bật, tắt và cài đặt các tiện ích mở rộng" />
      <div className="grid grid-cols-[15rem_minmax(0,1fr)] items-start gap-8">
        <PluginNav plugins={plugins} frontends={frontends} openId={page ? openId : null} onSelect={openPlugin} />
        {page ?? manager}
      </div>
    </div>
  )
}
