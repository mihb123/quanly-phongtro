import { useEffect, useState, type ReactNode } from 'react'
import { AlertCircle, Check, Key, Loader2, Lock, Settings, Trash2 } from '@/components/icons'
import { EmptyState } from '@/components/shared/EmptyState'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { apiErrorMessage, type PluginConfig, type PluginInfo, type PluginSettingField } from '@/api/plugins'
import { usePluginConfig } from '@/plugins/configStore'
import { cn } from '@/lib/utils'

const AUTOSAVE_MS = 800

type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'
type SaveFn = (values: Record<string, unknown>) => Promise<PluginConfig>

function useSaveStatus() {
  const [status, setStatus] = useState<SaveStatus>('idle')
  useEffect(() => {
    if (status !== 'saved') return
    const id = setTimeout(() => setStatus('idle'), 2000)
    return () => clearTimeout(id)
  }, [status])
  return [status, setStatus] as const
}

function SaveIndicator({ status }: { status: SaveStatus }) {
  if (status === 'saving') {
    return <span className="inline-flex items-center gap-1 text-xs text-muted-foreground"><Loader2 className="size-3 animate-spin" />Đang lưu</span>
  }
  if (status === 'saved') {
    return <span className="inline-flex items-center gap-1 text-xs text-success"><Check className="size-3" />Đã lưu</span>
  }
  if (status === 'error') {
    return <span className="inline-flex items-center gap-1 text-xs text-destructive"><AlertCircle className="size-3" />Chưa lưu</span>
  }
  return null
}

function SettingRow({ field, status, error, inline, children }: { field: PluginSettingField; status: SaveStatus; error?: string; inline?: boolean; children: ReactNode }) {
  const id = `plugin-setting-${field.key}`
  return (
    <div className={cn('flex gap-3 px-5 py-4', inline ? 'items-center justify-between' : 'flex-col sm:flex-row sm:items-center sm:justify-between sm:gap-8')}>
      <div className="min-w-0 flex-1 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          <Label htmlFor={id} className="text-sm font-medium">{field.label}</Label>
          <SaveIndicator status={status} />
        </div>
        {field.description ? <p className="text-sm text-muted-foreground">{field.description}</p> : null}
        {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
      </div>
      <div className={cn('flex items-center gap-2', inline ? 'shrink-0' : 'w-full sm:w-72 sm:shrink-0')}>{children}</div>
    </div>
  )
}

function useCommit(field: PluginSettingField, save: SaveFn) {
  const [status, setStatus] = useSaveStatus()
  const [error, setError] = useState('')
  const fail = (message: string) => {
    setStatus('error')
    setError(message)
  }
  const commit = async (value: unknown) => {
    setStatus('saving')
    setError('')
    try {
      await save({ [field.key]: value })
      setStatus('saved')
      return true
    } catch (err) {
      fail(apiErrorMessage(err, `Không lưu được "${field.label}", vui lòng thử lại`))
      return false
    }
  }
  return { status, error, commit, fail }
}

function rangeError(field: PluginSettingField, n: number) {
  if (field.min !== undefined && n < field.min) return `Tối thiểu ${field.min}`
  if (field.max !== undefined && n > field.max) return `Tối đa ${field.max}`
  return ''
}

function ToggleSetting({ field, value, save }: { field: PluginSettingField; value: unknown; save: SaveFn }) {
  const { status, error, commit } = useCommit(field, save)
  const [draft, setDraft] = useState<boolean | null>(null)
  const checked = draft ?? value === true
  return (
    <SettingRow field={field} status={status} error={error} inline>
      <Switch
        id={`plugin-setting-${field.key}`}
        checked={checked}
        disabled={status === 'saving'}
        onCheckedChange={async (next) => {
          setDraft(next)
          await commit(next)
          setDraft(null)
        }}
      />
    </SettingRow>
  )
}

function SelectSetting({ field, value, save }: { field: PluginSettingField; value: unknown; save: SaveFn }) {
  const { status, error, commit } = useCommit(field, save)
  const [draft, setDraft] = useState<string | null>(null)
  const current = draft ?? (typeof value === 'string' ? value : null)
  const items = field.options ?? []
  return (
    <SettingRow field={field} status={status} error={error}>
      <Select
        items={items}
        value={current}
        onValueChange={async (next) => {
          if (next == null || next === current) return
          setDraft(next)
          await commit(next)
          setDraft(null)
        }}
      >
        <SelectTrigger id={`plugin-setting-${field.key}`} className="min-h-11 w-full cursor-pointer text-base md:min-h-9 md:text-sm">
          <SelectValue placeholder={field.placeholder || 'Chọn'} />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {items.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}
          </SelectGroup>
        </SelectContent>
      </Select>
    </SettingRow>
  )
}

const toText = (value: unknown) => (value === undefined || value === null ? '' : String(value))

function TextSetting({ field, value, save }: { field: PluginSettingField; value: unknown; save: SaveFn }) {
  const { status, error, commit, fail } = useCommit(field, save)
  const stored = toText(value)
  const [draft, setDraft] = useState(stored)
  const [synced, setSynced] = useState(stored)
  const [failed, setFailed] = useState<string | null>(null)

  if (stored !== synced) {
    setSynced(stored)
    setDraft(stored)
  }

  const flush = (text: string) => {
    if (text === stored || status === 'saving') return
    let next: unknown = text.trim() === '' ? null : text
    if (field.type === 'number' && next !== null) {
      const n = Number(text)
      const invalid = Number.isFinite(n) ? rangeError(field, n) : 'Vui lòng nhập một số'
      if (invalid) {
        fail(invalid)
        setFailed(text)
        return
      }
      next = n
    }
    void commit(next).then((ok) => setFailed(ok ? null : text))
  }

  useEffect(() => {
    if (draft === stored || draft === failed || status === 'saving') return
    const id = setTimeout(() => flush(draft), AUTOSAVE_MS)
    return () => clearTimeout(id)
  })

  const common = {
    id: `plugin-setting-${field.key}`,
    value: draft,
    placeholder: field.placeholder,
    onChange: (e: { target: { value: string } }) => setDraft(e.target.value),
    onBlur: () => flush(draft),
    'aria-invalid': error ? true : undefined,
  }

  return (
    <SettingRow field={field} status={status} error={error}>
      {field.type === 'textarea' ? (
        <Textarea {...common} rows={3} className="text-base md:text-sm" />
      ) : (
        <Input
          {...common}
          type={field.type === 'number' ? 'number' : 'text'}
          inputMode={field.type === 'number' ? 'decimal' : undefined}
          min={field.min}
          max={field.max}
          className="min-h-11 text-base md:min-h-9 md:text-sm"
          onKeyDown={(e) => {
            if (e.key === 'Enter') e.currentTarget.blur()
          }}
        />
      )}
    </SettingRow>
  )
}

function PasswordSetting({ field, isSet, save }: { field: PluginSettingField; isSet: boolean; save: SaveFn }) {
  const { status, error, commit } = useCommit(field, save)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')
  const busy = status === 'saving'

  const submit = async () => {
    if (!draft) return
    if (await commit(draft)) {
      setEditing(false)
      setDraft('')
    }
  }

  return (
    <SettingRow field={field} status={status} error={error}>
      {editing ? (
        <form
          className="flex w-full items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            void submit()
          }}
        >
          <Input
            id={`plugin-setting-${field.key}`}
            type="password"
            autoComplete="new-password"
            autoFocus
            value={draft}
            placeholder={field.placeholder}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Escape') setEditing(false)
            }}
            className="min-h-11 min-w-0 flex-1 text-base md:min-h-9 md:text-sm"
          />
          <Button type="submit" size="sm" disabled={!draft || busy}>Lưu</Button>
          <Button type="button" size="sm" variant="ghost" disabled={busy} onClick={() => { setEditing(false); setDraft('') }}>Hủy</Button>
        </form>
      ) : isSet ? (
        <div className="flex w-full flex-wrap items-center gap-2 sm:justify-end">
          <span className="mr-auto inline-flex items-center gap-1.5 text-sm text-muted-foreground sm:mr-0">
            <Lock className="size-3.5" />Đã đặt
          </span>
          <Button size="sm" variant="outline" disabled={busy} onClick={() => setEditing(true)} aria-label={`Đổi ${field.label}`}>Đổi</Button>
          <Button size="sm" variant="ghost" disabled={busy} onClick={() => void commit(null)} aria-label={`Xóa ${field.label}`}>
            <Trash2 data-icon="inline-start" />Xóa
          </Button>
        </div>
      ) : (
        <Button size="sm" variant="outline" className="sm:ml-auto" onClick={() => setEditing(true)}>
          <Key data-icon="inline-start" />Đặt mật khẩu
        </Button>
      )}
    </SettingRow>
  )
}

