declare module '@qlpt/plugin' {
  import type { ComponentType } from 'react'

  export const apiVersion: number

  export type PluginSettingType = 'toggle' | 'text' | 'textarea' | 'password' | 'number' | 'select'

  export interface PluginSettingField {
    key: string
    type: PluginSettingType
    label: string
    description?: string
    group?: string
    placeholder?: string
    default?: unknown
    options?: { value: string; label: string }[]
    min?: number
    max?: number
  }

  export interface PluginInfo {
    id: string
    name: string
    version: string
    description: string
    author: string
    homepage?: string
    default_enabled: boolean
    enabled: boolean
    permissions?: string[]
    settings?: PluginSettingField[]
    source: 'builtin' | 'installed'
    web: boolean
    style: boolean
  }

  export interface PluginViewProps {
    plugin: PluginInfo
    openSettings: () => void
  }

  export interface FrontendPlugin {
    id: string
    icon?: ComponentType<{ className?: string }>
    View: ComponentType<PluginViewProps>
  }

  export function definePlugin(plugin: FrontendPlugin): FrontendPlugin

  export interface PluginApi {
    get<T>(path: string): Promise<T>
    post<T>(path: string, body?: unknown): Promise<T>
    put<T>(path: string, body?: unknown): Promise<T>
    patch<T>(path: string, body?: unknown): Promise<T>
    delete<T>(path: string): Promise<T>
    errorMessage(error: unknown, fallback: string): string
  }

  export function createPluginApi(pluginId: string): PluginApi

  export interface PluginConfig {
    values: Record<string, unknown>
    secrets: Record<string, boolean>
  }

  export function usePluginConfig(pluginId: string): {
    config: PluginConfig | null
    error: string
    save: (values: Record<string, unknown>) => Promise<PluginConfig>
    reload: () => Promise<PluginConfig>
  }

  interface ToastOptions {
    description?: string
    duration?: number
    action?: { label: string; onClick: () => void }
  }

  type ToastFn = (message: string, options?: ToastOptions) => string | number

  export const toast: ToastFn & {
    success: ToastFn
    error: ToastFn
    info: ToastFn
    warning: ToastFn
    message: ToastFn
  }

  export function refreshAppData(): void

  type ClassValue = string | number | bigint | boolean | null | undefined | ClassValue[] | Record<string, unknown>

  export function cn(...inputs: ClassValue[]): string
}

declare module '@qlpt/plugin/ui' {
  import type { ComponentProps, ComponentType, ReactElement, ReactNode } from 'react'

  type Icon = ComponentType<{ className?: string }>
  type Render = { render?: ReactElement }

  export const Button: ComponentType<
    ComponentProps<'button'> &
      Render & {
        variant?: 'default' | 'outline' | 'secondary' | 'ghost' | 'destructive' | 'link'
        size?: 'default' | 'xs' | 'sm' | 'lg' | 'icon' | 'icon-xs' | 'icon-sm' | 'icon-lg'
      }
  >
  export const Badge: ComponentType<
    ComponentProps<'span'> & Render & { variant?: 'default' | 'secondary' | 'destructive' | 'outline' | 'ghost' | 'link' }
  >

  export const Card: ComponentType<ComponentProps<'div'> & { size?: 'default' | 'sm' }>
  export const CardHeader: ComponentType<ComponentProps<'div'>>
  export const CardTitle: ComponentType<ComponentProps<'div'>>
  export const CardDescription: ComponentType<ComponentProps<'div'>>
  export const CardAction: ComponentType<ComponentProps<'div'>>
  export const CardContent: ComponentType<ComponentProps<'div'>>
  export const CardFooter: ComponentType<ComponentProps<'div'>>

  export const Input: ComponentType<ComponentProps<'input'>>
  export const Textarea: ComponentType<ComponentProps<'textarea'>>
  export const Label: ComponentType<ComponentProps<'label'>>

  type CheckedProps = {
    checked?: boolean
    defaultChecked?: boolean
    onCheckedChange?: (checked: boolean) => void
    disabled?: boolean
    id?: string
    name?: string
    className?: string
    'aria-label'?: string
  }
  export const Checkbox: ComponentType<CheckedProps>
  export const Switch: ComponentType<CheckedProps & { size?: 'default' | 'sm' }>

  export const Select: ComponentType<{
    value?: string | null
    defaultValue?: string | null
    onValueChange?: (value: string | null) => void
    items?: Record<string, ReactNode> | { value: string; label: ReactNode }[]
    disabled?: boolean
    children?: ReactNode
  }>
  export const SelectTrigger: ComponentType<ComponentProps<'button'> & { size?: 'default' | 'sm' }>
  export const SelectValue: ComponentType<{ placeholder?: ReactNode; className?: string }>
  export const SelectContent: ComponentType<{ className?: string; children?: ReactNode }>
  export const SelectGroup: ComponentType<{ className?: string; children?: ReactNode }>
  export const SelectLabel: ComponentType<{ className?: string; children?: ReactNode }>
  export const SelectItem: ComponentType<{ value: string; className?: string; disabled?: boolean; children?: ReactNode }>
  export const SelectSeparator: ComponentType<{ className?: string }>

