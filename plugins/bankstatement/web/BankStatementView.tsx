import { memo, useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { AlertTriangle, Check, CheckCircle2, FileUp, Loader2, Receipt, Trash2, Wallet } from '@/components/icons'
import { SectionCard } from '@/components/shared/SectionCard'
import { StatCard } from '@/components/shared/StatCard'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Field, FieldDescription, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useIsMobile } from '@/hooks/use-mobile'
import { cn } from '@/lib/utils'
import { formatCurrency } from '@/utils/format'
import { createPluginApi, refreshAppData, toast } from '@/plugins/sdk'

const api = createPluginApi('bank-statement')

type ItemStatus = 'paid' | 'matched' | 'already_recorded' | 'ambiguous' | 'unmatched' | 'failed'

interface InvoiceSummary {
  id: string
  room_name: string
  house_name: string
  period: string
  total_amount: number
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
    ambiguous: number
    unmatched: number
    failed: number
  }
  items: ImportItem[]
}

const STATUS: Record<ItemStatus, { label: string; className: string }> = {
  paid: { label: 'Đã đánh dấu thanh toán', className: 'bg-success/10 text-success' },
  matched: { label: 'Sẽ đánh dấu thanh toán', className: 'bg-info/10 text-info' },
  already_recorded: { label: 'Đã đối soát trước', className: 'bg-secondary text-secondary-foreground' },
  ambiguous: { label: 'Cần xác nhận', className: 'bg-warning/10 text-warning' },
  unmatched: { label: 'Không khớp hóa đơn', className: 'bg-secondary text-secondary-foreground' },
  failed: { label: 'Lỗi cập nhật', className: 'bg-destructive/10 text-destructive' },
}

const formatDate = (iso: string) => new Date(iso).toLocaleDateString('vi-VN')
const formatPeriod = (period: string) => `${period.slice(5)}/${period.slice(0, 4)}`
const describeInvoice = (inv: InvoiceSummary) => `${inv.room_name} · ${inv.house_name} · kỳ ${formatPeriod(inv.period)}`

function ItemStatusBadge({ status }: { status: ItemStatus }) {
  const conf = STATUS[status]
  return <Badge variant="ghost" className={cn('font-semibold', conf.className)}>{conf.label}</Badge>
}

function ItemOutcome({ item }: { item: ImportItem }) {
  const diff = item.invoice ? Math.round(item.amount - item.invoice.total_amount) : 0
  return (
    <div className="min-w-0 space-y-1">
      <ItemStatusBadge status={item.status} />
      {item.invoice ? <p className="text-sm text-foreground">{describeInvoice(item.invoice)}</p> : null}
      {diff !== 0 && item.invoice ? (
        <p className="text-xs text-warning">
          {diff < 0 ? 'Chuyển thiếu' : 'Chuyển dư'} {formatCurrency(Math.abs(diff))} so với hóa đơn {formatCurrency(item.invoice.total_amount)}
        </p>
      ) : null}
      {item.candidates?.length ? (
        <ul className="space-y-0.5 text-xs text-muted-foreground">
          {item.candidates.map((c) => <li key={c.id}>{describeInvoice(c)}</li>)}
        </ul>
      ) : null}
      {item.message && !item.candidates?.length ? <p className="text-xs text-muted-foreground">{item.message}</p> : null}
    </div>
  )
}

interface SelectableProps {
  item: ImportItem
  selectable: boolean
  checked: boolean
  onToggle: (reference: string) => void
}

function SelectCheckbox({ item, checked, onToggle }: Omit<SelectableProps, 'selectable'>) {
  if (item.status !== 'matched') return null
  return <Checkbox checked={checked} onCheckedChange={() => onToggle(item.reference)} aria-label={`Chọn giao dịch ${item.reference}`} />
}

const ResultRow = memo(function ResultRow({ item, selectable, checked, onToggle }: SelectableProps) {
  return (
    <TableRow>
      {selectable ? <TableCell className="align-top"><SelectCheckbox item={item} checked={checked} onToggle={onToggle} /></TableCell> : null}
      <TableCell className="align-top tabular-nums">{formatDate(item.date)}</TableCell>
      <TableCell className="text-right align-top font-medium tabular-nums">{formatCurrency(item.amount)}</TableCell>
      <TableCell className="align-top whitespace-normal">
        <p className="break-words">{item.description || '—'}</p>
        <p className="text-xs text-muted-foreground">{item.reference}</p>
      </TableCell>
      <TableCell className="align-top whitespace-normal"><ItemOutcome item={item} /></TableCell>
    </TableRow>
  )
})

