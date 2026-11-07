import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from '@/components/ui/sheet'
import {
  Bank, ChevronLeft, ChevronRight, Copy, FileExport, Inbox, LogOut, Plus, RefreshCw, Tag, Target, Trash2, Phone2, Mail, AlertTriangle, Key, Send, ExternalLink,
} from '@/components/icons'
import { cn } from '@/lib/utils'
import { useAccounts, useAllBudgets, useCategories, useCategoryMap, useRawEvents, useRecurrences, useRules, useSyncStatus, useThuChi } from '../hooks'
import { dayKey, FREQS, freqInfo, monthLabel, money, parseAmount, RECURRENCE_BACKFILL_MONTHS, recurrenceBackfill, recurrenceLabel, relativeTime } from '../format'
import { budgetKey, categoryLabel, categoryTree, effectiveBudgets, flatTree } from '../stats'
import { localTx } from '../db'
import { uuidv7 } from '../sync'
import type { Account, AccountKind, Category, Direction, Freq, IngestInfo, Recurrence, Rule, TelegramCode, TelegramInfo, Transaction } from '../types'
import { AccountIcon, CategoryIcon, Empty, Section } from './common'
import { COLORS, ICON_KEYS, ICONS } from './icons'
import { CategoryGrid } from './TxSheets'

type Sub = 'accounts' | 'categories' | 'rules' | 'budgets' | 'recurring' | 'connect' | 'telegram' | 'unparsed' | null

const items: { key: Exclude<Sub, null>; label: string; icon: typeof Bank; desc: string }[] = [
  { key: 'accounts', label: 'Tài khoản', icon: Bank, desc: 'Ngân hàng, ví, tiền mặt' },
  { key: 'telegram', label: 'Telegram', icon: Send, desc: 'Nhắn "cafe 15k" là ghi xong' },
  { key: 'connect', label: 'Kết nối ngân hàng', icon: Key, desc: 'SePay, email, thiết bị forward' },
  { key: 'unparsed', label: 'Thông báo không đọc được', icon: AlertTriangle, desc: 'Xem, đọc lại hoặc bỏ qua' },
  { key: 'categories', label: 'Danh mục', icon: Tag, desc: 'Tên, màu, biểu tượng' },
  { key: 'rules', label: 'Luật phân loại', icon: Inbox, desc: 'Tự gán danh mục theo nội dung' },
  { key: 'budgets', label: 'Hạn mức', icon: Target, desc: 'Giới hạn chi theo tháng' },
  { key: 'recurring', label: 'Định kỳ', icon: RefreshCw, desc: 'Tự ghi giao dịch lặp lại: tiền nhà, lương…' },
]

