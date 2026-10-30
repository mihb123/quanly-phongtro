import { memo, useCallback, useMemo, useState, type FormEvent, type ReactNode } from 'react'
import { AlertTriangle, Check, CheckCircle2, FileUp, Loader2, Lock, Receipt, Wallet } from '@/components/icons'
import { SectionCard } from '@/components/shared/SectionCard'
import { StatCard } from '@/components/shared/StatCard'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Field, FieldDescription, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { InvoiceDetailModal } from '@/components/home/modals/InvoiceDetailModal'
import { getInvoiceById, type Invoice } from '@/api/invoice'
import { useIsMobile } from '@/hooks/use-mobile'
import { cn } from '@/lib/utils'
import { formatCurrency } from '@/utils/format'
import { createPluginApi, refreshAppData, toast, usePluginConfig, type PluginViewProps } from '@/plugins/sdk'

const api = createPluginApi('bank-statement')

type ItemStatus = 'paid' | 'matched' | 'already_recorded' | 'invoice_paid' | 'ambiguous' | 'unmatched' | 'failed'

interface InvoiceSummary {
  id: string
  room_name: string
  house_name: string
  period: string
  total_amount: number
  status: string
}

interface ImportItem {
  reference: string
  date: string
  amount: number
  description: string
  status: ItemStatus
  method?: string
  message?: string
  invoice?: InvoiceSummary
  candidates?: InvoiceSummary[]
}

interface ImportResult {
  dry_run: boolean
  files: string[]
  summary: {
    credits: number
    total_credit: number
    paid: number
    matched: number
    already_recorded: number
    invoice_paid: number
    ambiguous: number
    unmatched: number
    failed: number
  }
  items: ImportItem[]
}

type Tone = { label: string; className: string }

const TONE = {
  success: 'bg-success/10 text-success',
  info: 'bg-info/10 text-info',
  warning: 'bg-warning/10 text-warning',
  muted: 'bg-secondary text-secondary-foreground',
  error: 'bg-destructive/10 text-destructive',
}

const STATUS: Record<ItemStatus, (Tone & { note: string }) | null> = {
  matched: { label: 'Chưa thanh toán', className: TONE.info, note: 'Cần cập nhật đã thanh toán' },
  paid: { label: 'Đã cập nhật', className: TONE.success, note: 'Vừa đánh dấu đã thanh toán' },
  invoice_paid: { label: 'HĐ đã thanh toán', className: TONE.muted, note: 'Bỏ qua, không cập nhật lại' },
  already_recorded: { label: 'Đã đối soát', className: TONE.muted, note: 'Lần nhập trước · bỏ qua' },
  ambiguous: { label: 'Cần chọn HĐ', className: TONE.warning, note: 'Xử lý thủ công' },
  failed: { label: 'Lỗi cập nhật', className: TONE.error, note: '' },
  unmatched: null,
}

const PRIORITY: Record<ItemStatus, number> = {
  matched: 0, ambiguous: 0, failed: 0, paid: 1, invoice_paid: 2, already_recorded: 2, unmatched: 3,
}

const formatDate = (iso: string) => new Date(iso).toLocaleDateString('vi-VN')
const formatPeriod = (period: string) => `${period.slice(5)}/${period.slice(0, 4)}`
const monthOf = (iso: string) => {
  const d = new Date(iso)
  return `${String(d.getMonth() + 1).padStart(2, '0')}/${d.getFullYear()}`
}
const roomLabel = (inv: InvoiceSummary) => `${inv.room_name} · ${inv.house_name}`

interface ItemView {
  match: (Tone & { note: string }) | null
  status: (Tone & { note: string }) | null
}

function viewOf(item: ImportItem): ItemView {
  const status = STATUS[item.status]
  const statusView = status ? { ...status, note: item.status === 'failed' ? item.message ?? '' : status.note } : null
  if (item.invoice) {
    const diff = Math.round(item.amount - item.invoice.total_amount)
    return {
      match: diff === 0
        ? { label: 'Khớp đúng', className: TONE.success, note: 'Bằng số tiền hóa đơn' }
        : { label: 'Lệch làm tròn', className: TONE.success, note: `${diff < 0 ? 'Thiếu' : 'Dư'} ${formatCurrency(Math.abs(diff))}` },
      status: statusView,
    }
  }
  if (item.candidates?.length) {
    return {
      match: { label: 'Khớp nhiều HĐ', className: TONE.warning, note: `${item.candidates.length} hóa đơn` },
      status: statusView,
    }
  }
  if (item.status === 'unmatched') {
    return {
      match: { label: 'Không khớp', className: TONE.muted, note: `Không có HĐ kỳ ${monthOf(item.date)}` },
      status: null,
    }
  }
  return { match: null, status: statusView }
}