function SettingControl({ field, config, save }: { field: PluginSettingField; config: PluginConfig; save: SaveFn }) {
  const value = config.values[field.key]
  switch (field.type) {
    case 'toggle':
      return <ToggleSetting field={field} value={value} save={save} />
    case 'select':
      return <SelectSetting field={field} value={value} save={save} />
    case 'password':
      return <PasswordSetting field={field} isSet={config.secrets[field.key] === true} save={save} />
    default:
      return <TextSetting field={field} value={value} save={save} />
  }
}

function groupFields(fields: PluginSettingField[]) {
  const groups: { title: string; fields: PluginSettingField[] }[] = []
  for (const field of fields) {
    const title = field.group || ''
    const last = groups[groups.length - 1]
    if (last && last.title === title) last.fields.push(field)
    else groups.push({ title, fields: [field] })
  }
  return groups
}

export function PluginSettingsPanel({ plugin }: { plugin: PluginInfo }) {
  const fields = plugin.settings ?? []
  const { config, error, save, reload } = usePluginConfig(plugin.id)

  if (fields.length === 0) {
    return <Card><EmptyState icon={Settings} title="Tiện ích này không có cài đặt" /></Card>
  }
  if (!config) {
    return error ? (
      <Card>
        <EmptyState
          icon={AlertCircle}
          title="Không tải được cài đặt"
          description={error}
          action={<Button variant="outline" size="sm" onClick={() => reload().catch(() => {})}>Thử lại</Button>}
        />
      </Card>
    ) : (
      <Card className="gap-0 divide-y p-0">
        {fields.map((f) => (
          <div key={f.key} className="flex items-center justify-between gap-6 px-5 py-4">
            <div className="flex-1 space-y-2"><Skeleton className="h-4 w-40" /><Skeleton className="h-3 w-3/4" /></div>
            <Skeleton className="h-8 w-24" />
          </div>
        ))}
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {groupFields(fields).map((group) => (
        <section key={group.title || '_'} className="space-y-2">
          {group.title ? <h3 className="px-1 text-xs font-semibold tracking-wide text-muted-foreground uppercase">{group.title}</h3> : null}
          <Card className="gap-0 divide-y p-0">
            {group.fields.map((field) => <SettingControl key={field.key} field={field} config={config} save={save} />)}
          </Card>
        </section>
      ))}
    </div>
  )
}
