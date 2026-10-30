import * as React from 'react'
import * as ReactDOM from 'react-dom'
import * as jsxRuntime from 'react/jsx-runtime'
import * as icons from '@/components/icons'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectSeparator, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCaption, TableCell, TableFooter, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { EmptyState } from '@/components/shared/EmptyState'
import { SearchInput } from '@/components/shared/SearchInput'
import { SectionCard } from '@/components/shared/SectionCard'
import { StatCard } from '@/components/shared/StatCard'
import { getPluginAsset, type PluginInfo } from '@/api/plugins'
import { cn } from '@/lib/utils'
import { createPluginApi, definePlugin, refreshAppData, toast, usePluginConfig, type FrontendPlugin } from './sdk'

export const PLUGIN_API_VERSION = 1

const BUNDLE_EXPORT = '__qlpt_plugin__'

const shared: Record<string, unknown> = {
  __qlpt_react__: React,
  __qlpt_jsx__: jsxRuntime,
  __qlpt_react_dom__: ReactDOM,
  __qlpt_sdk__: { apiVersion: PLUGIN_API_VERSION, cn, createPluginApi, definePlugin, refreshAppData, toast, usePluginConfig },
  __qlpt_ui__: {
    Badge, Button, Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle, Checkbox,
    ConfirmDialog, Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger,
    EmptyState, Input, Label, SearchInput, SectionCard, Select, SelectContent, SelectGroup, SelectItem, SelectLabel,
    SelectSeparator, SelectTrigger, SelectValue, Separator, Skeleton, StatCard, Switch, Table, TableBody, TableCaption,
    TableCell, TableFooter, TableHead, TableHeader, TableRow, Tabs, TabsContent, TabsList, TabsTrigger, Textarea, Tooltip,
    TooltipContent, TooltipTrigger,
  },
  __qlpt_icons__: icons,
}

const loaded = new Map<string, Promise<FrontendPlugin>>()

export function loadInstalledPlugin(info: PluginInfo) {
  const key = `${info.id}@${info.version}`
  let pending = loaded.get(key)
  if (!pending) {
    pending = evaluateBundle(info)
    pending.catch(() => loaded.delete(key))
    loaded.set(key, pending)
  }
  return pending
}

async function evaluateBundle(info: PluginInfo) {
  const [code, css] = await Promise.all([
    getPluginAsset(info.id, 'index.js'),
    info.style ? getPluginAsset(info.id, 'index.css') : Promise.resolve(''),
  ])
  const names = Object.keys(shared)
  const factory = new Function(
    ...names,
    `${code}\n;return typeof ${BUNDLE_EXPORT} === 'undefined' ? undefined : ${BUNDLE_EXPORT};\n//# sourceURL=qlpt-plugin:///${info.id}/index.js`,
  )
  const exported = factory(...names.map((name) => shared[name])) as { default?: FrontendPlugin } | undefined
  const plugin = exported?.default
  if (!plugin || plugin.id !== info.id || !plugin.View) {
    throw new Error(`Bundle của ${info.id} phải export default definePlugin({ id: '${info.id}', View })`)
  }
  applyStyle(info.id, css)
  return plugin
}

function applyStyle(id: string, css: string) {
  document.querySelector(`style[data-plugin="${id}"]`)?.remove()
  if (!css) return
  const style = document.createElement('style')
  style.dataset.plugin = id
  style.textContent = css
  document.head.appendChild(style)
}

export function unloadInstalledPlugin(id: string) {
  for (const key of loaded.keys()) {
    if (key.startsWith(`${id}@`)) loaded.delete(key)
  }
  applyStyle(id, '')
}
