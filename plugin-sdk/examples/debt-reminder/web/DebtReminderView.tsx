import { useCallback, useEffect, useState } from 'react'
import { createPluginApi, toast, usePluginConfig, type PluginViewProps } from '@qlpt/plugin'
import {
  Badge,
  Button,
  Card,
  EmptyState,
  Skeleton,
  StatCard,
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
  Tabs,
  TabsList,
  TabsTrigger,
} from '@qlpt/plugin/ui'
import { AlertCircle, Check, Clock, Receipt, Settings, Wallet, X } from '@qlpt/plugin/icons'

const api = createPluginApi('debt-reminder')

type PeriodFilter = 'all' | 'this' | 'last'

interface DebtItem {
  id: string
  house_name: string
  room_name: string
  period: string
  total_amount: number
  tenant_names: string[]
  reminded_at: string | null
  need_remind: boolean
}

interface Summary {
  items: DebtItem[]
  count: number
  need_remind: number
  total_amount: number
}

const money = (value: number) => `${new Intl.NumberFormat('vi-VN').format(value)} ₫`

const remindedLabel = (iso: string) =>
  new Date(iso).toLocaleString('vi-VN', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })

function monthKey(offset: number) {
  const date = new Date()
  date.setDate(1)
  date.setMonth(date.getMonth() + offset)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
}

function periodQuery(filter: PeriodFilter) {
  if (filter === 'all') return ''
  return `?period=${monthKey(filter === 'this' ? 0 : -1)}`
}

export function DebtReminderView({ plugin, openSettings }: PluginViewProps) {
  const { config } = usePluginConfig(plugin.id)
  const [filter, setFilter] = useState<PeriodFilter>('all')
  const [summary, setSummary] = useState<Summary | null>(null)
  const [error, setError] = useState('')
  const [busyId, setBusyId] = useState<string | null>(null)
  const settingsKey = JSON.stringify(config?.values ?? {})

  const load = useCallback(async () => {
    try {
      setSummary(await api.get<Summary>(`/summary${periodQuery(filter)}`))
      setError('')
    } catch (err) {
      setError(api.errorMessage(err, 'Không tải được danh sách công nợ'))
    }
  }, [filter])

  useEffect(() => {
    load()
  }, [load, settingsKey])

  const toggleReminded = async (item: DebtItem) => {
    setBusyId(item.id)
    try {
      if (item.need_remind) {
        await api.post(`/reminders/${item.id}`)
        toast.success(`Đã đánh dấu nhắc ${item.room_name}`)
      } else {
        await api.delete(`/reminders/${item.id}`)
      }
      await load()
    } catch (err) {
      toast.error(api.errorMessage(err, 'Không cập nhật được trạng thái nhắc'))
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <Tabs value={filter} onValueChange={(value) => setFilter(value as PeriodFilter)}>
          <TabsList>
            <TabsTrigger value="all">Tất cả</TabsTrigger>
            <TabsTrigger value="this">Tháng này</TabsTrigger>
            <TabsTrigger value="last">Tháng trước</TabsTrigger>
          </TabsList>
        </Tabs>
        <Button variant="ghost" size="sm" onClick={openSettings}>
          <Settings data-icon="inline-start" />
          <span className="hidden sm:inline">Cài đặt</span>
        </Button>
      </div>

      {error && !summary ? (
        <Card>
          <EmptyState
            icon={AlertCircle}
            title="Không tải được công nợ"
            description={error}
            action={<Button variant="outline" size="sm" onClick={load}>Thử lại</Button>}
          />
        </Card>
      ) : !summary ? (
        <div className="space-y-3">
          <div className="grid gap-3 sm:grid-cols-3">
            <Skeleton className="h-24" />
            <Skeleton className="h-24" />
            <Skeleton className="h-24" />
          </div>
          <Skeleton className="h-64" />
        </div>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            <StatCard label="Hóa đơn chưa thu" value={String(summary.count)} icon={Receipt} />
            <StatCard label="Tổng tiền chưa thu" value={money(summary.total_amount)} icon={Wallet} tone="negative" />
            <StatCard label="Cần nhắc khách" value={String(summary.need_remind)} icon={Clock} tone="warning" />
          </div>

          <Card className="gap-0 p-0">
            {summary.items.length === 0 ? (
              <EmptyState icon={Check} title="Không có hóa đơn cần thu" description="Mọi hóa đơn trong kỳ đã được thanh toán." />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="pl-4">Phòng</TableHead>
                    <TableHead className="hidden md:table-cell">Khách</TableHead>
                    <TableHead className="hidden sm:table-cell">Kỳ</TableHead>
                    <TableHead className="text-right">Số tiền</TableHead>
                    <TableHead className="pr-4 text-right">Nhắc</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {summary.items.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="pl-4">
                        <div className="font-medium text-foreground">{item.room_name}</div>
                        <div className="text-xs text-muted-foreground">
                          {item.house_name}
                          <span className="sm:hidden"> · {item.period}</span>
                        </div>
                      </TableCell>
                      <TableCell className="hidden max-w-48 truncate md:table-cell">{item.tenant_names.join(', ') || '—'}</TableCell>
                      <TableCell className="hidden tabular-nums sm:table-cell">{item.period}</TableCell>
                      <TableCell className="text-right font-medium tabular-nums">{money(item.total_amount)}</TableCell>
                      <TableCell className="pr-4 text-right">
                        <div className="flex items-center justify-end gap-2">
                          {item.reminded_at ? (
                            <Badge variant={item.need_remind ? 'outline' : 'secondary'} className="hidden font-normal lg:inline-flex">
                              {remindedLabel(item.reminded_at)}
                            </Badge>
                          ) : null}
                          <Button
                            variant={item.need_remind ? 'outline' : 'ghost'}
                            size="sm"
                            disabled={busyId === item.id}
                            onClick={() => toggleReminded(item)}
                          >
                            {item.need_remind ? <Check data-icon="inline-start" /> : <X data-icon="inline-start" />}
                            {item.need_remind ? 'Đã nhắc' : 'Bỏ'}
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
                <TableFooter>
                  <TableRow>
                    <TableCell className="pl-4 font-medium">Tổng</TableCell>
                    <TableCell className="hidden md:table-cell" />
                    <TableCell className="hidden sm:table-cell" />
                    <TableCell className="text-right font-semibold tabular-nums">{money(summary.total_amount)}</TableCell>
                    <TableCell className="pr-4" />
                  </TableRow>
                </TableFooter>
              </Table>
            )}
          </Card>
        </>
      )}
    </div>
  )
}