function TwoLines({ primary, secondary, title, className }: { primary: ReactNode; secondary?: string; title?: string; className?: string }) {
  return (
    <div className={cn('min-w-0', className)} title={title}>
      <div className="truncate">{primary}</div>
      {secondary ? <p className="truncate text-xs text-muted-foreground">{secondary}</p> : null}
    </div>
  )
}

function ToneBadge({ tone }: { tone: Tone }) {
  return <Badge variant="ghost" className={cn('max-w-full font-semibold', tone.className)}>{tone.label}</Badge>
}

function BadgeCell({ tone }: { tone: (Tone & { note: string }) | null }) {
  if (!tone) return <span className="text-muted-foreground">—</span>
  return <TwoLines primary={<ToneBadge tone={tone} />} secondary={tone.note} title={tone.note} />
}

type OpenInvoice = (id: string) => void

const linkClass = 'cursor-pointer underline decoration-dotted underline-offset-4 hover:text-primary hover:decoration-solid'

function InvoiceCell({ item, onOpenInvoice, className }: { item: ImportItem; onOpenInvoice: OpenInvoice; className?: string }) {
  const inv = item.invoice
  if (inv) {
    return (
      <button type="button" className={cn('block w-full min-w-0 text-left', className)} title="Xem chi tiết hóa đơn" onClick={() => onOpenInvoice(inv.id)}>
        <span className={cn('block truncate', linkClass)}>{roomLabel(inv)}</span>
        <span className="block truncate text-xs text-muted-foreground">Kỳ {formatPeriod(inv.period)} · {formatCurrency(inv.total_amount)}</span>
      </button>
    )
  }
  if (item.candidates?.length) {
    return (
      <div className={cn('min-w-0', className)}>
        <p className="truncate">{item.candidates.length} hóa đơn cùng số tiền</p>
        <p className="truncate text-xs text-muted-foreground">
          {item.candidates.map((c, i) => (
            <span key={c.id}>
              {i > 0 ? ', ' : null}
              <button type="button" className={linkClass} title={`Xem hóa đơn ${roomLabel(c)} · kỳ ${formatPeriod(c.period)}`} onClick={() => onOpenInvoice(c.id)}>
                {c.room_name}
              </button>
            </span>
          ))}
        </p>
      </div>
    )
  }
  return <span className="text-muted-foreground">—</span>
}

interface SelectableProps {
  item: ImportItem
  selectable: boolean
  checked: boolean
  onToggle: (reference: string) => void
  onOpenInvoice: OpenInvoice
}

function SelectCheckbox({ item, checked, onToggle }: Pick<SelectableProps, 'item' | 'checked' | 'onToggle'>) {
  if (item.status !== 'matched') return null
  return <Checkbox checked={checked} onCheckedChange={() => onToggle(item.reference)} aria-label={`Chọn giao dịch ${item.reference}`} />
}

const ResultRow = memo(function ResultRow({ item, selectable, checked, onToggle, onOpenInvoice }: SelectableProps) {
  const view = viewOf(item)
  return (
    <TableRow>
      {selectable ? <TableCell className="align-top"><SelectCheckbox item={item} checked={checked} onToggle={onToggle} /></TableCell> : null}
      <TableCell className="align-top">
        <TwoLines primary={<span className="font-medium tabular-nums">{formatCurrency(item.amount)}</span>} secondary={formatDate(item.date)} />
      </TableCell>
      <TableCell className="align-top whitespace-normal" title={`${item.description}\n${item.reference}`}>
        <p className="line-clamp-2 break-words">{item.description || '—'}</p>
      </TableCell>
      <TableCell className="align-top">
        <InvoiceCell item={item} onOpenInvoice={onOpenInvoice} />
      </TableCell>
      <TableCell className="align-top"><BadgeCell tone={view.match} /></TableCell>
      <TableCell className="align-top"><BadgeCell tone={view.status} /></TableCell>
    </TableRow>
  )
})