export function Settings({ sub, setSub, onLogout, userEmail }: { sub: Sub; setSub: (s: Sub) => void; onLogout?: () => void; userEmail?: string }) {
  const status = useSyncStatus()
  const { engine, db } = useThuChi()
  const unparsed = useRawEvents(['unparsed', 'error'])

  if (sub) {
    const title = items.find((i) => i.key === sub)?.label
    return (
      <div className="space-y-3">
        <button type="button" onClick={() => setSub(null)} className="flex items-center gap-1 text-sm text-muted-foreground">
          <ChevronLeft className="size-4" /> Cài đặt
        </button>
        <h2 className="text-lg font-semibold">{title}</h2>
        {sub === 'accounts' && <AccountsPanel />}
        {sub === 'categories' && <CategoriesPanel />}
        {sub === 'rules' && <RulesPanel />}
        {sub === 'budgets' && <BudgetsPanel />}
        {sub === 'recurring' && <RecurrencesPanel />}
        {sub === 'connect' && <ConnectPanel />}
        {sub === 'telegram' && <TelegramPanel />}
        {sub === 'unparsed' && <UnparsedPanel />}
      </div>
    )
  }

  async function exportCSV() {
    const [txs, cats, accs] = await Promise.all([db.transactions.orderBy('occurred_at').toArray(), db.categories.toArray(), db.accounts.toArray()])
    const cm = new Map(cats.map((c) => [c.id, c.name]))
    const am = new Map(accs.map((a) => [a.id, a.name]))
    const esc = (v: unknown) => `"${String(v ?? '').replace(/"/g, '""')}"`
    const lines = [['Thời gian', 'Tài khoản', 'Chiều', 'Số tiền', 'Danh mục', 'Loại', 'Nội dung', 'Ghi chú'].join(',')]
    for (const t of txs as Transaction[]) {
      lines.push([new Date(t.occurred_at).toLocaleString('vi-VN'), am.get(t.account_id ?? ''), t.direction === 'out' ? 'Chi' : 'Thu', t.amount, cm.get(t.category_id ?? ''), t.kind, t.merchant || t.raw_content, t.note].map(esc).join(','))
    }
    const blob = new Blob(['﻿' + lines.join('\n')], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `thu-chi-${new Date().toISOString().slice(0, 10)}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="space-y-3">
      <Section>
        <div className="divide-y">
          {items.map((it) => (
            <button key={it.key} type="button" onClick={() => setSub(it.key)} className="flex w-full items-center gap-3 px-3 py-3 text-left hover:bg-muted/60">
              <span className="inline-flex size-9 items-center justify-center rounded-full bg-muted">
                <it.icon className="size-4.5" />
              </span>
              <span className="flex-1">
                <span className="block font-medium">{it.label}</span>
                <span className="block text-xs text-muted-foreground">{it.desc}</span>
              </span>
              {it.key === 'unparsed' && unparsed.length > 0 && <span className="rounded-full bg-amber-500 px-2 text-xs text-white">{unparsed.length}</span>}
              <ChevronRight className="size-4 text-muted-foreground" />
            </button>
          ))}
        </div>
      </Section>
      <Section title="Đồng bộ">
        <div className="flex items-center justify-between gap-2 px-3 pb-3 text-sm">
          <span className="text-muted-foreground">
            {status.state === 'offline' ? 'Ngoại tuyến' : status.state === 'error' ? `Lỗi: ${status.error}` : `Lần cuối ${relativeTime(status.lastSyncAt)}`}
            {status.pending > 0 && ` · ${status.pending} thay đổi chờ gửi`}
          </span>
          <Button size="sm" variant="outline" onClick={() => engine.sync()}>
            <RefreshCw /> Đồng bộ
          </Button>
        </div>
        {status.rejected.length > 0 && (
          <div className="mx-3 mb-3 rounded-lg bg-destructive/10 p-2 text-xs text-destructive">
            {status.rejected.length} thay đổi bị máy chủ từ chối (gần nhất: {status.rejected[0].error})
          </div>
        )}
      </Section>
      <Section>
        <div className="divide-y">
          <button type="button" onClick={exportCSV} className="flex w-full items-center gap-3 px-3 py-3 text-left hover:bg-muted/60">
            <FileExport className="size-5" /> <span className="flex-1">Xuất dữ liệu (CSV)</span>
          </button>
          {onLogout && (
            <button type="button" onClick={onLogout} className="flex w-full items-center gap-3 px-3 py-3 text-left text-destructive hover:bg-muted/60">
              <LogOut className="size-5" />
              <span className="flex-1">
                Đăng xuất
                <span className="block text-xs text-muted-foreground">{userEmail} · xoá dữ liệu trên máy này</span>
              </span>
            </button>
          )}
        </div>
      </Section>
    </div>
  )
}

const KIND_LABEL: Record<AccountKind, string> = { bank: 'Ngân hàng', ewallet: 'Ví điện tử', credit: 'Thẻ tín dụng', cash: 'Tiền mặt' }

function AccountsPanel() {
  const accounts = useAccounts()
  const [editing, setEditing] = useState<Account | 'new' | null>(null)
  return (
    <>
      <Section>
        <div className="divide-y">
          {accounts.map((a) => (
            <button key={a.id} type="button" onClick={() => setEditing(a)} className="flex w-full items-center gap-3 px-3 py-2.5 text-left hover:bg-muted/60">
              <AccountIcon account={a} />
              <span className="min-w-0 flex-1">
                <span className="block truncate font-medium">{a.name}</span>
                <span className="block text-xs text-muted-foreground">
                  {KIND_LABEL[a.kind]}
                  {a.account_no_last4 && ` · ···${a.account_no_last4}`} · {a.connector === 'manual' ? 'nhập tay' : a.connector}
                </span>
              </span>
              <span className="tabular-nums">{money(a.balance)}</span>
            </button>
          ))}
        </div>
      </Section>
      <Button className="w-full" variant="outline" onClick={() => setEditing('new')}>
        <Plus /> Thêm tài khoản
      </Button>
      <p className="text-xs text-muted-foreground">
        Khai báo số tài khoản đầy đủ của các tài khoản của bạn (chỉ lưu dạng mã hoá một chiều) để app nhận ra chuyển khoản nội bộ và không tính vào thu/chi.
      </p>
      <AccountSheet key={editing === 'new' ? 'new' : (editing?.id ?? 'none')} value={editing} onClose={() => setEditing(null)} />
    </>
  )
}

function AccountSheet({ value, onClose }: { value: Account | 'new' | null; onClose: () => void }) {
  const { engine, db } = useThuChi()
  const isNew = value === 'new'
  const acc = value && value !== 'new' ? value : null
  const [name, setName] = useState(acc?.name ?? '')
  const [kind, setKind] = useState<AccountKind>(acc?.kind ?? 'bank')
  const [bank, setBank] = useState(acc?.bank_code ?? '')
  const [accountNo, setAccountNo] = useState('')
  const [holder, setHolder] = useState(acc?.holder_name ?? '')
  const [balance, setBalance] = useState(acc ? String(acc.balance) : '')
  const [include, setInclude] = useState(acc?.include_in_stats ?? true)

  const manual = isNew || acc?.connector === 'manual'

  async function adjustBalance(accountId: string, target: number) {
    const id = uuidv7()
    await engine.action(
      'adjust_balance',
      { id, account_id: accountId, balance: target },
      async () => {
        const current = await db.accounts.get(accountId)
        if (!current) return
        const diff = target - current.balance
        if (!diff) return
        await db.transactions.put(localTx({ id, account_id: accountId, direction: diff > 0 ? 'in' : 'out', amount: Math.abs(diff), occurred_at: new Date().toISOString(), kind: 'adjustment' }))
        await db.accounts.update(accountId, { balance: target })
        return { [accountId]: diff }
      },
      [{ entity: 'transactions', id }],
    )
  }

  async function save() {
    if (!name.trim()) return
    const changes: Record<string, unknown> = { name: name.trim(), kind, bank_code: bank.trim().toUpperCase(), holder_name: holder.trim(), include_in_stats: include }
    const rawBalance = balance.trim()
    const negative = /^[-−]/.test(rawBalance)
    const b = manual && rawBalance ? (negative ? -parseAmount(rawBalance.slice(1)) : parseAmount(rawBalance)) : NaN
    if (accountNo.trim()) changes.account_no = accountNo.trim()
    if (isNew) {
      if (Number.isFinite(b)) changes.balance = b
      const local = { ...changes }
      delete local.account_no
      await engine.create('accounts', { ...changes }, { ...local, connector: 'manual', balance: changes.balance ?? 0, balance_at: null, account_no_last4: accountNo.replace(/\D/g, '').slice(-4), sort: 50 })
    } else if (acc) {
      await engine.patch('accounts', acc.id, changes)
      if (manual && Number.isFinite(b) && b !== acc.balance) await adjustBalance(acc.id, b)
    }
    toast.success('Đã lưu tài khoản')
    onClose()
  }

  return (
    <Sheet open={!!value} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="bottom" className="max-h-[92dvh] overflow-y-auto rounded-t-2xl pb-[max(1rem,env(safe-area-inset-bottom))]">
        <SheetHeader>
          <SheetTitle>{isNew ? 'Thêm tài khoản' : 'Sửa tài khoản'}</SheetTitle>
        </SheetHeader>
        <div className="space-y-3 px-4">
          <div className="space-y-1.5">
            <Label htmlFor="acc-name">Tên</Label>
            <Input id="acc-name" maxLength={100} value={name} onChange={(e) => setName(e.target.value)} placeholder="vd: Timo, Vietcombank" />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Loại</Label>
              <Select value={kind} onValueChange={(v) => setKind(v as AccountKind)} items={Object.entries(KIND_LABEL).map(([value, label]) => ({ value, label }))}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(KIND_LABEL).map(([k, l]) => (
                    <SelectItem key={k} value={k}>
                      {l}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="acc-bank">Mã ngân hàng</Label>
              <Input id="acc-bank" value={bank} onChange={(e) => setBank(e.target.value)} placeholder="VCB, TIMO, MOMO…" />
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="acc-no">Số tài khoản {acc?.account_no_last4 && <span className="text-muted-foreground">(đang lưu ···{acc.account_no_last4})</span>}</Label>
            <Input id="acc-no" inputMode="numeric" value={accountNo} onChange={(e) => setAccountNo(e.target.value)} placeholder="Để trống nếu không đổi" />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="acc-holder">Tên chủ tài khoản</Label>
            <Input id="acc-holder" value={holder} onChange={(e) => setHolder(e.target.value)} placeholder="NGUYEN VAN A" />
          </div>
          {manual && (
            <div className="space-y-1.5">
              <Label htmlFor="acc-bal">{isNew ? 'Số dư ban đầu' : 'Số dư hiện tại'}</Label>
              <Input id="acc-bal" inputMode="decimal" value={balance} onChange={(e) => setBalance(e.target.value)} placeholder="vd: 2tr" />
              {!isNew && <p className="text-xs text-muted-foreground">Đổi số dư sẽ tạo giao dịch “Điều chỉnh số dư” cho phần chênh lệch</p>}
            </div>
          )}
          <label className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-sm">
            Tính vào thống kê
            <Switch checked={include} onCheckedChange={setInclude} />
          </label>
          <div className="flex gap-2">
            <Button className="flex-1" onClick={save}>
              Lưu
            </Button>
            {acc && acc.connector === 'manual' && (
              <Button
                variant="destructive"
                onClick={async () => {
                  await engine.remove('accounts', acc.id)
                  onClose()
                }}
              >
                <Trash2 />
              </Button>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}

const NO_PARENT = '__root__'

function CategoriesPanel() {
  const categories = useCategories()
  const { engine, preset } = useThuChi()
  const [editing, setEditing] = useState<Category | 'expense' | 'income' | null>(null)
  async function addDefaults() {
    await engine.action('add_default_categories', {})
    toast.success('Đã gửi yêu cầu, danh mục mẫu hiện ra sau khi đồng bộ')
  }
  return (
    <>
      <Section>
        <div className="flex items-center gap-3 p-3">
          <p className="min-w-0 flex-1 text-xs text-muted-foreground">
            {preset === 'landlord'
              ? 'Bộ danh mục cho nhà trọ: sửa chữa điện nước, trang bị phòng (tủ lạnh, kệ bếp…), vệ sinh, môi giới, quảng cáo, thuế phí, tiền phòng, tiền cọc… Thiếu cái nào thì bấm thêm, danh mục bạn đã có giữ nguyên.'
              : 'Thêm lại các danh mục mặc định còn thiếu, danh mục bạn đã có giữ nguyên.'}
          </p>
          <Button size="sm" variant="outline" className="shrink-0" onClick={addDefaults}>
            <Plus /> Thêm danh mục mẫu
          </Button>
        </div>
      </Section>
      {(['expense', 'income'] as const).map((kind) => (
        <Section key={kind} title={kind === 'expense' ? 'Chi' : 'Thu'} action={<Button size="xs" variant="ghost" onClick={() => setEditing(kind)}><Plus /> Thêm</Button>}>
          <div className="divide-y">
            {flatTree(categories.filter((c) => c.kind === kind)).map(({ category: c, child }) => (
              <button key={c.id} type="button" onClick={() => setEditing(c)} className={cn('flex w-full items-center gap-3 py-2 pr-3 text-left hover:bg-muted/60', child ? 'pl-10' : 'pl-3')}>
                <CategoryIcon category={c} />
                <span className="min-w-0 flex-1 truncate">{c.name}</span>
                <ChevronRight className="size-4 text-muted-foreground" />
              </button>
            ))}
          </div>
        </Section>
      ))}
      <CategorySheet key={typeof editing === 'object' && editing ? editing.id : (editing ?? 'none')} value={editing} onClose={() => setEditing(null)} />
    </>
  )
}

function CategorySheet({ value, onClose }: { value: Category | 'expense' | 'income' | null; onClose: () => void }) {
  const { engine, db } = useThuChi()
  const categories = useCategories()
  const cat = value && typeof value === 'object' ? value : null
  const kind = cat ? cat.kind : value
  const [name, setName] = useState(cat?.name ?? '')
  const [icon, setIcon] = useState(cat?.icon ?? 'tag')
  const [color, setColor] = useState(cat?.color ?? COLORS[6])
  const [parentId, setParentId] = useState(cat?.parent_id ?? NO_PARENT)
  const hasChildren = !!cat && categories.some((c) => c.parent_id === cat.id)
  const parents = categoryTree(categories.filter((c) => c.kind === kind)).map((n) => n.category).filter((c) => c.id !== cat?.id)
  const parentItems = [{ value: NO_PARENT, label: 'Không (danh mục gốc)' }, ...parents.map((c) => ({ value: c.id, label: c.name }))]

  async function save() {
    if (!name.trim()) return
    const parent_id = parentId === NO_PARENT ? null : parentId
    if (cat) await engine.patch('categories', cat.id, { name: name.trim(), icon, color, ...(parent_id !== cat.parent_id ? { parent_id } : {}) })
    else await engine.create('categories', { name: name.trim(), icon, color, kind: value as string, sort: 200, parent_id }, { key: '' })
    onClose()
  }

  async function remove(c: Category) {
    await engine.remove('categories', c.id, async () => {
      await db.categories.filter((x) => x.parent_id === c.id).modify({ parent_id: null })
    })
    onClose()
  }

  return (
    <Sheet open={!!value} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="bottom" className="max-h-[92dvh] overflow-y-auto rounded-t-2xl pb-[max(1rem,env(safe-area-inset-bottom))]">
        <SheetHeader>
          <SheetTitle>{cat ? 'Sửa danh mục' : 'Thêm danh mục'}</SheetTitle>
        </SheetHeader>
        <div className="space-y-3 px-4">
          <div className="space-y-1.5">
            <Label htmlFor="cat-name">Tên</Label>
            <Input id="cat-name" maxLength={60} value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>Thuộc danh mục</Label>
            <Select value={parentId} onValueChange={(v) => setParentId(String(v))} items={parentItems} disabled={hasChildren}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {parentItems.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {hasChildren && <p className="text-xs text-muted-foreground">Danh mục đang có danh mục con nên không thể đặt làm con.</p>}
          </div>
          <div className="space-y-1.5">
            <Label>Biểu tượng</Label>
            <div className="grid grid-cols-8 gap-1.5">
              {ICON_KEYS.map((k) => {
                const I = ICONS[k]
                return (
                  <button key={k} type="button" aria-label={k} onClick={() => setIcon(k)} className={cn('flex aspect-square items-center justify-center rounded-md border', icon === k ? 'border-primary bg-primary/10' : 'border-transparent hover:bg-muted')}>
                    <I className="size-5" style={{ color }} />
                  </button>
                )
              })}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label>Màu</Label>
            <div className="flex flex-wrap gap-2">
              {COLORS.map((c) => (
                <button key={c} type="button" aria-label={c} onClick={() => setColor(c)} className={cn('size-8 rounded-full border-2', color === c ? 'border-foreground' : 'border-transparent')} style={{ backgroundColor: c }} />
              ))}
            </div>
          </div>
          <div className="flex gap-2">
            <Button className="flex-1" onClick={save}>
              Lưu
            </Button>
            {cat && (
              <Button variant="destructive" aria-label="Xoá danh mục" onClick={() => remove(cat)}>
                <Trash2 />
              </Button>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}

const MATCH_LABEL: Record<Rule['match'], string> = { contains: 'Nội dung chứa', merchant: 'Đúng tên', regex: 'Biểu thức', own_account: 'TK của mình' }

function RulesPanel() {
  const rules = useRules()
  const categories = useCategories()
  const catMap = useCategoryMap()
  const { engine } = useThuChi()
  const [pattern, setPattern] = useState('')
  const [categoryId, setCategoryId] = useState('')

  async function add() {
    if (!pattern.trim() || !categoryId) return
    await engine.create('rules', { pattern: pattern.trim(), match: 'contains', category_id: categoryId, origin: 'manual' }, { merchant_name: '', hits: 0 })
    setPattern('')
  }

  return (
    <>
      <Section title="Thêm luật">
        <div className="space-y-2 px-3 pb-3">
          <Input maxLength={200} value={pattern} onChange={(e) => setPattern(e.target.value)} placeholder="Nội dung chứa… vd: PHO THIN" />
          <div className="flex gap-2">
            <Select value={categoryId} onValueChange={(v) => setCategoryId(String(v))} items={categories.map((c) => ({ value: c.id, label: c.name }))}>
              <SelectTrigger className="flex-1">
                <SelectValue placeholder="Chọn danh mục" />
              </SelectTrigger>
              <SelectContent>
                {flatTree(categories).map(({ category: c, child }) => (
                  <SelectItem key={c.id} value={c.id}>
                    {categoryLabel(c, child)}
                    {c.kind === 'income' ? ' (thu)' : ''}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button onClick={add} disabled={!pattern.trim() || !categoryId}>
              <Plus /> Thêm
            </Button>
          </div>
        </div>
      </Section>
      <Section>
        <div className="divide-y">
          {rules.map((r) => (
            <div key={r.id} className="flex items-center gap-3 px-3 py-2">
              <span className="min-w-0 flex-1">
                <span className="block truncate font-medium">{r.pattern}</span>
                <span className="block text-xs text-muted-foreground">
                  {MATCH_LABEL[r.match]} → {r.match === 'own_account' ? 'chuyển nội bộ' : (catMap.get(r.category_id ?? '')?.name ?? '?')} · {r.origin === 'learned' ? 'tự học' : 'tự đặt'}
                  {r.hits > 0 && ` · ${r.hits} lần`}
                </span>
              </span>
              <Button variant="ghost" size="icon-sm" aria-label="Xoá luật" onClick={() => engine.remove('rules', r.id)}>
                <Trash2 />
              </Button>
            </div>
          ))}
          {rules.length === 0 && <Empty>Chưa có luật. Khi bạn chọn danh mục cho giao dịch, app sẽ tự học.</Empty>}
        </div>
      </Section>
    </>
  )
}

function BudgetsPanel() {
  const { month, engine } = useThuChi()
  const budgets = useAllBudgets()
  const categories = useCategories().filter((c) => c.kind === 'expense')
  const effective = effectiveBudgets(budgets, month)
  const before = effectiveBudgets(budgets, month, true)
  const rows: { key: string; name: string; category: Category | null; child: boolean }[] = [
    { key: 'total', name: 'Tổng chi', category: null, child: false },
    ...flatTree(categories).map(({ category: c, child }) => ({ key: c.id, name: c.name, category: c, child })),
  ]

  async function setLimit(key: string, text: string) {
    const amount = text.trim() ? parseAmount(text) : 0
    if (!Number.isFinite(amount) || amount < 0) return
    if (amount === (effective.get(key)?.limit_amount ?? 0)) return
    const own = budgets.filter((b) => b.month === month && budgetKey(b) === key)
    if (own.length === 0) {
      await engine.create('budgets', { category_id: key === 'total' ? null : key, month, limit_amount: amount })
      return
    }
    if (amount === (before.get(key)?.limit_amount ?? 0)) {
      for (const b of own) await engine.remove('budgets', b.id)
      return
    }
    const [first, ...rest] = own
    await engine.patch('budgets', first.id, { limit_amount: amount })
    for (const b of rest) await engine.remove('budgets', b.id)
  }

  return (
    <Section title={`Hạn mức ${monthLabel(month).toLowerCase()}`}>
      <p className="px-3 pb-2 text-xs text-muted-foreground">Hạn mức áp dụng từ tháng đang chọn cho các tháng sau, tới khi bạn đổi. Xoá trống để tắt. Hạn mức danh mục cha tính cả danh mục con.</p>
      <div className="divide-y">
        {rows.map((r) => {
          const eff = effective.get(r.key)
          const limit = eff?.limit_amount ?? 0
          const inherited = eff && eff.month < month && limit > 0
          const off = eff && eff.month === month && limit === 0 && (before.get(r.key)?.limit_amount ?? 0) > 0
          return (
            <div key={r.key} className={cn('flex items-center gap-3 py-2 pr-3', r.child ? 'pl-10' : 'pl-3')}>
              {r.category ? <CategoryIcon category={r.category} /> : <span className="size-9 shrink-0" />}
              <span className="min-w-0 flex-1">
                <span className="block truncate">{r.name}</span>
                {inherited && <span className="block text-xs text-muted-foreground">Từ {monthLabel(eff.month).toLowerCase()}</span>}
                {off && <span className="block text-xs text-muted-foreground">Đã tắt từ tháng này</span>}
              </span>
              <Input
                key={`${r.key}-${eff?.id ?? ''}-${limit}`}
                defaultValue={limit > 0 ? String(limit) : ''}
                onBlur={(e) => setLimit(r.key, e.target.value)}
                placeholder="—"
                inputMode="decimal"
                className="w-32 shrink-0 text-right"
              />
            </div>
          )
        })}
      </div>
    </Section>
  )
}

function fromDateInput(value: string, endOfDay = false) {
  const [y, m, d] = value.split('-').map(Number)
  if (!y || !m || !d) return null
  const date = endOfDay ? new Date(y, m - 1, d, 23, 59, 59) : new Date(y, m - 1, d, 8, 0, 0)
  return Number.isNaN(date.getTime()) ? null : date.toISOString()
}

function shortDate(iso: string) {
  return new Date(iso).toLocaleDateString('vi-VN')
}

function recurrenceStatus(r: Recurrence) {
  if (r.active) return `tiếp theo ${shortDate(r.next_at)}`
  if (r.end_at && new Date(r.next_at) > new Date(r.end_at)) return 'đã kết thúc'
  return 'tạm dừng'
}

function RecurrencesPanel() {
  const recurrences = useRecurrences()
  const catMap = useCategoryMap()
  const [editing, setEditing] = useState<Recurrence | 'new' | null>(null)
  return (
    <>
      <Section>
        <div className="divide-y">
          {recurrences.map((r) => {
            const category = r.category_id ? catMap.get(r.category_id) : null
            return (
              <button key={r.id} type="button" onClick={() => setEditing(r)} className={cn('flex min-h-14 w-full items-center gap-3 px-3 py-2.5 text-left hover:bg-muted/60', !r.active && 'opacity-60')}>
                <CategoryIcon category={category} />
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-medium">{r.merchant || category?.name || (r.direction === 'in' ? 'Khoản thu định kỳ' : 'Khoản chi định kỳ')}</span>
                  <span className="block truncate text-xs text-muted-foreground">
                    {recurrenceLabel(r)} · {recurrenceStatus(r)}
                  </span>
                </span>
                <span className={cn('shrink-0 font-semibold tabular-nums', r.direction === 'in' && 'text-emerald-600 dark:text-emerald-400')}>
                  {r.direction === 'in' ? '+' : '−'}
                  {money(r.amount)}
                </span>
              </button>
            )
          })}
          {recurrences.length === 0 && <Empty>Chưa có ghi chép định kỳ. Thêm các khoản lặp lại như tiền nhà, lương, internet để app tự ghi.</Empty>}
        </div>
      </Section>
      <Button className="w-full" variant="outline" onClick={() => setEditing('new')}>
        <Plus /> Thêm định kỳ
      </Button>
      <p className="text-xs text-muted-foreground">Giao dịch được tự ghi vào ngày đến hạn, kể cả khi bạn không mở app. Ngày 29–31 sẽ ghi vào ngày cuối của tháng ngắn hơn. Xoá định kỳ không xoá các giao dịch đã ghi.</p>
      <RecurrenceSheet key={editing === 'new' ? 'new' : (editing?.id ?? 'none')} value={editing} onClose={() => setEditing(null)} />
    </>
  )
}

function RecurrenceSheet({ value, onClose }: { value: Recurrence | 'new' | null; onClose: () => void }) {
  const { engine } = useThuChi()
  const accounts = useAccounts()
  const categories = useCategories()
  const rec = value && value !== 'new' ? value : null
  const [direction, setDirection] = useState<Direction>(rec?.direction ?? 'out')
  const [amountText, setAmountText] = useState(rec ? String(rec.amount) : '')
  const [merchant, setMerchant] = useState(rec?.merchant ?? '')
  const [categoryId, setCategoryId] = useState<string | null>(rec?.category_id ?? null)
  const [pickedAccount, setPickedAccount] = useState(rec?.account_id ?? '')
  const [freq, setFreq] = useState<Freq>(rec?.freq ?? 'monthly')
  const [intervalText, setIntervalText] = useState(String(rec?.interval ?? 1))
  const initStart = dayKey(rec ? new Date(rec.start_at) : new Date())
  const initEnd = rec?.end_at ? dayKey(new Date(rec.end_at)) : ''
  const [start, setStart] = useState(initStart)
  const [end, setEnd] = useState(initEnd)
  const [active, setActive] = useState(rec?.active ?? true)

  const accountId = pickedAccount || (accounts.find((a) => a.kind === 'cash') ?? accounts[0])?.id || ''
  const kindCats = categories.filter((c) => c.kind === (direction === 'out' ? 'expense' : 'income'))
  const amount = parseAmount(amountText)
  const interval = Number(intervalText)
  const startISO = fromDateInput(start)
  const endISO = end ? fromDateInput(end, true) : null
  const unit = freqInfo(freq).unit
  const valid = Number.isFinite(amount) && amount > 0 && !!accountId && Number.isInteger(interval) && interval >= 1 && interval <= 1000 && !!startISO && (!end || (!!endISO && endISO >= startISO))
  const expired = !!rec && !rec.active && !!rec.end_at && rec.next_at > rec.end_at
  const backfill = !rec && valid && startISO && start < dayKey(new Date()) ? recurrenceBackfill({ freq, interval, start_at: startISO, end_at: endISO }) : null

  async function save() {
    if (!valid || !startISO) return
    const fields: Record<string, unknown> = { direction, amount, merchant: merchant.trim(), category_id: categoryId, account_id: accountId, freq, interval, active }
    if (rec) {
      const changes: Record<string, unknown> = {}
      for (const [k, v] of Object.entries(fields)) if (v !== (rec as unknown as Record<string, unknown>)[k]) changes[k] = v
      if (start !== initStart) changes.start_at = startISO
      if (end !== initEnd) changes.end_at = endISO
      if (expired && active === rec.active && end !== initEnd && (!endISO || endISO >= rec.next_at)) changes.active = true
      if (Object.keys(changes).length) await engine.patch('recurrences', rec.id, changes)
    } else {
      await engine.create('recurrences', { ...fields, note: '', start_at: startISO, end_at: endISO }, { next_at: startISO })
    }
    toast.success('Đã lưu định kỳ')
    onClose()
  }

  async function remove(r: Recurrence) {
    await engine.remove('recurrences', r.id)
    toast.success('Đã xoá định kỳ, các giao dịch đã ghi vẫn giữ nguyên')
    onClose()
  }

  return (
    <Sheet open={!!value} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="bottom" className="max-h-[92dvh] overflow-y-auto rounded-t-2xl pb-[max(1rem,env(safe-area-inset-bottom))]">
        <SheetHeader className="pb-0">
          <SheetTitle>{rec ? 'Sửa định kỳ' : 'Thêm định kỳ'}</SheetTitle>
          <SheetDescription>{rec ? `${recurrenceLabel(rec)} · ${recurrenceStatus(rec)}` : 'Tự ghi giao dịch theo lịch lặp lại'}</SheetDescription>
        </SheetHeader>
        <div className="space-y-4 px-4">
          <div className="grid grid-cols-2 gap-1 rounded-lg bg-muted p-1">
            {(['out', 'in'] as const).map((d) => (
              <button
                key={d}
                type="button"
                onClick={() => {
                  setDirection(d)
                  setCategoryId(null)
                }}
                className={cn('rounded-md py-2 text-sm font-medium', direction === d ? 'bg-background shadow-sm' : 'text-muted-foreground')}
              >
                {d === 'out' ? 'Chi' : 'Thu'}
              </button>
            ))}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="rec-amount">Số tiền</Label>
            <Input id="rec-amount" inputMode="decimal" placeholder="vd: 3tr, 250k" value={amountText} onChange={(e) => setAmountText(e.target.value)} className="text-lg font-semibold" />
            {Number.isFinite(amount) && amount > 0 && <p className="text-xs text-muted-foreground">{money(amount)}</p>}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="rec-merchant">Nội dung</Label>
            <Input id="rec-merchant" maxLength={200} placeholder="vd: Tiền nhà, Internet, Lương" value={merchant} onChange={(e) => setMerchant(e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>Danh mục</Label>
            <CategoryGrid categories={kindCats} value={categoryId} onChange={(id) => setCategoryId(id === categoryId ? null : id)} />
          </div>
          <div className="space-y-1.5">
            <Label>Tài khoản</Label>
            <Select value={accountId} onValueChange={(v) => setPickedAccount(String(v))} items={accounts.map((a) => ({ value: a.id, label: a.name }))}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {accounts.map((a) => (
                  <SelectItem key={a.id} value={a.id}>
                    {a.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Lặp lại</Label>
              <Select value={freq} onValueChange={(v) => setFreq(v as Freq)} items={FREQS}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {FREQS.map((f) => (
                    <SelectItem key={f.value} value={f.value}>
                      {f.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="rec-interval">Mỗi bao nhiêu {unit}</Label>
              <Input id="rec-interval" type="number" inputMode="numeric" min={1} max={1000} value={intervalText} onChange={(e) => setIntervalText(e.target.value)} aria-invalid={!(Number.isInteger(interval) && interval >= 1) || undefined} />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="rec-start">Bắt đầu</Label>
              <Input id="rec-start" type="date" value={start} onChange={(e) => setStart(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="rec-end">Kết thúc</Label>
              <Input id="rec-end" type="date" value={end} min={start || undefined} onChange={(e) => setEnd(e.target.value)} aria-invalid={(!!end && !!endISO && !!startISO && endISO < startISO) || undefined} />
            </div>
          </div>
          {backfill && backfill.count > 0 && (
            <p className="rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs">
              Ngày bắt đầu đã qua: sẽ ghi ngay {backfill.count} giao dịch đã đến hạn
              {backfill.truncated ? ` (chỉ ghi bù các kỳ trong ${RECURRENCE_BACKFILL_MONTHS} tháng gần nhất)` : ''}.
            </p>
          )}
          {expired && <p className="text-xs text-muted-foreground">Định kỳ đã kết thúc. Gia hạn ngày kết thúc (hoặc để trống) để tự bật lại.</p>}
          {rec && (
            <label className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-sm">
              <span>
                <span className="block font-medium">Đang bật</span>
                <span className="block text-xs text-muted-foreground">Tắt để tạm dừng, bật lại sẽ tiếp tục từ kỳ kế tiếp (không ghi bù)</span>
              </span>
              <Switch checked={active} onCheckedChange={setActive} />
            </label>
          )}
          <div className="flex gap-2">
            <Button className="flex-1" onClick={save} disabled={!valid}>
              Lưu
            </Button>
            {rec && (
              <Button variant="destructive" aria-label="Xoá định kỳ" onClick={() => remove(rec)}>
                <Trash2 />
              </Button>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}

function CopyField({ label, value }: { label: string; value: string }) {
  return (
    <div className="space-y-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <div className="flex gap-1">
        <Input readOnly value={value} className="font-mono text-xs" onFocus={(e) => e.target.select()} />
        <Button
          variant="outline"
          size="icon"
          aria-label="Sao chép"
          onClick={async () => {
            try {
              await navigator.clipboard.writeText(value)
              toast.success('Đã sao chép')
            } catch {
              toast.error('Không sao chép được')
            }
          }}
        >
          <Copy />
        </Button>
      </div>
    </div>
  )
}

function ConnectPanel() {
  const { transport } = useThuChi()
  const [info, setInfo] = useState<IngestInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [secret, setSecret] = useState('')
  const [method, setMethod] = useState<'hmac' | 'apikey'>('hmac')
  const [newToken, setNewToken] = useState<{ secret: string; url: string; kind: string } | null>(null)
  const [deviceName, setDeviceName] = useState('')

  const [reload, setReload] = useState(0)
  const load = () => setReload((n) => n + 1)

  useEffect(() => {
    let cancelled = false
    transport<{ data: IngestInfo }>('GET', '/plugins/thu-chi/ingest/info')
      .then((res) => {
        if (cancelled) return
        setInfo(res.data)
        setError(null)
      })
      .catch(() => !cancelled && setError('Cần có mạng để xem và đổi cấu hình kết nối.'))
    return () => {
      cancelled = true
    }
  }, [transport, reload])

  async function saveSePay() {
    try {
      await transport('PUT', '/plugins/thu-chi/ingest/sepay', { auth_method: method, secret })
      setSecret('')
      toast.success('Đã lưu cấu hình SePay')
      load()
    } catch {
      toast.error('Lưu thất bại')
    }
  }

  async function createToken(kind: 'notify' | 'email') {
    const name = deviceName.trim() || (kind === 'notify' ? 'Android' : 'Email')
    try {
      const res = await transport<{ data: { secret: string; url: string } }>('POST', '/plugins/thu-chi/ingest/tokens', { kind, name })
      setNewToken({ ...res.data, kind })
      setDeviceName('')
      load()
    } catch {
      toast.error('Tạo token thất bại')
    }
  }

  async function revoke(id: string) {
    try {
      await transport('DELETE', `/plugins/thu-chi/ingest/tokens/${id}`)
      load()
    } catch {
      toast.error('Thu hồi thất bại')
    }
  }

  if (error) return <Empty>{error}</Empty>
  if (!info) return <Empty>Đang tải…</Empty>

  return (
    <div className="space-y-3">
      <Section title="SePay (TPBank, MB, ACB…)">
        <div className="space-y-3 px-3 pb-3 text-sm">
          <p className="text-muted-foreground">
            Trong my.sepay.vn → Tích hợp Webhook: dán URL dưới, chọn loại giao dịch <b>Tất cả</b>, bảo mật <b>HMAC-SHA256</b>, rồi dán Secret Key vào đây.
          </p>
          <CopyField label="Webhook URL" value={info.urls.sepay} />
          <p className="text-xs">{info.sepay ? `Đã cấu hình (${info.sepay.config?.auth_method}) · nhận lần cuối ${relativeTime(info.sepay.last_used_at)}` : 'Chưa cấu hình'}</p>
          <div className="flex gap-2">
            <Select value={method} onValueChange={(v) => setMethod(v as 'hmac' | 'apikey')} items={[{ value: 'hmac', label: 'HMAC' }, { value: 'apikey', label: 'API Key' }]}>
              <SelectTrigger className="w-28">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="hmac">HMAC</SelectItem>
                <SelectItem value="apikey">API Key</SelectItem>
              </SelectContent>
            </Select>
            <Input type="password" value={secret} onChange={(e) => setSecret(e.target.value)} placeholder="Secret Key từ SePay" />
          </div>
          <Button size="sm" onClick={saveSePay} disabled={secret.trim().length < 8}>
            Lưu SePay
          </Button>
        </div>
      </Section>

      <Section title="Thiết bị forward & email">
        <div className="space-y-3 px-3 pb-3 text-sm">
          <p className="text-muted-foreground">
            Android (SmsForwarder/MacroDroid) gửi thông báo app ngân hàng; email Timo đi qua Cloudflare Email Worker. Mỗi nguồn dùng token riêng, thu hồi được.
          </p>
          <Input value={deviceName} onChange={(e) => setDeviceName(e.target.value)} placeholder="Tên, vd: Android Minh" />
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="outline" onClick={() => createToken('notify')}>
              <Phone2 /> Token thông báo Android
            </Button>
            <Button size="sm" variant="outline" onClick={() => createToken('email')}>
              <Mail /> Token email
            </Button>
          </div>
          <div className="divide-y rounded-lg border">
            {info.tokens.map((t) => (
              <div key={t.id} className="flex items-center gap-2 px-3 py-2">
                {t.kind === 'notify' ? <Phone2 className="size-4" /> : <Mail className="size-4" />}
                <span className="min-w-0 flex-1">
                  <span className="block truncate">{t.name}</span>
                  <span className="block text-xs text-muted-foreground">nhận lần cuối {relativeTime(t.last_used_at)}</span>
                </span>
                <Button size="xs" variant="ghost" onClick={() => revoke(t.id)}>
                  Thu hồi
                </Button>
              </div>
            ))}
            {info.tokens.length === 0 && <p className="px-3 py-2 text-xs text-muted-foreground">Chưa có thiết bị</p>}
          </div>
        </div>
      </Section>

      <Sheet open={!!newToken} onOpenChange={(o) => !o && setNewToken(null)}>
        <SheetContent side="bottom" className="rounded-t-2xl pb-[max(1rem,env(safe-area-inset-bottom))]">
          <SheetHeader>
            <SheetTitle>Token mới</SheetTitle>
            <SheetDescription>Chỉ hiện một lần. Lưu lại ngay.</SheetDescription>
          </SheetHeader>
          {newToken && (
            <div className="space-y-3 px-4 text-sm">
              <CopyField label="URL" value={newToken.url} />
              <CopyField label="Header Authorization" value={`Bearer ${newToken.secret}`} />
              {newToken.kind === 'notify' ? (
                <p className="text-xs text-muted-foreground">
                  Body JSON: {'{"package":"…","title":"…","text":"…","posted_at":"…","lat":…,"lng":…}'}. SmsForwarder: Webhook, method POST, thêm header Authorization, template dùng [from]/[title]/[content]/[receive_time].
                </p>
              ) : (
                <p className="text-xs text-muted-foreground">Đặt token vào biến môi trường THUCHI_TOKEN của Email Worker (xem deploy/cloudflare-email-worker).</p>
              )}
            </div>
          )}
        </SheetContent>
      </Sheet>
    </div>
  )
}

const TELEGRAM_EXAMPLES = [
  { text: 'cafe 15k', desc: 'chi 15.000đ, tự vào Ăn uống' },
  { text: 'ăn trưa 45k, gửi xe 5k', desc: 'nhiều khoản trong một tin' },
  { text: '+2tr lương', desc: 'dấu + là khoản thu' },
  { text: 'hôm qua grab 32k', desc: 'ghi cho ngày hôm qua; hoặc 20/10 tiền điện 450k' },
]

function Step({ n, title, children }: { n: number; title: React.ReactNode; children?: React.ReactNode }) {
  return (
    <li className="flex gap-3">
      <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-semibold text-primary-foreground">{n}</span>
      <div className="min-w-0 flex-1 space-y-2 pt-0.5">
        <p className="leading-5 font-medium">{title}</p>
        {children && <div className="space-y-2 text-xs text-muted-foreground">{children}</div>}
      </div>
    </li>
  )
}

function TelegramPanel() {
  const { transport } = useThuChi()
  const accounts = useAccounts()
  const [info, setInfo] = useState<TelegramInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [code, setCode] = useState<TelegramCode | null>(null)
  const [reload, setReload] = useState(0)
  const load = () => setReload((n) => n + 1)

  useEffect(() => {
    let cancelled = false
    transport<{ data: TelegramInfo }>('GET', '/plugins/thu-chi/telegram/info')
      .then((res) => {
        if (cancelled) return
        setInfo(res.data)
        setError(null)
      })
      .catch(() => !cancelled && setError('Cần có mạng để kết nối Telegram.'))
    return () => {
      cancelled = true
    }
  }, [transport, reload])

  const linkedCount = info?.links.length ?? 0
  useEffect(() => {
    if (!code) return
    const timer = setInterval(() => {
      transport<{ data: TelegramInfo }>('GET', '/plugins/thu-chi/telegram/info')
        .then((res) => {
          setInfo(res.data)
          if (res.data.links.length > linkedCount) {
            setCode(null)
            toast.success('Đã kết nối Telegram')
          } else if (new Date(code.expires_at).getTime() < Date.now()) {
            setCode(null)
          }
        })
        .catch(() => {})
    }, 3000)
    return () => clearInterval(timer)
  }, [code, linkedCount, transport])

  async function createCode() {
    try {
      const res = await transport<{ data: TelegramCode }>('POST', '/plugins/thu-chi/telegram/link')
      setCode(res.data)
    } catch {
      toast.error('Chưa tạo được mã kết nối')
    }
  }

  async function setAccount(id: string, accountId: string) {
    try {
      await transport('PUT', `/plugins/thu-chi/telegram/links/${id}`, { default_account_id: accountId === 'auto' ? null : accountId })
      load()
    } catch {
      toast.error('Lưu thất bại')
    }
  }

  async function unlink(id: string) {
    try {
      await transport('DELETE', `/plugins/thu-chi/telegram/links/${id}`)
      load()
    } catch {
      toast.error('Ngắt kết nối thất bại')
    }
  }

  if (error) return <Empty>{error}</Empty>
  if (!info) return <Empty>Đang tải…</Empty>
  if (!info.enabled) return <Empty>Máy chủ chưa bật bot Telegram.</Empty>

  const accountItems = [{ value: 'auto', label: 'Tiền mặt (mặc định)' }, ...accounts.map((a) => ({ value: a.id, label: a.name }))]
  const bot = `@${info.bot_username}`
  const examples = info.examples?.length ? info.examples : TELEGRAM_EXAMPLES
  const linked = info.links.length > 0
  const expiresAt = code ? new Date(code.expires_at).toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' }) : ''

  return (
    <div className="space-y-3">
      {linked && (
        <Section title="Tài khoản Telegram đã kết nối">
          <div className="space-y-3 px-3 pb-3 text-sm">
            <div className="divide-y rounded-lg border">
              {info.links.map((l) => (
                <div key={l.id} className="space-y-2 px-3 py-2">
                  <div className="flex items-center gap-2">
                    <Send className="size-4 text-sky-500" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium">{l.telegram_name || l.telegram_username || l.telegram_user_id}</span>
                      <span className="block truncate text-xs text-muted-foreground">
                        {l.telegram_username ? `@${l.telegram_username} · ` : ''}kết nối {relativeTime(l.linked_at)}
                      </span>
                    </span>
                    <Button size="xs" variant="ghost" onClick={() => unlink(l.id)}>
                      Ngắt
                    </Button>
                  </div>
                  <div className="flex items-center gap-2">
                    <Label className="shrink-0 text-xs text-muted-foreground">Ghi vào</Label>
                    <Select value={l.default_account_id ?? 'auto'} onValueChange={(v) => v && setAccount(l.id, v)} items={accountItems}>
                      <SelectTrigger size="sm" className="min-w-0 flex-1">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {accountItems.map((a) => (
                          <SelectItem key={a.value} value={a.value}>
                            {a.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              ))}
            </div>
            <p className="text-xs text-muted-foreground">
              Nhắn cho {bot} (chat riêng hoặc trong nhóm có bot), ví dụ <code className="rounded bg-muted px-1">{examples[0].text}</code>, là khoản đó được ghi vào sổ.
            </p>
            <a href={`https://t.me/${info.bot_username}`} target="_blank" rel="noreferrer" className={cn(buttonVariants({ variant: 'outline' }), 'w-full')}>
              <ExternalLink /> Mở chat với {bot}
            </a>
          </div>
        </Section>
      )}

      <Section title={linked ? 'Kết nối thêm tài khoản Telegram' : 'Kết nối Telegram'}>
        <div className="space-y-4 px-3 pb-3 text-sm">
          <p className="text-xs text-muted-foreground">
            {linked
              ? 'Dùng khi bạn có thêm điện thoại/tài khoản Telegram khác, hoặc muốn người cùng quản lý ghi chung vào sổ này: tạo mã rồi gửi link kết nối cho người đó.'
              : `Sau khi kết nối, bạn chỉ cần nhắn cho bot ${bot} kiểu “${examples[0].text}” là khoản đó được ghi vào sổ. Mỗi tài khoản Telegram chỉ cần kết nối một lần.`}
          </p>
          <ol className="space-y-4">
            <Step n={1} title="Cài Telegram và đăng nhập">
              <p>
                Trên điện thoại hoặc máy tính. Chưa có thì tải ở{' '}
                <a href="https://telegram.org/apps" target="_blank" rel="noreferrer" className="text-primary underline underline-offset-2">
                  telegram.org/apps
                </a>
                .
              </p>
            </Step>
            <Step n={2} title="Tạo mã kết nối">
              {code ? (
                <p>
                  Mã của bạn: <code className="rounded bg-muted px-1 font-semibold text-foreground">{code.code}</code> · dùng 1 lần, hết hạn lúc {expiresAt}.
                </p>
              ) : (
                <Button className="w-full" onClick={createCode}>
                  <Send /> Tạo mã kết nối
                </Button>
              )}
            </Step>
            <Step n={3} title={<>Mở {bot} trên Telegram và bấm <b>Start</b></>}>
              {code ? (
                <>
                  <a href={code.url} target="_blank" rel="noreferrer" className={cn(buttonVariants({ size: 'lg' }), 'w-full')}>
                    <ExternalLink /> Mở Telegram
                  </a>
                  <p>
                    Telegram mở khung chat với {bot}. Bấm nút <b>Start</b> (hoặc <b>Bắt đầu</b>) ở cuối màn hình. Trên máy tính, nếu trình duyệt hiện trang t.me thì bấm <b>Start Bot</b> hoặc <b>Open in Telegram</b>.
                  </p>
                </>
              ) : (
                <p>Nút “Mở Telegram” hiện ra sau khi tạo mã ở bước 2.</p>
              )}
            </Step>
            <Step n={4} title={<>Bot trả lời “✅ Đã kết nối” là xong</>}>
              {code ? (
                <p className="flex items-center gap-1.5">
                  <RefreshCw className="size-3.5 animate-spin" /> Đang chờ Telegram xác nhận, trang này tự cập nhật.
                </p>
              ) : (
                <p>Trang này tự cập nhật, không cần tải lại.</p>
              )}
            </Step>
          </ol>

          {code && (
            <>
              <div className="space-y-2 rounded-lg border border-dashed p-3">
                <p className="text-xs font-medium">Nút “Mở Telegram” không mở được? Làm tay:</p>
                <ol className="list-decimal space-y-1 pl-4 text-xs text-muted-foreground">
                  <li>
                    Mở Telegram, gõ <b>{bot}</b> vào ô tìm kiếm rồi chọn bot.
                  </li>
                  <li>Gửi cho bot đúng tin nhắn dưới đây (bấm sao chép rồi dán):</li>
                </ol>
                <CopyField label="Tin nhắn kết nối" value={code.command} />
                <CopyField label="Hoặc gửi link này cho người cần kết nối (mở trên máy có Telegram)" value={code.url} />
              </div>
              <div className="flex gap-2">
                <Button size="sm" variant="ghost" className="flex-1" onClick={() => setCode(null)}>
                  Huỷ
                </Button>
                <Button size="sm" variant="outline" className="flex-1" onClick={createCode}>
                  <RefreshCw /> Tạo mã mới
                </Button>
              </div>
            </>
          )}
        </div>
      </Section>

      <Section title="Ghi chung trong nhóm Telegram (tuỳ chọn)">
        <div className="space-y-4 px-3 pb-3 text-sm">
          <p className="text-xs text-muted-foreground">Hợp khi nhiều người cùng ghi chi phí, ví dụ hai vợ chồng hoặc người quản lý. Mọi khoản trong nhóm đều vào sổ này.</p>
          <ol className="space-y-4">
            <Step n={1} title="Thêm bot vào nhóm">
              <p>
                Mở nhóm → bấm tên nhóm → <b>Thêm thành viên</b> → tìm <b>{bot}</b> → <b>Thêm</b>.
              </p>
            </Step>
            <Step n={2} title="Mỗi người tự kết nối tài khoản Telegram của mình">
              <p>Người nào muốn ghi thì làm các bước kết nối ở trên (bạn tạo mã rồi gửi link cho họ). Tin của người chưa kết nối sẽ không được ghi, bot sẽ nhắc họ kết nối.</p>
            </Step>
            <Step n={3} title="Nhắn trong nhóm như chat riêng">
              {info.reads_group_messages ? (
                <p>Bot đọc mọi tin trong nhóm, tin nào có số tiền sẽ được ghi.</p>
              ) : (
                <p className="rounded-lg bg-amber-500/10 p-2 text-amber-700 dark:text-amber-400">
                  Bot đang bật privacy mode nên trong nhóm chỉ thấy lệnh và tin trả lời bot (chat riêng vẫn ghi bình thường). Người quản trị bot cần tắt: BotFather → /setprivacy → chọn bot → Disable, rồi xoá và thêm lại bot vào nhóm.
                </p>
              )}
            </Step>
          </ol>
        </div>
      </Section>

      <Section title="Cách nhắn">
        <div className="divide-y px-3 pb-1 text-sm">
          {examples.map((ex) => (
            <div key={ex.text} className="flex items-baseline justify-between gap-3 py-2">
              <code className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-xs">{ex.text}</code>
              <span className="text-right text-xs text-muted-foreground">{ex.desc}</span>
            </div>
          ))}
        </div>
        <p className="px-3 pb-3 text-xs text-muted-foreground">
          Bot tự chọn danh mục. Sai thì bấm ✏️ trên tin trả lời để chọn lại, bot nhớ cho lần sau (xem ở Luật phân loại). Trả lời tin của bot bằng tên danh mục, số tiền mới hoặc “xoá” để sửa nhanh; sửa tin gốc thì giao dịch sửa theo. Gõ /tong để xem tổng tháng.
        </p>
      </Section>
    </div>
  )
}