  export const Table: ComponentType<ComponentProps<'table'>>
  export const TableHeader: ComponentType<ComponentProps<'thead'>>
  export const TableBody: ComponentType<ComponentProps<'tbody'>>
  export const TableFooter: ComponentType<ComponentProps<'tfoot'>>
  export const TableRow: ComponentType<ComponentProps<'tr'>>
  export const TableHead: ComponentType<ComponentProps<'th'>>
  export const TableCell: ComponentType<ComponentProps<'td'>>
  export const TableCaption: ComponentType<ComponentProps<'caption'>>

  export const Tabs: ComponentType<{
    value?: string
    defaultValue?: string
    onValueChange?: (value: string) => void
    className?: string
    children?: ReactNode
  }>
  export const TabsList: ComponentType<{ variant?: 'default' | 'line'; className?: string; children?: ReactNode }>
  export const TabsTrigger: ComponentType<{ value: string; disabled?: boolean; className?: string; children?: ReactNode }>
  export const TabsContent: ComponentType<{ value: string; className?: string; children?: ReactNode }>

  export const Dialog: ComponentType<{ open?: boolean; onOpenChange?: (open: boolean) => void; children?: ReactNode }>
  export const DialogTrigger: ComponentType<ComponentProps<'button'> & Render>
  export const DialogClose: ComponentType<ComponentProps<'button'> & Render>
  export const DialogContent: ComponentType<ComponentProps<'div'> & { showCloseButton?: boolean }>
  export const DialogHeader: ComponentType<ComponentProps<'div'>>
  export const DialogFooter: ComponentType<ComponentProps<'div'>>
  export const DialogTitle: ComponentType<ComponentProps<'h2'>>
  export const DialogDescription: ComponentType<ComponentProps<'p'>>

  export const Tooltip: ComponentType<{ children?: ReactNode }>
  export const TooltipTrigger: ComponentType<ComponentProps<'button'> & Render>
  export const TooltipContent: ComponentType<{ side?: 'top' | 'right' | 'bottom' | 'left'; className?: string; children?: ReactNode }>

  export const Skeleton: ComponentType<ComponentProps<'div'>>
  export const Separator: ComponentType<{ orientation?: 'horizontal' | 'vertical'; className?: string }>

  export const EmptyState: ComponentType<{ icon?: Icon; title: string; description?: string; action?: ReactNode; className?: string }>
  export const ConfirmDialog: ComponentType<{
    title: string
    message: string
    confirmText?: string
    cancelText?: string
    onConfirm: () => void
    onCancel: () => void
    isLoading?: boolean
    confirmPhrase?: string
  }>
  export const SearchInput: ComponentType<{ value: string; onSearch: (value: string) => void; placeholder?: string; label: string; className?: string }>
  export const SectionCard: ComponentType<{
    title: ReactNode
    icon?: Icon
    action?: ReactNode
    children: ReactNode
    className?: string
    headerClassName?: string
    bodyClassName?: string
  }>
  export const StatCard: ComponentType<{
    label: string
    value: string
    icon: Icon
    tone?: 'default' | 'positive' | 'negative' | 'warning' | 'info'
    sub?: ReactNode
    onClick?: () => void
  }>
}

declare module '@qlpt/plugin/icons' {
  import type { ComponentType, SVGProps } from 'react'

  type Icon = ComponentType<Omit<SVGProps<SVGSVGElement>, 'ref'> & { className?: string; size?: number | string; strokeWidth?: number }>

  export const AlertCircle: Icon
  export const AlertTriangle: Icon
  export const ArrowRight: Icon
  export const Building: Icon
  export const Building2: Icon
  export const Calendar: Icon
  export const Check: Icon
  export const CheckCircle2: Icon
  export const ChevronDown: Icon
  export const ChevronLeft: Icon
  export const ChevronRight: Icon
  export const ChevronUp: Icon
  export const Clock: Icon
  export const Copy: Icon
  export const DollarSign: Icon
  export const DoorOpen: Icon
  export const Download: Icon
  export const Droplets: Icon
  export const Edit: Icon
  export const ExternalLink: Icon
  export const Eye: Icon
  export const EyeOff: Icon
  export const FileIcon: Icon
  export const FileText: Icon
  export const FileUp: Icon
  export const FilterX: Icon
  export const Home: Icon
  export const ImageIcon: Icon
  export const Key: Icon
  export const LayoutDashboard: Icon
  export const Loader2: Icon
  export const Lock: Icon
  export const LogIn: Icon
  export const LogOut: Icon
  export const Mail: Icon
  export const Monitor: Icon
  export const Moon: Icon
  export const MoreHorizontal: Icon
  export const Pencil: Icon
  export const Phone: Icon
  export const Plus: Icon
  export const Puzzle: Icon
  export const Receipt: Icon
  export const RefreshCw: Icon
  export const Save: Icon
  export const Search: Icon
  export const Send: Icon
  export const Settings: Icon
  export const Shield: Icon
  export const Sun: Icon
  export const Trash2: Icon
  export const TrendingDown: Icon
  export const TrendingUp: Icon
  export const Upload: Icon
  export const User: Icon
  export const UserCircle: Icon
  export const UserPlus: Icon
  export const Users: Icon
  export const Wallet: Icon
  export const X: Icon
  export const Zap: Icon
  export const ZoomIn: Icon
}