const ResultCard = memo(function ResultCard({ item, selectable, checked, onToggle, onOpenInvoice }: SelectableProps) {
  const view = viewOf(item)
  return (
    <div className="space-y-2 border-b px-4 py-3 last:border-b-0">
      <div className="flex items-baseline justify-between gap-3">
        <span className="flex items-center gap-3 font-semibold tabular-nums">
          {selectable ? <SelectCheckbox item={item} checked={checked} onToggle={onToggle} /> : null}
          {formatCurrency(item.amount)}
        </span>
        <span className="text-xs text-muted-foreground tabular-nums">{formatDate(item.date)}</span>
      </div>
      <p className="line-clamp-2 text-sm break-words text-muted-foreground">{item.description || '—'}</p>
      {item.invoice || item.candidates?.length ? <InvoiceCell item={item} onOpenInvoice={onOpenInvoice} className="text-sm" /> : null}
      <div className="grid grid-cols-2 gap-3 text-sm">
        <BadgeCell tone={view.match} />
        <BadgeCell tone={view.status} />
      </div>
    </div>
  )
})

interface ResultSectionProps {
  result: ImportResult
  canApply: boolean
  applying: boolean
  selected: Set<string>
  onToggle: (reference: string) => void
  onToggleAll: (references: string[]) => void
  onApply: (references: string[]) => void
}

function ResultSection({ result, canApply, applying, selected, onToggle, onToggleAll, onApply }: ResultSectionProps) {
  const isMobile = useIsMobile()
  const s = result.summary
  const unpaidMatched = s.matched + s.paid + s.failed
  const invoiceMatched = unpaidMatched + s.already_recorded + s.invoice_paid + s.ambiguous
  const skipped = s.already_recorded + s.invoice_paid
  const items = useMemo(
    () => result.items.map((item, index) => ({ item, index })).sort((a, b) => PRIORITY[a.item.status] - PRIORITY[b.item.status] || a.index - b.index).map(({ item }) => item),
    [result.items],
  )
  const matchedRefs = useMemo(() => result.items.filter((i) => i.status === 'matched').map((i) => i.reference), [result.items])
  const selectable = canApply && matchedRefs.length > 0
  const selectedRefs = matchedRefs.filter((ref) => selected.has(ref))
  const allSelected = selectedRefs.length === matchedRefs.length
  const [detail, setDetail] = useState<Invoice | null>(null)
  const openInvoice = useCallback(async (id: string) => {
    toast.loading('Đang tải hóa đơn...', { id: 'bank-statement-invoice' })
    try {
      setDetail(await getInvoiceById(id))
      toast.dismiss('bank-statement-invoice')
    } catch (error) {
      toast.error(api.errorMessage(error, 'Không tải được hóa đơn'), { id: 'bank-statement-invoice' })
    }
  }, [])
  return (
    <>
      {detail ? (
        <InvoiceDetailModal
          invoice={detail}
          onClose={() => setDetail(null)}
          onChanged={() => getInvoiceById(detail.id).then(setDetail).catch(() => {})}
        />
      ) : null}
      <div className="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-3">
        <StatCard label="Khoản tiền vào" value={formatCurrency(s.total_credit)} icon={Wallet} sub={`${s.credits} giao dịch`} />
        <StatCard
          label="Khớp tiền với hóa đơn"
          value={String(invoiceMatched)}
          icon={Receipt}
          sub={s.unmatched ? `${s.unmatched} không khớp hóa đơn nào` : undefined}
        />
        <StatCard
          label="Khớp HĐ chưa thanh toán"
          value={String(unpaidMatched)}
          icon={CheckCircle2}
          tone="positive"
          sub={[
            s.matched ? `${s.matched} cần cập nhật` : '',
            s.paid ? `${s.paid} vừa đánh dấu đã TT` : '',
            s.failed ? `${s.failed} lỗi cập nhật` : '',
          ].filter(Boolean).join(' · ') || undefined}
        />
        <StatCard
          label="Đã thanh toán · bỏ qua"
          value={String(skipped)}
          icon={Check}
          sub={[
            s.already_recorded ? `${s.already_recorded} đã đối soát trước đó` : '',
            s.invoice_paid ? `${s.invoice_paid} HĐ đã thanh toán` : '',
          ].filter(Boolean).join(' · ') || undefined}
        />
        <StatCard label="Cần chọn hóa đơn" value={String(s.ambiguous)} icon={AlertTriangle} tone="warning" />
      </div>

      <SectionCard
        title={result.dry_run ? 'Kết quả xem trước' : 'Kết quả đối soát'}
        icon={FileUp}
        headerClassName="flex-wrap"
        action={selectable ? (
          <div className="flex w-full flex-col-reverse gap-2 sm:w-auto sm:flex-row">
            <Button type="button" variant="outline" size="sm" className="min-h-10 sm:min-h-0" disabled={applying || selectedRefs.length === 0} onClick={() => onApply(selectedRefs)}>
              <Check data-icon="inline-start" />
              Cập nhật đã chọn ({selectedRefs.length})
            </Button>
            <Button type="button" size="sm" className="min-h-10 sm:min-h-0" disabled={applying} onClick={() => onApply(matchedRefs)}>
              {applying ? <Loader2 data-icon="inline-start" className="animate-spin" /> : <CheckCircle2 data-icon="inline-start" />}
              Cập nhật tất cả ({matchedRefs.length})
            </Button>
          </div>
        ) : (
          <span className="hidden truncate text-xs text-muted-foreground sm:inline">{result.files.join(', ')}</span>
        )}
      >
        {isMobile ? (
          <div>
            <div className="flex justify-between border-b bg-muted/40 px-4 py-2 text-sm font-medium">
              <span>Tổng {s.credits} giao dịch</span>
              <span className="tabular-nums">{formatCurrency(s.total_credit)}</span>
            </div>
            {items.map((item) => (
              <ResultCard key={item.reference} item={item} selectable={selectable} checked={selected.has(item.reference)} onToggle={onToggle} onOpenInvoice={openInvoice} />
            ))}
          </div>
        ) : (
          <Table className="table-fixed">
            <colgroup>
              {selectable ? <col className="w-10" /> : null}
              <col className="w-[14%]" />
              <col className="w-[28%]" />
              <col className="w-[22%]" />
              <col className="w-[16%]" />
              <col className="w-[20%]" />
            </colgroup>
            <TableHeader>
              <TableRow>
                {selectable ? (
                  <TableHead>
                    <Checkbox checked={allSelected} onCheckedChange={() => onToggleAll(allSelected ? [] : matchedRefs)} aria-label="Chọn tất cả giao dịch khớp" />
                  </TableHead>
                ) : null}
                <TableHead>Số tiền · Ngày</TableHead>
                <TableHead>Nội dung chuyển khoản</TableHead>
                <TableHead>Hóa đơn cùng tháng</TableHead>
                <TableHead>Khớp tiền</TableHead>
                <TableHead>Trạng thái hóa đơn</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow className="bg-muted/40 font-medium hover:bg-muted/40">
                {selectable ? <TableCell /> : null}
                <TableCell className="tabular-nums">{formatCurrency(s.total_credit)}</TableCell>
                <TableCell colSpan={4} className="text-muted-foreground">Tổng {s.credits} giao dịch ghi có</TableCell>
              </TableRow>
              {items.map((item) => (
                <ResultRow key={item.reference} item={item} selectable={selectable} checked={selected.has(item.reference)} onToggle={onToggle} onOpenInvoice={openInvoice} />
              ))}
            </TableBody>
          </Table>
        )}
      </SectionCard>
    </>
  )
}

