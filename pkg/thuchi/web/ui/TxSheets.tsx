import { useMemo, useState } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from '@/components/ui/sheet'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Trash2, MapPin, TransferIcon, Check, X, Pencil, RefreshCw } from '@/components/icons'
import { cn } from '@/lib/utils'
import { useAccounts, useCategories, useParties, useRules, useThuChi } from '../hooks'
import { money, parseAmount, recurrenceLabel, timeLabel } from '../format'
import { suggestCategory } from '../classify'
import { categoryTree } from '../stats'
import { isManual, localTx, shiftBalances } from '../db'
import { uuidv7 } from '../sync'
import { isDebtKind, type Category, type DebtKind, type Direction, type Transaction, type TxKind } from '../types'
import { Amount, CategoryIcon } from './common'
import { txTitle } from './icons'

export function CategoryGrid({ categories, value, onChange }: { categories: Category[]; value: string | null; onChange: (id: string) => void }) {
  const groups = useMemo(() => {
    const out: { parent: Category | null; items: Category[] }[] = []
    for (const n of categoryTree(categories)) {
      const last = out[out.length - 1]
      if (n.children.length) out.push({ parent: n.category, items: [n.category, ...n.children] })
      else if (last && !last.parent) last.items.push(n.category)
      else out.push({ parent: null, items: [n.category] })
    }
    return out
  }, [categories])
  return (
    <div className="space-y-2">
      {groups.map((g, i) => (
        <div key={g.parent?.id ?? `g${i}`} className={cn(g.parent && 'rounded-lg bg-muted/40 p-1.5')}>
          {g.parent && <p className="px-1 pb-1 text-xs font-medium text-muted-foreground">{g.parent.name}</p>}
          <div className="grid grid-cols-4 gap-2">
            {g.items.map((c) => (
              <button
                key={c.id}
                type="button"
                onClick={() => onChange(c.id)}
                className={cn(
                  'flex min-h-16 flex-col items-center justify-center gap-1 rounded-lg border p-1.5 text-center text-xs',
                  value === c.id ? 'border-primary bg-primary/10' : 'border-transparent hover:bg-muted',
                )}
              >
                <CategoryIcon category={c} />
                <span className="line-clamp-2 leading-tight">{c.name}</span>
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function localInputValue(iso: string) {
  const d = new Date(iso)
  const off = d.getTimezoneOffset() * 60000
  return new Date(d.getTime() - off).toISOString().slice(0, 16)
}

const DEBT_FORM: { kind: DebtKind; label: string; direction: Direction }[] = [
  { kind: 'lend', label: 'Cho vay', direction: 'out' },
  { kind: 'borrow', label: 'Đi vay', direction: 'in' },
  { kind: 'collect', label: 'Thu nợ', direction: 'in' },
  { kind: 'repay', label: 'Trả nợ', direction: 'out' },
]

function PartyInput({ id, value, onChange, onCommit }: { id: string; value: string; onChange: (v: string) => void; onCommit?: (v: string) => void }) {
  const parties = useParties()
  const q = value.trim().toLowerCase()
  const hints = parties.filter((p) => p.toLowerCase() !== q && p.toLowerCase().includes(q)).slice(0, 6)
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>Người / đối tượng</Label>
      <Input id={id} maxLength={100} value={value} onChange={(e) => onChange(e.target.value)} onBlur={() => onCommit?.(value)} placeholder="vd: Anh Nam, Công ty A" autoComplete="off" />
      {hints.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {hints.map((p) => (
            <button
              key={p}
              type="button"
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => {
                onChange(p)
                onCommit?.(p)
              }}
              className="max-w-full truncate rounded-full border px-2.5 py-1 text-xs hover:bg-muted"
            >
              {p}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function DebtField({ tx }: { tx: Transaction }) {
  const { engine } = useThuChi()
  const [party, setParty] = useState(tx.party ?? '')
  const options: { kind: TxKind; label: string }[] = [{ kind: 'normal', label: 'Không' }, ...DEBT_FORM.filter((o) => o.direction === tx.direction)]
  const savedParty = tx.party ?? ''
  function savePartyValue(v: string) {
    const next = v.trim()
    if (next !== savedParty) void engine.patch('transactions', tx.id, { party: next })
  }
  return (
    <div className="space-y-2">
      <Label>Vay nợ</Label>
      <div className="grid grid-cols-3 gap-1 rounded-lg bg-muted p-1">
        {options.map((o) => (
          <button
            key={o.kind}
            type="button"
            onClick={() => o.kind !== tx.kind && void engine.patch('transactions', tx.id, { kind: o.kind, ...(o.kind !== 'normal' && party.trim() !== savedParty ? { party: party.trim() } : {}) })}
            className={cn('truncate rounded-md py-1.5 text-sm', tx.kind === o.kind ? 'bg-background font-medium shadow-sm' : 'text-muted-foreground')}
          >
            {o.label}
          </button>
        ))}
      </div>
      {isDebtKind(tx.kind) && (
        <PartyInput
          id="tx-party"
          value={party}
          onChange={setParty}
          onCommit={savePartyValue}
        />
      )}
    </div>
  )
}

function NoteField({ id, initial }: { id: string; initial: string }) {
  const { engine } = useThuChi()
  const [note, setNote] = useState(initial)
  return (
    <div className="space-y-1.5">
      <Label htmlFor="tx-note">Ghi chú</Label>
      <Textarea
        id="tx-note"
        value={note}
        onChange={(e) => setNote(e.target.value)}
        onBlur={() => note !== initial && void engine.patch('transactions', id, { note })}
        rows={2}
        maxLength={2000}
        placeholder="Thêm ghi chú…"
      />
    </div>
  )
}

export function TransactionSheet({ txId, onClose }: { txId: string | null; onClose: () => void }) {
  const { engine, db } = useThuChi()
  const categories = useCategories()
  const accounts = useAccounts()
  const [learn, setLearn] = useState(true)
  const [editing, setEditing] = useState(false)
  const tx = useLiveQuery(() => (txId ? db.transactions.get(txId) : undefined), [db, txId])
  const pair = useLiveQuery(() => (tx?.pair_id ? db.transactions.get(tx.pair_id) : undefined), [db, tx?.pair_id])
  const outId = tx?.kind === 'transfer' ? (tx.direction === 'out' ? tx.id : tx.pair_id) : null
  const fee = useLiveQuery(() => (outId ? db.transactions.where('pair_id').equals(outId).filter((t) => t.kind === 'normal').first() : undefined), [db, outId])
  const recurrence = useLiveQuery(() => (tx?.recurrence_id ? db.recurrences.get(tx.recurrence_id) : undefined), [db, tx?.recurrence_id])

  if (!txId || !tx) return null
  const live = tx
  const expenseSide = live.direction === 'out' || live.kind === 'refund'
  const kindCats = categories.filter((c) => c.kind === (expenseSide ? 'expense' : 'income'))
  const account = accounts.find((a) => a.id === live.account_id)
  const manual = isManual(live)
  const manualPair = manual && live.kind === 'transfer' && !!pair && isManual(pair) && pair.kind === 'transfer' && pair.pair_id === live.id && pair.direction !== live.direction ? pair : undefined
  const pairAccount = pair ? accounts.find((a) => a.id === pair.account_id) : undefined

  function close() {
    setEditing(false)
    onClose()
  }

  async function setCategory(id: string) {
    await engine.action('set_category', { id: live.id, category_id: id, learn }, async () => {
      await db.transactions.update(live.id, { category_id: id, needs_review: false })
    })
    toast.success(learn ? 'Đã phân loại và ghi nhớ cho giao dịch tương tự' : 'Đã phân loại')
    close()
  }

  async function confirmPair(ok: boolean) {
    await engine.action(ok ? 'confirm_transfer' : 'reject_transfer', { a: live.id }, async () => {
      const ids = [live.id, ...(live.pair_id ? [live.pair_id] : [])]
      for (const id of ids) {
        await db.transactions.update(id, ok ? { kind: 'transfer', pair_status: 'confirmed', needs_review: false, category_id: null } : { kind: 'normal', pair_status: 'rejected', needs_review: false })
      }
    })
    close()
  }

  async function toggleTransfer() {
    const toTransfer = live.kind !== 'transfer'
    await engine.action(toTransfer ? 'mark_transfer' : 'unmark_transfer', { id: live.id }, async () => {
      await db.transactions.update(live.id, toTransfer ? { kind: 'transfer', pair_status: 'confirmed', needs_review: false, category_id: null } : { kind: 'normal', pair_status: '' })
    })
    close()
  }

  async function remove() {
    if (manualPair) {
      const rows = [live, manualPair, ...(fee ? [fee] : [])]
      await engine.action('delete_transfer', { id: live.id }, async () => {
        await db.transactions.bulkDelete(rows.map((r) => r.id))
        return shiftBalances(db, rows.map((r) => [r, null]))
      })
    } else {
      await engine.remove('transactions', live.id, () => shiftBalances(db, [[live, null]]))
    }
    toast.success('Đã xoá giao dịch')
    close()
  }

  async function attachLocation() {
    if (!('geolocation' in navigator)) return
    navigator.geolocation.getCurrentPosition(
      async (pos) => {
        await engine.patch('transactions', live.id, { lat: pos.coords.latitude, lng: pos.coords.longitude })
        toast.success('Đã gắn vị trí hiện tại')
      },
      () => toast.error('Không lấy được vị trí'),
      { timeout: 8000, maximumAge: 60000 },
    )
  }

  return (
    <Sheet open={!!txId} onOpenChange={(o) => !o && close()}>
      <SheetContent side="bottom" className="max-h-[92dvh] overflow-y-auto rounded-t-2xl pb-[max(1rem,env(safe-area-inset-bottom))]">
        {editing ? (
          <>
            <SheetHeader className="pb-0">
              <SheetTitle>Sửa giao dịch</SheetTitle>
              <SheetDescription>{txTitle(live)}</SheetDescription>
            </SheetHeader>
            <QuickAddForm editing={{ tx: live, pair: manualPair, fee: manualPair ? fee : undefined }} onDone={close} />
          </>
        ) : (
          <>
            <SheetHeader className="pb-0">
              <SheetTitle className="pr-10">{txTitle(live)}</SheetTitle>
              <SheetDescription className="flex flex-wrap items-center gap-x-2">
                <Amount tx={live} className="text-lg" />
                <span>
                  {new Date(live.occurred_at).toLocaleDateString('vi-VN')} {timeLabel(live.occurred_at)} ·{' '}
                  {manualPair
                    ? live.direction === 'out'
                      ? `${account?.name ?? '?'} → ${pairAccount?.name ?? '?'}`
                      : `${pairAccount?.name ?? '?'} → ${account?.name ?? '?'}`
                    : (account?.name ?? 'Không rõ tài khoản')}
                </span>
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 px-4">
              {live.raw_content && <p className="rounded-lg bg-muted px-3 py-2 text-xs break-words text-muted-foreground">{live.raw_content}</p>}
              {live.recurrence_id && (
                <p className="flex items-center gap-2 rounded-lg bg-primary/10 px-3 py-2 text-xs text-primary">
                  <RefreshCw className="size-4 shrink-0" />
                  <span className="min-w-0">
                    Tự ghi từ định kỳ{recurrence ? ` ${recurrenceLabel(recurrence).toLowerCase()}${recurrence.merchant ? ` “${recurrence.merchant}”` : ''}` : ' (đã xoá)'}
                  </span>
                </p>
              )}
              {manualPair && fee && <p className="text-sm text-muted-foreground">Phí chuyển: {money(fee.amount)}</p>}

              {live.pair_status === 'suggested' && (
                <div className="space-y-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm">
                  <p>
                    Có thể là chuyển nội bộ {pair ? `với ${money(pair.amount)} ở ${pairAccount?.name ?? 'tài khoản khác'}` : ''}. Xác nhận để không tính vào thu/chi.
                  </p>
                  <div className="flex gap-2">
                    <Button size="sm" onClick={() => confirmPair(true)}>
                      <Check /> Đúng, chuyển nội bộ
                    </Button>
                    <Button size="sm" variant="outline" onClick={() => confirmPair(false)}>
                      <X /> Không phải
                    </Button>
                  </div>
                </div>
              )}

              {live.kind !== 'transfer' && live.kind !== 'card_payment' && live.kind !== 'adjustment' && !isDebtKind(live.kind) && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between gap-2">
                    <Label>Danh mục</Label>
                    <label className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Checkbox checked={learn} onCheckedChange={(v) => setLearn(!!v)} />
                      Áp dụng cho giao dịch giống vậy
                    </label>
                  </div>
                  <CategoryGrid categories={kindCats} value={live.kind === 'refund' ? live.refund_of_category_id : live.category_id} onChange={setCategory} />
                </div>
              )}

              {(live.kind === 'normal' || isDebtKind(live.kind)) && <DebtField key={live.id} tx={live} />}

              {!manualPair && <NoteField key={live.id} id={live.id} initial={live.note} />}

              <div className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2">
                <div>
                  <p className="text-sm font-medium">Không tính vào thống kê</p>
                  <p className="text-xs text-muted-foreground">Ẩn khỏi tổng chi/thu, vẫn giữ trong danh sách</p>
                </div>
                <Switch checked={live.excluded} onCheckedChange={(v) => engine.patch('transactions', live.id, { excluded: v })} />
              </div>

              <div className="flex flex-wrap gap-2">
                {manual && (
                  <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
                    <Pencil /> Sửa
                  </Button>
                )}
                {!manualPair && live.kind !== 'adjustment' && (
                  <Button variant="outline" size="sm" onClick={toggleTransfer}>
                    <TransferIcon /> {live.kind === 'transfer' ? 'Không phải chuyển nội bộ' : 'Đánh dấu chuyển nội bộ'}
                  </Button>
                )}
                <Button variant="outline" size="sm" onClick={attachLocation}>
                  <MapPin /> {live.lat != null ? 'Cập nhật vị trí' : 'Gắn vị trí hiện tại'}
                </Button>
                {manual && (
                  <Button variant="destructive" size="sm" onClick={remove}>
                    <Trash2 /> Xoá
                  </Button>
                )}
              </div>
              <p className="text-xs text-muted-foreground">
                Nguồn: {live.sources.join(', ')}
                {live.reconciled ? ' · đã đối chiếu' : ''}
              </p>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}

export function QuickAddSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <Sheet open={open} onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="bottom" className="max-h-[92dvh] overflow-y-auto rounded-t-2xl pb-[max(1rem,env(safe-area-inset-bottom))]">
        <SheetHeader className="pb-0">
          <SheetTitle>Thêm giao dịch</SheetTitle>
          <SheetDescription>Lưu ngay trên máy, tự đồng bộ khi có mạng</SheetDescription>
        </SheetHeader>
        {open && <QuickAddForm onDone={onClose} />}
      </SheetContent>
    </Sheet>
  )
}

type FormKind = 'out' | 'in' | 'transfer' | 'debt'

const FORM_KINDS: { key: FormKind; label: string }[] = [
  { key: 'out', label: 'Chi' },
  { key: 'in', label: 'Thu' },
  { key: 'transfer', label: 'Chuyển tiền' },
  { key: 'debt', label: 'Vay nợ' },
]

interface Editing {
  tx: Transaction
  pair?: Transaction
  fee?: Transaction
}

function initialForm(e?: Editing) {
  const base = { debtKind: 'lend' as DebtKind, party: '' }
  if (!e) return { ...base, kind: 'out' as FormKind, amount: '', title: '', category: null, from: '', to: '', fee: '', when: localInputValue(new Date().toISOString()) }
  if (e.pair) {
    const out = e.tx.direction === 'out' ? e.tx : e.pair
    const inn = out === e.tx ? e.pair : e.tx
    return { ...base, kind: 'transfer' as FormKind, amount: String(out.amount), title: out.note, category: null, from: out.account_id ?? '', to: inn.account_id ?? '', fee: e.fee ? String(e.fee.amount) : '', when: localInputValue(out.occurred_at) }
  }
  const debt = isDebtKind(e.tx.kind)
  return {
    debtKind: debt ? (e.tx.kind as DebtKind) : base.debtKind,
    party: e.tx.party ?? '',
    kind: (debt ? 'debt' : e.tx.direction) as FormKind,
    amount: String(e.tx.amount),
    title: e.tx.merchant,
    category: e.tx.category_id,
    from: e.tx.account_id ?? '',
    to: '',
    fee: '',
    when: localInputValue(e.tx.occurred_at),
  }
}

function allowedKinds(e?: Editing): FormKind[] {
  if (!e) return FORM_KINDS.map((k) => k.key)
  if (e.pair) return ['transfer']
  if (e.tx.kind === 'normal') return ['out', 'in', 'transfer', 'debt']
  if (isDebtKind(e.tx.kind)) return ['out', 'in', 'debt']
  return ['out', 'in']
}

function QuickAddForm({ editing, onDone }: { editing?: Editing; onDone: () => void }) {
  const { engine, db } = useThuChi()
  const categories = useCategories()
  const accounts = useAccounts()
  const rules = useRules()
  const [init] = useState(() => initialForm(editing))
  const [kind, setKind] = useState<FormKind>(init.kind)
  const [amountText, setAmountText] = useState(init.amount)
  const [title, setTitle] = useState(init.title)
  const [pickedCategory, setPickedCategory] = useState<string | null>(init.category)
  const [pickedAccount, setPickedAccount] = useState(init.from)
  const [pickedTo, setPickedTo] = useState(init.to)
  const [feeText, setFeeText] = useState(init.fee)
  const [when, setWhen] = useState(init.when)
  const [debtKind, setDebtKind] = useState<DebtKind>(init.debtKind)
  const [party, setParty] = useState(init.party)

  const transfer = kind === 'transfer'
  const debt = kind === 'debt'
  const direction: Direction = debt ? DEBT_FORM.find((d) => d.kind === debtKind)!.direction : kind === 'in' ? 'in' : 'out'
  const showCategory = !transfer && !debt && (!editing || editing.tx.kind === 'normal' || isDebtKind(editing.tx.kind))
  const kindCats = useMemo(() => categories.filter((c) => c.kind === (direction === 'out' ? 'expense' : 'income')), [categories, direction])
  const suggested = useMemo(() => (showCategory && title.trim() ? suggestCategory(title, direction, categories, rules) : null), [showCategory, title, direction, categories, rules])
  const categoryId = showCategory ? (pickedCategory ?? suggested) : null
  const swapSides = !!editing && !editing.pair && editing.tx.direction === 'in'
  const accountId = pickedAccount || (transfer && swapSides ? '' : (accounts.find((a) => a.kind === 'cash') ?? accounts[0])?.id || '')
  const toId = pickedTo || accounts.find((a) => a.id !== accountId)?.id || ''
  const accountItems = accounts.map((a) => ({ value: a.id, label: a.name }))

  const amount = parseAmount(amountText)
  const fee = feeText.trim() ? parseAmount(feeText) : 0
  const valid = Number.isFinite(amount) && amount > 0 && !!accountId && (!transfer || (!!toId && toId !== accountId && Number.isFinite(fee) && fee >= 0)) && (!debt || !!party.trim())
  const kinds = allowedKinds(editing)

  function selectKind(next: FormKind) {
    if (swapSides && next === 'transfer' && kind !== 'transfer') {
      setPickedTo(accountId)
      setPickedAccount('')
    } else if (swapSides && next !== 'transfer' && kind === 'transfer') {
      setPickedAccount(toId)
      setPickedTo('')
    }
    setKind(next)
    setPickedCategory(null)
  }

  async function submitTransfer() {
    const e = editing
    const original = e ? (e.pair && e.tx.direction !== 'out' ? e.pair : e.tx).occurred_at : null
    const occurred = original && when === init.when ? original : new Date(when).toISOString()
    let outId = uuidv7()
    let inId = uuidv7()
    if (e?.pair) {
      outId = e.tx.direction === 'out' ? e.tx.id : e.pair.id
      inId = e.tx.direction === 'out' ? e.pair.id : e.tx.id
    } else if (e) {
      if (e.tx.direction === 'out') outId = e.tx.id
      else inId = e.tx.id
    }
    const feeId = e?.fee?.id ?? uuidv7()
    const feeCat = categories.find((c) => c.key === 'phi_ngan_hang' && c.kind === 'expense')
    const note = title.trim()
    const args = { out_id: outId, in_id: inId, from_account_id: accountId, to_account_id: toId, amount, fee, fee_id: feeId, occurred_at: occurred, note }
    const fresh = [outId, inId, feeId].filter((id) => id !== e?.tx.id && id !== e?.pair?.id && id !== e?.fee?.id)
    await engine.action(
      e ? 'update_transfer' : 'create_transfer',
      args,
      async () => {
        const [prevOut, prevIn, prevFee] = await db.transactions.bulkGet([outId, inId, feeId])
        const side = { amount, occurred_at: occurred, note, kind: 'transfer' as const, pair_status: 'confirmed' as const, category_id: null, needs_review: false }
        const nextOut = { ...localTx({ id: outId, account_id: accountId, direction: 'out', amount, occurred_at: occurred }), ...prevOut, ...side, account_id: accountId, direction: 'out' as const, pair_id: inId }
        const nextIn = { ...localTx({ id: inId, account_id: toId, direction: 'in', amount, occurred_at: occurred }), ...prevIn, ...side, account_id: toId, direction: 'in' as const, pair_id: outId }
        await db.transactions.bulkPut([nextOut, nextIn])
        let nextFee: Transaction | null = null
        if (fee > 0) {
          nextFee = { ...localTx({ id: feeId, account_id: accountId, direction: 'out', amount: fee, occurred_at: occurred, merchant: 'Phí chuyển khoản', category_id: feeCat?.id ?? null }), ...prevFee, account_id: accountId, direction: 'out', amount: fee, occurred_at: occurred, pair_id: outId, deleted_at: null }
          await db.transactions.put(nextFee)
        } else if (prevFee) {
          await db.transactions.delete(feeId)
        }
        return shiftBalances(db, [
          [prevOut, nextOut],
          [prevIn, nextIn],
          [prevFee, nextFee],
        ])
      },
      fresh.map((id) => ({ entity: 'transactions', id })),
    )
  }

  async function submitNormal() {
    const occurred = new Date(when).toISOString()
    const merchant = title.trim()
    const who = debt ? party.trim() : ''
    if (editing) {
      const before = editing.tx
      const changes: Record<string, unknown> = {}
      if (debt && debtKind !== before.kind) changes.kind = debtKind
      if (!debt && isDebtKind(before.kind)) changes.kind = 'normal'
      if (debt && who !== (before.party ?? '')) changes.party = who
      if (accountId !== before.account_id) changes.account_id = accountId
      if (direction !== before.direction) changes.direction = direction
      if (amount !== before.amount) changes.amount = amount
      if (when !== init.when) changes.occurred_at = occurred
      if (merchant !== before.merchant) changes.merchant = merchant
      if (showCategory && categoryId !== before.category_id) changes.category_id = categoryId
      if (Object.keys(changes).length === 0) return
      await engine.patch('transactions', before.id, changes, async () => shiftBalances(db, [[before, await db.transactions.get(before.id)]]))
      return
    }
    const id = uuidv7()
    const extra = debt ? { kind: debtKind, party: who } : {}
    const row = localTx({ id, account_id: accountId, direction, amount, occurred_at: occurred, merchant, category_id: categoryId, ...extra })
    await engine.create(
      'transactions',
      { id, account_id: accountId, direction, amount, occurred_at: occurred, merchant, raw_content: '', category_id: categoryId, ...extra },
      row as unknown as Record<string, unknown>,
      () => shiftBalances(db, [[null, row]]),
    )
  }

  async function submit() {
    if (!valid) return
    if (transfer) await submitTransfer()
    else await submitNormal()
    toast.success(editing ? 'Đã cập nhật giao dịch' : 'Đã lưu giao dịch')
    onDone()
  }

  return (
    <form
      className="space-y-4 px-4"
      onSubmit={(e) => {
        e.preventDefault()
        void submit()
      }}
    >
      {kinds.length > 1 && (
        <div className="grid auto-cols-fr grid-flow-col gap-1 rounded-lg bg-muted p-1">
          {FORM_KINDS.filter((k) => kinds.includes(k.key)).map((k) => (
            <button
              key={k.key}
              type="button"
              onClick={() => selectKind(k.key)}
              className={cn('truncate rounded-md px-1 py-2 text-[13px] font-medium sm:text-sm', kind === k.key ? 'bg-background shadow-sm' : 'text-muted-foreground')}
            >
              {k.label}
            </button>
          ))}
        </div>
      )}
      {debt && (
        <>
          <div className="grid grid-cols-4 gap-1">
            {DEBT_FORM.map((d) => (
              <button
                key={d.kind}
                type="button"
                onClick={() => setDebtKind(d.kind)}
                className={cn('truncate rounded-md border px-1 py-1.5 text-[13px] sm:text-sm', debtKind === d.kind ? 'border-primary bg-primary/10 font-medium' : 'text-muted-foreground')}
              >
                {d.label}
              </button>
            ))}
          </div>
          <PartyInput id="qa-party" value={party} onChange={setParty} />
        </>
      )}
      <div className="space-y-1.5">
        <Label htmlFor="qa-amount">Số tiền</Label>
        <Input id="qa-amount" inputMode="decimal" autoFocus={!editing} placeholder="vd: 45k, 1,2tr, 150000" value={amountText} onChange={(e) => setAmountText(e.target.value)} className="text-lg font-semibold" />
        {Number.isFinite(amount) && amount > 0 && <p className="text-xs text-muted-foreground">{money(amount)}</p>}
      </div>
      <div className="space-y-1.5">
        <Label htmlFor="qa-title">{transfer ? 'Ghi chú' : 'Nội dung'}</Label>
        <Input id="qa-title" maxLength={transfer ? 2000 : 200} placeholder={transfer ? 'vd: Nạp ví, rút tiền mặt…' : debt ? 'vd: Mượn tiền sửa xe…' : 'vd: Phở Thìn, Grab…'} value={title} onChange={(e) => setTitle(e.target.value)} />
      </div>
      {showCategory && (
        <div className="space-y-1.5">
          <Label>Danh mục</Label>
          <CategoryGrid categories={kindCats} value={categoryId} onChange={setPickedCategory} />
        </div>
      )}
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label>{transfer ? 'Từ tài khoản' : 'Tài khoản'}</Label>
          <Select value={accountId} onValueChange={(v) => setPickedAccount(String(v))} items={accountItems}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Chọn tài khoản" />
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
        {transfer ? (
          <div className="space-y-1.5">
            <Label>Đến tài khoản</Label>
            <Select value={toId} onValueChange={(v) => setPickedTo(String(v))} items={accountItems}>
              <SelectTrigger className="w-full" aria-invalid={toId === accountId || undefined}>
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
        ) : (
          <WhenField value={when} onChange={setWhen} />
        )}
      </div>
      {transfer && (
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <Label htmlFor="qa-fee">Phí chuyển</Label>
            <Input id="qa-fee" inputMode="decimal" placeholder="0" value={feeText} onChange={(e) => setFeeText(e.target.value)} />
          </div>
          <WhenField value={when} onChange={setWhen} />
        </div>
      )}
      {transfer && toId === accountId && <p className="text-xs text-destructive">Chọn hai tài khoản khác nhau</p>}
      <Button type="submit" className="w-full" disabled={!valid}>
        Lưu
      </Button>
    </form>
  )
}

function WhenField({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor="qa-when">Thời gian</Label>
      <Input id="qa-when" type="datetime-local" value={value} onChange={(e) => onChange(e.target.value)} />
    </div>
  )
}