function UnparsedPanel() {
  const events = useRawEvents(['unparsed', 'error'])
  const { engine, db } = useThuChi()
  const [open, setOpen] = useState<string | null>(null)
  return (
    <Section>
      <div className="divide-y">
        {events.map((e) => (
          <div key={e.id} className="space-y-2 px-3 py-2.5">
            <button type="button" className="w-full text-left" onClick={() => setOpen(open === e.id ? null : e.id)}>
              <span className="flex items-center justify-between gap-2 text-sm font-medium">
                <span className="truncate">{e.source_key || e.source}</span>
                <span className="shrink-0 text-xs text-muted-foreground">{relativeTime(e.received_at)}</span>
              </span>
              <span className="block text-xs break-words text-amber-700 dark:text-amber-400">{e.error}</span>
            </button>
            {open === e.id && e.payload && <pre className="max-h-60 overflow-auto rounded bg-muted p-2 text-[11px] whitespace-pre-wrap break-all">{e.payload}</pre>}
            <div className="flex gap-2">
              <Button size="xs" variant="outline" onClick={() => engine.action('reparse_raw', { id: e.id }, async () => void (await db.rawEvents.update(e.id, { status: 'pending' })))}>
                <RefreshCw /> Đọc lại
              </Button>
              <Button size="xs" variant="ghost" onClick={() => engine.action('dismiss_raw', { id: e.id }, async () => void (await db.rawEvents.update(e.id, { status: 'dismissed' })))}>
                Bỏ qua
              </Button>
            </div>
          </div>
        ))}
        {events.length === 0 && <Empty>Không có thông báo nào lỗi</Empty>}
      </div>
    </Section>
  )
}