const ResultCard = memo(function ResultCard({ item, selectable, checked, onToggle }: SelectableProps) {
  return (
    <div className="space-y-2 border-b px-4 py-3 last:border-b-0">
      <div className="flex items-baseline justify-between gap-3">
        <span className="flex items-center gap-3 font-semibold tabular-nums">
          {selectable ? <SelectCheckbox item={item} checked={checked} onToggle={onToggle} /> : null}
          {formatCurrency(item.amount)}
        </span>
        <span className="text-xs text-muted-foreground tabular-nums">{formatDate(item.date)}</span>
      </div>
      <p className="text-sm break-words text-muted-foreground">{item.description || '—'}</p>
      <ItemOutcome item={item} />
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
  const settled = result.dry_run ? s.matched : s.paid
  const matchedRefs = useMemo(() => result.items.filter((i) => i.status === 'matched').map((i) => i.reference), [result.items])
  const selectable = canApply && matchedRefs.length > 0
  const selectedRefs = matchedRefs.filter((ref) => selected.has(ref))
  const allSelected = selectedRefs.length === matchedRefs.length
  return (
    <>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-3">
        <StatCard label="Khoản tiền vào" value={formatCurrency(s.total_credit)} icon={Wallet} sub={`${s.credits} giao dịch`} />
        <StatCard
          label={result.dry_run ? 'Sẽ đánh dấu đã thanh toán' : 'Đã đánh dấu thanh toán'}
          value={String(settled)}
          icon={CheckCircle2}
          tone="positive"
          sub={s.already_recorded ? `${s.already_recorded} đã đối soát trước` : undefined}
        />
        <StatCard label="Cần xác nhận thủ công" value={String(s.ambiguous)} icon={AlertTriangle} tone="warning" />
        <StatCard
          label="Không khớp hóa đơn"
          value={String(s.unmatched + s.failed)}
          icon={Receipt}
          sub={s.failed ? `${s.failed} lỗi cập nhật` : undefined}
        />
      </div>

      <SectionCard
        title={result.dry_run ? 'Kết quả xem trước' : 'Kết quả đối soát'}
        icon={FileUp}
        headerClassName="flex-wrap"
        action={selectable ? (
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" size="sm" disabled={applying || selectedRefs.length === 0} onClick={() => onApply(selectedRefs)}>
              <Check data-icon="inline-start" />
              Cập nhật đã chọn ({selectedRefs.length})
            </Button>
            <Button type="button" size="sm" disabled={applying} onClick={() => onApply(matchedRefs)}>
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
            {result.items.map((item) => (
              <ResultCard key={item.reference} item={item} selectable={selectable} checked={selected.has(item.reference)} onToggle={onToggle} />
            ))}
          </div>
        ) : (
          <Table className="table-fixed">
            <colgroup>
              {selectable ? <col className="w-10" /> : null}
              <col className="w-[12%]" />
              <col className="w-[16%]" />
              <col className="w-[40%]" />
              <col className="w-[32%]" />
            </colgroup>
            <TableHeader>
              <TableRow>
                {selectable ? (
                  <TableHead>
                    <Checkbox checked={allSelected} onCheckedChange={() => onToggleAll(allSelected ? [] : matchedRefs)} aria-label="Chọn tất cả giao dịch khớp" />
                  </TableHead>
                ) : null}
                <TableHead>Ngày</TableHead>
                <TableHead className="text-right">Số tiền</TableHead>
                <TableHead>Nội dung chuyển khoản</TableHead>
                <TableHead>Kết quả</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow className="bg-muted/40 font-medium hover:bg-muted/40">
                {selectable ? <TableCell /> : null}
                <TableCell>Tổng</TableCell>
                <TableCell className="text-right tabular-nums">{formatCurrency(s.total_credit)}</TableCell>
                <TableCell colSpan={2} className="text-muted-foreground">{s.credits} giao dịch ghi có</TableCell>
              </TableRow>
              {result.items.map((item) => (
                <ResultRow key={item.reference} item={item} selectable={selectable} checked={selected.has(item.reference)} onToggle={onToggle} />
              ))}
            </TableBody>
          </Table>
        )}
      </SectionCard>
    </>
  )
}

export function BankStatementView() {
  const [file, setFile] = useState<File | null>(null)
  const [fileInputKey, setFileInputKey] = useState(0)
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(false)
  const [dryRun, setDryRun] = useState(false)
  const [hasSavedPassword, setHasSavedPassword] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [forgetting, setForgetting] = useState(false)
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

  useEffect(() => {
    let active = true
    api.get<{ has_saved_password: boolean }>('/settings')
      .then((data) => {
        if (active) setHasSavedPassword(data.has_saved_password)
      })
      .catch(() => {})
    return () => {
      active = false
    }
  }, [])

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (!file || submitting) return
    const form = new FormData()
    form.append('file', file)
    form.append('password', password)
    form.append('remember_password', String(remember && password !== ''))
    form.append('dry_run', String(dryRun))
    setSubmitting(true)
    try {
      const data = await api.post<ImportResult>('/import', form)
      setResult(data)
      setSource({ file, password })
      setSelected(new Set())
      if (remember && password) setHasSavedPassword(true)
      setPassword('')
      setFile(null)
      setFileInputKey((k) => k + 1)
      if (!data.dry_run && data.summary.paid > 0) {
        refreshAppData()
        toast.success(`Đã đánh dấu ${data.summary.paid} hóa đơn là đã thanh toán`)
      } else if (data.dry_run) {
        toast.info(`Xem trước: ${data.summary.matched} hóa đơn khớp`)
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

  const forgetPassword = async () => {
    setForgetting(true)
    try {
      await api.delete('/settings/password')
      setHasSavedPassword(false)
      toast.success('Đã xóa mật khẩu đã lưu')
    } catch (error) {
      toast.error(api.errorMessage(error, 'Không xóa được mật khẩu đã lưu'))
    } finally {
      setForgetting(false)
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
                <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                  <span>Đã lưu mật khẩu giải nén.</span>
                  <Button type="button" variant="ghost" size="sm" disabled={forgetting} onClick={forgetPassword}>
                    <Trash2 data-icon="inline-start" />
                    Xóa mật khẩu đã lưu
                  </Button>
                </div>
              ) : null}
            </Field>

            <Field orientation="horizontal">
              <Checkbox id="statement-remember" checked={remember} onCheckedChange={(checked) => setRemember(checked === true)} />
              <FieldLabel htmlFor="statement-remember" className="font-normal">Ghi nhớ mật khẩu (lưu mã hóa trên máy chủ)</FieldLabel>
            </Field>

            <Field orientation="horizontal">
              <Checkbox id="statement-dry-run" checked={dryRun} onCheckedChange={(checked) => setDryRun(checked === true)} />
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