export function BankStatementView({ plugin, openSettings }: PluginViewProps) {
  const { config, reload: reloadConfig } = usePluginConfig(plugin.id)
  const [file, setFile] = useState<File | null>(null)
  const [fileInputKey, setFileInputKey] = useState(0)
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [dryRunChoice, setDryRunChoice] = useState<boolean | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [result, setResult] = useState<ImportResult | null>(null)
  const [source, setSource] = useState<{ file: File; password: string } | null>(null)
  const [selected, setSelected] = useState<Set<string>>(() => new Set())
  const [applying, setApplying] = useState(false)

  const toggleSelected = useCallback((reference: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(reference)) next.delete(reference)
      else next.add(reference)
      return next
    })
  }, [])
  const toggleAll = useCallback((references: string[]) => setSelected(new Set(references)), [])

  const hasSavedPassword = config?.secrets.zip_password === true
  const dryRun = dryRunChoice ?? config?.values.preview_first !== false
  const rememberPassword = remember && password !== ''

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (!file || submitting) return
    const form = new FormData()
    form.append('file', file)
    form.append('password', password)
    form.append('remember_password', String(rememberPassword))
    form.append('dry_run', String(dryRun))
    setSubmitting(true)
    try {
      const data = await api.post<ImportResult>('/import', form)
      setResult(data)
      setSource({ file, password })
      setSelected(new Set())
      if (rememberPassword) reloadConfig().catch(() => {})
      setPassword('')
      setFile(null)
      setFileInputKey((k) => k + 1)
      if (!data.dry_run && data.summary.paid > 0) {
        refreshAppData()
        toast.success(`Đã đánh dấu ${data.summary.paid} hóa đơn là đã thanh toán`)
      } else if (data.dry_run) {
        toast.info(`Xem trước: ${data.summary.matched} khoản khớp hóa đơn chưa thanh toán`)
      } else {
        toast.info('Không có hóa đơn nào được cập nhật')
      }
    } catch (error) {
      toast.error(api.errorMessage(error, 'Không đối soát được file sao kê'))
    } finally {
      setSubmitting(false)
    }
  }

  const apply = async (references: string[]) => {
    if (!source || applying || references.length === 0) return
    const form = new FormData()
    form.append('file', source.file)
    form.append('password', source.password)
    form.append('dry_run', 'false')
    references.forEach((ref) => form.append('references', ref))
    setApplying(true)
    try {
      const data = await api.post<ImportResult>('/import', form)
      setResult(data)
      setSelected(new Set())
      if (data.summary.paid > 0) {
        refreshAppData()
        toast.success(`Đã đánh dấu ${data.summary.paid} hóa đơn là đã thanh toán`)
      } else {
        toast.info('Không có hóa đơn nào được cập nhật')
      }
    } catch (error) {
      toast.error(api.errorMessage(error, 'Không cập nhật được hóa đơn'))
    } finally {
      setApplying(false)
    }
  }

  return (
    <div className="space-y-6">
      <SectionCard title="Tải file sao kê" icon={FileUp} bodyClassName="p-5">
        <form onSubmit={submit}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="statement-file">File sao kê (.zip hoặc .pdf)</FieldLabel>
              <Input
                key={fileInputKey}
                id="statement-file"
                type="file"
                accept=".zip,.pdf,application/zip,application/pdf"
                onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              />
              <FieldDescription>
                File zip ngân hàng gửi (có hoặc không có mật khẩu) hoặc file PDF phiếu báo có / sổ phụ. Hiện hỗ trợ TPBank.
              </FieldDescription>
            </Field>

            <Field>
              <FieldLabel htmlFor="statement-password">Mật khẩu giải nén</FieldLabel>
              <Input
                id="statement-password"
                type="password"
                autoComplete="off"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={hasSavedPassword ? 'Để trống để dùng mật khẩu đã lưu' : 'Để trống nếu file không có mật khẩu'}
              />
              {hasSavedPassword ? (
                <FieldDescription>
                  <Lock className="mr-1 inline size-3.5 align-[-2px]" />
                  Đã có mật khẩu lưu sẵn, để trống ô trên là dùng mật khẩu đó.{' '}
                  <button type="button" className="font-medium text-foreground underline underline-offset-4" onClick={openSettings}>
                    Quản lý trong Cài đặt
                  </button>
                </FieldDescription>
              ) : null}
            </Field>

            {password ? (
              <Field orientation="horizontal">
                <Checkbox id="statement-remember" checked={remember} onCheckedChange={(checked) => setRemember(checked === true)} />
                <FieldLabel htmlFor="statement-remember" className="font-normal">
                  {hasSavedPassword ? 'Thay mật khẩu đã lưu bằng mật khẩu này' : 'Lưu mật khẩu cho lần sau (mã hóa trên máy chủ)'}
                </FieldLabel>
              </Field>
            ) : null}

            <Field orientation="horizontal">
              <Checkbox id="statement-dry-run" checked={dryRun} onCheckedChange={(checked) => setDryRunChoice(checked === true)} />
              <FieldLabel htmlFor="statement-dry-run" className="font-normal">Chỉ xem trước, chưa cập nhật hóa đơn</FieldLabel>
            </Field>

            <div>
              <Button type="submit" disabled={!file || submitting}>
                {submitting ? <Loader2 data-icon="inline-start" className="animate-spin" /> : <FileUp data-icon="inline-start" />}
                {dryRun ? 'Xem trước đối soát' : 'Đối soát & cập nhật'}
              </Button>
            </div>
          </FieldGroup>
        </form>
      </SectionCard>

      {result ? (
        <ResultSection
          result={result}
          canApply={source !== null}
          applying={applying}
          selected={selected}
          onToggle={toggleSelected}
          onToggleAll={toggleAll}
          onApply={apply}
        />
      ) : null}
    </div>
  )
}
