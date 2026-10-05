import { memo, useState, useEffect, useMemo } from 'react'
import { Plus, Receipt, Zap, TrendingUp, CheckCircle2, Clock, FilterX, Download, Pencil, Loader2, Trash2, Send, AlertTriangle } from '@/components/icons'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { PageHeader } from '@/components/shared/PageHeader'
import { EmptyState } from '@/components/shared/EmptyState'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { countActiveInvoiceFilters, invoiceBreakdownKey, invoiceListKey, useInvoiceStore } from '@/data/invoiceData'
import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'
import { type Invoice, getAllInvoices, getInvoiceBreakdown, getInvoiceImageBlob, getInvoicesPage } from '@/api/invoice'
import { sendInvoiceViaZalo } from '@/api/zalo'
import { CreateInvoiceModal } from './modals/CreateInvoiceModal'
import { InvoiceDetailModal } from './modals/InvoiceDetailModal'
import { QuickCreateInvoiceModal } from './modals/QuickCreateInvoiceModal'
import { EditInvoiceModal } from './modals/EditInvoiceModal'
import { BackendImagePreviewModal } from './modals/BackendImagePreviewModal'
import { HouseButtonGroup, HOUSE_BUTTON_LIMIT } from './HouseButtonGroup'
import { toast } from 'sonner'
import { formatCurrency } from '@/utils/format'
import { InvoiceMobileCard } from './invoice/InvoiceMobileCard'
import { SectionCard } from '@/components/shared/SectionCard'
import { BreakdownTable, type BreakdownRow } from '@/components/shared/BreakdownTable'
import type { InvoiceAmounts } from '@/api/invoice'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { DataPagination } from '@/components/shared/DataPagination'
import { Skeleton } from '@/components/ui/skeleton'
import { useQuery } from '@/lib/queryCache'
import { useStableCallback } from '@/hooks/useStableCallback'
import { cn } from '@/lib/utils'

const INVOICE_BREAKDOWN_ITEMS: { key: keyof InvoiceAmounts; label: string; alwaysShow?: boolean; deduction?: boolean }[] = [
  { key: 'room_fee', label: 'Tiền phòng', alwaysShow: true },
  { key: 'electricity_fee', label: 'Tiền điện', alwaysShow: true },
  { key: 'water_fee', label: 'Tiền nước', alwaysShow: true },
  { key: 'wifi_fee', label: 'Tiền mạng Wifi' },
  { key: 'parking_fee', label: 'Tiền gửi xe' },
  { key: 'service_fee', label: 'Phí dịch vụ chung' },
  { key: 'extra_person_fee', label: 'Phụ thu người thêm' },
  { key: 'extra_vehicle_fee', label: 'Phụ thu xe thêm' },
  { key: 'other_fee', label: 'Chi phí phát sinh' },
  { key: 'discount', label: 'Giảm trừ', deduction: true },
]

const EMPTY_INVOICES: Invoice[] = []

const filterControlClass = 'h-11 w-full rounded-md border border-input bg-background px-3 text-base shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground md:h-9 md:text-sm'

const errorText = (error: unknown) =>
  (error as { response?: { data?: { message?: string } } })?.response?.data?.message || 'Không tải được dữ liệu hóa đơn.'

// View quản lý hóa đơn: lọc, thống kê nhanh, danh sách (card mobile + table desktop) và các modal tạo/sửa/xem.
export function InvoicesView() {
  const { houses } = useHouseStore()
  const { getRoomsByHouse } = useRoomStore()
  const selectedHouse = useSelectedStore(state => state.selectedHouse)
  const selectHouse = useSelectedStore(state => state.selectHouse)

  const invoiceFilter = useInvoiceStore(state => state.invoiceFilter)
  const setInvoiceFilter = useInvoiceStore(state => state.setInvoiceFilter)
  const clearInvoiceFilter = useInvoiceStore(state => state.clearInvoiceFilter)
  const deleteInvoice = useInvoiceStore(state => state.deleteInvoice)

  const listQuery = useQuery(invoiceListKey(invoiceFilter), () => getInvoicesPage(invoiceFilter))
  const breakdownQuery = useQuery(invoiceBreakdownKey(invoiceFilter), () =>
    getInvoiceBreakdown({
      house_id: invoiceFilter.house_id || undefined,
      room_id: invoiceFilter.room_id || undefined,
      period: invoiceFilter.period || undefined,
      status: invoiceFilter.status || undefined,
    }),
  )
  const invoices = listQuery.data?.items ?? EMPTY_INVOICES
  const total = listQuery.data?.total ?? 0
  const breakdown = breakdownQuery.data ?? null
  const loadError = listQuery.error ?? breakdownQuery.error
  const activeFilterCount = countActiveInvoiceFilters(invoiceFilter)

  useEffect(() => {
    if (!listQuery.data || listQuery.isPlaceholder || listQuery.error !== undefined) return
    const lastPage = Math.max(1, Math.ceil(listQuery.data.total / invoiceFilter.limit))
    if (invoiceFilter.page > lastPage) setInvoiceFilter({ page: lastPage })
  }, [listQuery.data, listQuery.isPlaceholder, listQuery.error, invoiceFilter.page, invoiceFilter.limit, setInvoiceFilter])

  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showQuickCreateModal, setShowQuickCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [isDownloadingAll, setIsDownloadingAll] = useState(false)
  const [previewData, setPreviewData] = useState<{ url: string, filename: string } | null>(null)
  const [openedInvoice, setOpenedInvoice] = useState<Invoice | null>(null)
  const [invoiceToDelete, setInvoiceToDelete] = useState<Invoice | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)
  const selectedInvoice = openedInvoice ? invoices.find(inv => inv.id === openedInvoice.id) ?? openedInvoice : null

  useEffect(() => {
    if (selectedHouse && invoiceFilter.house_id !== selectedHouse.id) {
      setInvoiceFilter({ house_id: selectedHouse.id, room_id: '' })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedHouse])

  const handleClearFilter = () => {
    selectHouse(null);
    clearInvoiceFilter();
  };

  const [filterRooms, setFilterRooms] = useState<{ id: string; name: string }[]>([])

  const handleHouseFilterChange = (houseId: string) => {
    setInvoiceFilter({ house_id: houseId, room_id: '' });
    selectHouse(houses.find(h => h.id === houseId) || null);
  };

  useEffect(() => {
    if (invoiceFilter.house_id) {
      getRoomsByHouse(invoiceFilter.house_id).then(res => setFilterRooms(res)).catch(console.error)
    } else {
      setFilterRooms([])
    }
  }, [invoiceFilter.house_id, getRoomsByHouse])

  const houseNames = useMemo(() => new Map(houses.map(h => [h.id, h.name])), [houses])

  const stats = useMemo(() => {
    if (!breakdown) return null;
    return {
      totalInvoices: breakdown.expected.invoice_count,
      expectedRevenue: breakdown.expected.total_amount,
      collectedRevenue: breakdown.collected.total_amount,
      unpaidRevenue: breakdown.expected.total_amount - breakdown.collected.total_amount,
    };
  }, [breakdown]);

  const breakdownRows = useMemo<BreakdownRow[]>(() => {
    if (!breakdown) return [];
    return INVOICE_BREAKDOWN_ITEMS
      .filter(item => item.alwaysShow || breakdown.expected[item.key] !== 0)
      .map(item => {
        const expected = breakdown.expected[item.key];
        const collected = breakdown.collected[item.key];
        return { key: item.key, label: item.label, deduction: item.deduction, values: [expected, collected, expected - collected] };
      });
  }, [breakdown]);

  const openInvoice = useStableCallback((invoice: Invoice) => setOpenedInvoice(invoice))
  const editInvoice = useStableCallback((invoice: Invoice) => {
    setOpenedInvoice(invoice)
    setShowEditModal(true)
  })

  const handleDownload = useStableCallback(async (invoice: Invoice) => {
    try {
      toast.loading('Đang tạo ảnh hóa đơn...', { id: 'download-invoice' });

      const imageData = await getInvoiceImageBlob(invoice.id);

      const houseNameRaw = houses.find(h => h.id === invoice.house_id)?.name || 'NhaTro';
      const houseName = houseNameRaw.replace(/\s+/g, '');

      const blob = new Blob([imageData], { type: 'image/png' });
      const url = URL.createObjectURL(blob);
      const filename = `${invoice.room_name}_${invoice.period.replace('-', '_')}_${houseName}.png`;
      
      toast.dismiss('download-invoice');
      setPreviewData({ url, filename });
    } catch (error) {
      console.error('Lỗi tải ảnh:', error);
      toast.error('Lỗi khi tải ảnh hóa đơn', { id: 'download-invoice', description: 'Vui lòng thử lại sau.' });
    }
  });

  const handleDelete = useStableCallback((invoice: Invoice) => {
    if (invoice.status === 'PAID') {
      toast.error('Không thể xóa hóa đơn đã thanh toán', { id: 'delete-invoice' });
      return;
    }
    setInvoiceToDelete(invoice);
  });

  const confirmDelete = async () => {
    if (!invoiceToDelete) return;
    setIsDeleting(true);
    const res = await deleteInvoice(invoiceToDelete.id);
    setIsDeleting(false);
    setInvoiceToDelete(null);
    if (res.success) {
      toast.success(res.message, { id: 'delete-invoice' });
    } else {
      toast.error(res.message, { id: 'delete-invoice' });
    }
  };

  const handleDownloadAll = async () => {
    if (total === 0) return;

    setIsDownloadingAll(true);
    toast.loading(`Đang tải ${total} ảnh hóa đơn...`, { id: 'download-all' });

    try {
      const allInvoices = await getAllInvoices({
        house_id: invoiceFilter.house_id,
        room_id: invoiceFilter.room_id,
        period: invoiceFilter.period,
        status: invoiceFilter.status,
      });
      for (let i = 0; i < allInvoices.length; i++) {
        const invoice = allInvoices[i];
        try {
          const imageData = await getInvoiceImageBlob(invoice.id);

          const houseNameRaw = houses.find(h => h.id === invoice.house_id)?.name || 'NhaTro';
          const houseName = houseNameRaw.replace(/\s+/g, '');
          const periodStr = invoice.period.replace('-', '_');

          const blob = new Blob([imageData], { type: 'image/png' });
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = `${invoice.room_name}_${periodStr}_${houseName}.png`;
          
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
          
          // Delay to prevent browser from blocking multiple downloads
          await new Promise(resolve => setTimeout(resolve, 300));
          URL.revokeObjectURL(url);
        } catch (err) {
          console.error(`Lỗi tải hóa đơn ${invoice.room_name}:`, err);
        }
      }
      
      toast.success('Thành công', { id: 'download-all', description: `Đã hoàn tất tải ${allInvoices.length} ảnh hóa đơn.` });
    } catch (error) {
      console.error('Lỗi khi tải tất cả:', error);
      toast.error('Có lỗi xảy ra', { id: 'download-all', description: 'Không thể tải toàn bộ hóa đơn.' });
    } finally {
      setIsDownloadingAll(false);
    }
  };

  const handleSendZalo = useStableCallback(async (invoice: Invoice) => {
    if (invoice.status === 'PAID') {
      toast.error('Hóa đơn đã thanh toán', { id: 'send-zalo' });
      return;
    }
    toast.loading('Đang gửi ảnh hóa đơn qua Zalo...', { id: 'send-zalo' });
    try {
      await sendInvoiceViaZalo(invoice.id);
      toast.success('Đã gửi hóa đơn qua Zalo thành công', { id: 'send-zalo' });
    } catch (error: unknown) {
      console.error('Lỗi gửi Zalo:', error);
      const err = error as { response?: { data?: { message?: string } } };
      toast.error(err.response?.data?.message || 'Không thể gửi qua Zalo', { id: 'send-zalo' });
    }
  });

  const money = (value?: number) => (value === undefined ? '—' : formatCurrency(value))

  return (
    <div className="flex flex-col gap-6 safe-fade-in">
      {/* Modals */}
      {showCreateModal && (
        <CreateInvoiceModal onClose={() => setShowCreateModal(false)} />
      )}
      {showQuickCreateModal && (
        <QuickCreateInvoiceModal onClose={() => setShowQuickCreateModal(false)} />
      )}
      {selectedInvoice && !showEditModal && (
        <InvoiceDetailModal
          invoice={selectedInvoice}
          onClose={() => setOpenedInvoice(null)}
          onEdit={() => setShowEditModal(true)}
        />
      )}
      {showEditModal && selectedInvoice && (
        <EditInvoiceModal
          invoice={selectedInvoice}
          onClose={() => {
            setShowEditModal(false)
            setOpenedInvoice(null)
          }}
        />
      )}
      {invoiceToDelete && (
        <ConfirmDialog
          title="Xóa hóa đơn"
          message={`Bạn có chắc muốn xóa hóa đơn phòng ${invoiceToDelete.room_name} kỳ ${invoiceToDelete.period}?`}
          confirmText="Xóa"
          cancelText="Bỏ qua"
          isLoading={isDeleting}
          onConfirm={confirmDelete}
          onCancel={() => setInvoiceToDelete(null)}
        />
      )}
      {previewData && (
        <BackendImagePreviewModal
          imageUrl={previewData.url}
          filename={previewData.filename}
          title="Xem trước hóa đơn (Mẫu 1)"
          onClose={() => {
            URL.revokeObjectURL(previewData.url)
            setPreviewData(null)
          }}
        />
      )}

      {/* Header */}
      <PageHeader
        title="Quản lý hóa đơn"
        description="Danh sách hóa đơn điện, nước, dịch vụ hàng tháng"
        action={
          <>
            <Button onClick={() => setShowQuickCreateModal(true)} variant="outline" size="lg" className="cursor-pointer">
              <Zap data-icon="inline-start" />
              <span className="sm:hidden">Ghi nhanh</span>
              <span className="hidden sm:inline">Ghi điện nước nhanh</span>
            </Button>
            <Button onClick={() => setShowCreateModal(true)} size="lg" className="cursor-pointer">
              <Plus data-icon="inline-start" /> Tạo hóa đơn
            </Button>
          </>
        }
      />

      {/* Top Section: Filters & Compact Stats */}
      <Card className="gap-0 p-0 flex flex-col">
        <div className="grid grid-cols-2 items-end gap-3 border-b border-border/40 p-3 sm:gap-4 sm:p-4 lg:flex lg:flex-wrap">
          <div className={houses.length <= HOUSE_BUTTON_LIMIT ? 'col-span-2 w-full lg:w-auto' : 'col-span-2 w-full sm:col-span-1 lg:min-w-[200px] lg:flex-1'}>
            {houses.length <= HOUSE_BUTTON_LIMIT ? (
              <>
                <span id="invoice-filter-house-label" className="mb-1.5 block text-xs font-medium text-muted-foreground">Nhà trọ</span>
                <HouseButtonGroup
                  houses={houses}
                  isSelected={id => invoiceFilter.house_id === id}
                  onSelect={handleHouseFilterChange}
                  allLabel="Tất cả"
                  isAllSelected={!invoiceFilter.house_id}
                  onSelectAll={() => handleHouseFilterChange('')}
                />
              </>
            ) : (
              <>
                <label htmlFor="invoice-filter-house" className="mb-1.5 block text-xs font-medium text-muted-foreground">Nhà trọ</label>
                <select
                  id="invoice-filter-house"
                  className={filterControlClass}
                  value={invoiceFilter.house_id || ''}
                  onChange={(e) => handleHouseFilterChange(e.target.value)}
                >
                  <option value="">Tất cả nhà trọ</option>
                  {houses.map(h => (
                    <option key={h.id} value={h.id}>{h.name}</option>
                  ))}
                </select>
              </>
            )}
          </div>

          <div className="w-full lg:w-[160px]">
            <label htmlFor="invoice-filter-room" className="mb-1.5 block text-xs font-medium text-muted-foreground">Phòng</label>
            <select
              id="invoice-filter-room"
              className={filterControlClass}
              value={invoiceFilter.room_id || ''}
              onChange={(e) => setInvoiceFilter({ room_id: e.target.value })}
              disabled={!invoiceFilter.house_id}
            >
              <option value="">Tất cả phòng</option>
              {filterRooms.map(r => (
                <option key={r.id} value={r.id}>{r.name}</option>
              ))}
            </select>
          </div>

          <div className="w-full lg:w-[160px]">
            <label htmlFor="invoice-filter-period" className="mb-1.5 block text-xs font-medium text-muted-foreground">Kỳ hóa đơn</label>
            <input
              id="invoice-filter-period"
              type="month"
              className={filterControlClass}
              value={invoiceFilter.period || ''}
              onChange={(e) => setInvoiceFilter({ period: e.target.value })}
            />
          </div>

          <div className="w-full lg:w-[160px]">
            <label htmlFor="invoice-filter-status" className="mb-1.5 block text-xs font-medium text-muted-foreground">Trạng thái</label>
            <select
              id="invoice-filter-status"
              className={filterControlClass}
              value={invoiceFilter.status || ''}
              onChange={(e) => setInvoiceFilter({ status: e.target.value })}
            >
              <option value="">Tất cả</option>
              <option value="UNPAID">Chưa thanh toán</option>
              <option value="PENDING_VERIFICATION">Chờ xác nhận CK</option>
              <option value="PAID">Đã thanh toán</option>
            </select>
          </div>

          <Button
            variant="ghost"
            size="lg"
            onClick={handleClearFilter}
            disabled={activeFilterCount === 0}
            className="w-full cursor-pointer text-muted-foreground lg:w-auto"
          >
            <FilterX data-icon="inline-start" />
            Xóa lọc{activeFilterCount > 0 ? ` (${activeFilterCount})` : ''}
          </Button>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 gap-3 rounded-b-xl border-t border-border/40 bg-muted/30 p-3 sm:gap-4 sm:p-4 lg:flex lg:items-center lg:gap-8">
          <div className="flex items-center gap-3">
             <div className="size-8 rounded-full bg-muted text-muted-foreground flex items-center justify-center shrink-0">
               <Receipt className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="truncate text-xs font-medium text-muted-foreground">Tổng hóa đơn</p>
               <p className="truncate text-sm font-semibold leading-tight tabular-nums text-foreground sm:text-base">{stats ? stats.totalInvoices.toLocaleString('vi-VN') : '—'}</p>
             </div>
          </div>

          <div className="w-px h-8 bg-border hidden lg:block" />

          <div className="flex items-center gap-3">
             <div className="size-8 rounded-full bg-info/15 text-info flex items-center justify-center shrink-0">
               <TrendingUp className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="truncate text-xs font-medium text-muted-foreground">Tổng dự kiến</p>
               <p className="truncate text-sm font-semibold leading-tight tabular-nums text-foreground sm:text-base">{money(stats?.expectedRevenue)}</p>
             </div>
          </div>

          <div className="w-px h-8 bg-border hidden lg:block" />

          <div className="flex items-center gap-3">
             <div className="size-8 rounded-full bg-success/15 text-success flex items-center justify-center shrink-0">
               <CheckCircle2 className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="truncate text-xs font-medium text-muted-foreground">Đã thu</p>
               <p className="truncate text-sm font-semibold leading-tight tabular-nums text-foreground sm:text-base">{money(stats?.collectedRevenue)}</p>
             </div>
          </div>

          <div className="w-px h-8 bg-border hidden lg:block" />

          <div className="flex items-center gap-3">
             <div className="size-8 rounded-full bg-warning/15 text-warning flex items-center justify-center shrink-0">
               <Clock className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="truncate text-xs font-medium text-muted-foreground">Chưa thu</p>
               <p className="truncate text-sm font-semibold leading-tight tabular-nums text-foreground sm:text-base">{money(stats?.unpaidRevenue)}</p>
             </div>
          </div>
        </div>
      </Card>

      {loadError !== undefined && (
        <div role="alert" className="flex items-center justify-between gap-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          <span className="flex min-w-0 items-center gap-2">
            <AlertTriangle className="size-4 shrink-0" />
            <span className="truncate">{errorText(loadError)}{listQuery.data ? ' Đang hiển thị dữ liệu cũ.' : ''}</span>
          </span>
          <Button variant="outline" size="sm" onClick={() => { listQuery.refetch(); breakdownQuery.refetch() }}>Thử lại</Button>
        </div>
      )}

      {breakdown && breakdown.expected.invoice_count > 0 && stats && (
        <SectionCard icon={Receipt} title="Bóc tách khoản thu">
          <BreakdownTable
            itemLabel="Khoản thu"
            columns={[
              { label: 'Phải thu' },
              { label: 'Đã thu' },
              { label: 'Chưa thu' },
            ]}
            rows={breakdownRows}
            total={{
              label: 'Tổng cộng',
              values: [stats.expectedRevenue, stats.collectedRevenue, stats.unpaidRevenue],
            }}
          />
        </SectionCard>
      )}

      {/* List */}
      {listQuery.isLoading ? (
        <div className="flex flex-col gap-2" aria-busy="true" aria-label="Đang tải hóa đơn">
          {Array.from({ length: 5 }, (_, i) => <Skeleton key={i} className="h-20 rounded-xl" />)}
        </div>
      ) : invoices.length === 0 ? (
        <Card className="p-0">
          <EmptyState icon={Receipt} title="Không tìm thấy hóa đơn nào." className="py-20" />
        </Card>
      ) : (
        <div className={cn('flex flex-col gap-3 transition-opacity', listQuery.isPlaceholder && 'opacity-60')}>
          {/* Mobile Card View */}
          <div className="flex flex-col gap-2 md:hidden">
            <div className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2 text-sm">
              <span className="font-semibold text-foreground">Tổng {total.toLocaleString('vi-VN')} hóa đơn</span>
              <span className="font-semibold tabular-nums text-foreground">{money(stats?.expectedRevenue)}</span>
            </div>
            {invoices.map(invoice => (
              <InvoiceMobileCard
                key={invoice.id}
                invoice={invoice}
                houseName={invoiceFilter.house_id ? undefined : houseNames.get(invoice.house_id) || 'Không rõ nhà trọ'}
                onOpen={openInvoice}
                onEdit={editInvoice}
                onDownload={handleDownload}
                onSendZalo={handleSendZalo}
                onDelete={handleDelete}
              />
            ))}
          </div>

          {/* Desktop Table View */}
          <Card className="hidden gap-0 overflow-hidden p-0 md:flex">
            <Table>
              <TableHeader>
                <TableRow className="bg-secondary/30">
                  <TableHead className="px-6">Kỳ</TableHead>
                  <TableHead className="px-6">Phòng</TableHead>
                  <TableHead className="px-6 text-right">Tổng tiền</TableHead>
                  <TableHead className="px-6 text-center">Trạng thái</TableHead>
                  <TableHead className="px-6">Ngày tạo</TableHead>
                  <TableHead className="px-6 text-center">
                    <div className="flex items-center justify-center gap-2">
                      Hành động
                      <Button
                        onClick={handleDownloadAll}
                        disabled={total === 0 || isDownloadingAll}
                        variant="ghost"
                        size="icon"
                        className="text-muted-foreground"
                        aria-label="Tải tất cả hóa đơn theo bộ lọc"
                        title="Tải tất cả hóa đơn (Mẫu 1)"
                      >
                        {isDownloadingAll ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
                      </Button>
                    </div>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow className="bg-muted/50 hover:bg-muted/50">
                  <TableCell colSpan={2} className="px-6 py-2.5 font-semibold text-foreground">
                    Tổng ({total.toLocaleString('vi-VN')} hóa đơn)
                  </TableCell>
                  <TableCell className="px-6 py-2.5 text-right font-semibold tabular-nums text-foreground">
                    {money(stats?.expectedRevenue)}
                  </TableCell>
                  <TableCell colSpan={3} className="px-6 py-2.5 text-xs text-muted-foreground">
                    {stats ? `Đã thu ${formatCurrency(stats.collectedRevenue)} · Chưa thu ${formatCurrency(stats.unpaidRevenue)}` : null}
                  </TableCell>
                </TableRow>
                {invoices.map(invoice => (
                  <InvoiceRow
                    key={invoice.id}
                    invoice={invoice}
                    houseName={invoiceFilter.house_id ? undefined : houseNames.get(invoice.house_id) || 'Không rõ'}
                    onOpen={openInvoice}
                    onEdit={editInvoice}
                    onDownload={handleDownload}
                    onSendZalo={handleSendZalo}
                    onDelete={handleDelete}
                  />
                ))}
              </TableBody>
            </Table>
          </Card>

          <DataPagination
            page={invoiceFilter.page}
            pageSize={invoiceFilter.limit}
            total={total}
            onPageChange={page => setInvoiceFilter({ page })}
            onPageSizeChange={limit => setInvoiceFilter({ limit })}
          />
        </div>
      )}
    </div>
  )
}

interface InvoiceRowProps {
  invoice: Invoice
  houseName?: string
  onOpen: (invoice: Invoice) => void
  onEdit: (invoice: Invoice) => void
  onDownload: (invoice: Invoice) => void
  onSendZalo: (invoice: Invoice) => void
  onDelete: (invoice: Invoice) => void
}

const InvoiceRow = memo(function InvoiceRow({ invoice, houseName, onOpen, onEdit, onDownload, onSendZalo, onDelete }: InvoiceRowProps) {
  const shortHouse = houseName && houseName.length > 20 ? `${houseName.substring(0, 20)}...` : houseName
  return (
    <TableRow onClick={() => onOpen(invoice)} className="cursor-pointer group">
      <TableCell className="px-6 py-3 font-medium text-foreground">{invoice.period}</TableCell>
      <TableCell className="px-6 py-3 font-medium text-foreground" title={houseName}>
        {houseName ? `${invoice.room_name} (${shortHouse})` : invoice.room_name}
      </TableCell>
      <TableCell className="px-6 py-3 text-right font-semibold tabular-nums text-foreground">
        {formatCurrency(invoice.total_amount)}
      </TableCell>
      <TableCell className="px-6 py-3 text-center">
        <StatusBadge status={invoice.status} />
      </TableCell>
      <TableCell className="px-6 py-3 text-sm text-muted-foreground">
        {new Date(invoice.created_at).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' })}
      </TableCell>
      <TableCell className="px-6 py-3 text-center">
        <div className="flex items-center justify-center gap-1 transition-opacity pointer-fine:opacity-0 pointer-fine:group-hover:opacity-100 pointer-fine:group-focus-within:opacity-100">
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={(e) => { e.stopPropagation(); onEdit(invoice) }}
            className="text-muted-foreground"
            aria-label={`Sửa hóa đơn phòng ${invoice.room_name}`}
            title="Sửa hóa đơn"
          >
            <Pencil className="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={(e) => { e.stopPropagation(); onDownload(invoice) }}
            className="cursor-pointer text-muted-foreground"
            aria-label={`Tải hóa đơn phòng ${invoice.room_name}`}
            title="Tải hóa đơn (Mẫu 1)"
          >
            <Download className="size-4" />
          </Button>
          {invoice.status === 'UNPAID' && (
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={(e) => { e.stopPropagation(); onSendZalo(invoice) }}
              className="cursor-pointer text-muted-foreground"
              aria-label={`Gửi hóa đơn phòng ${invoice.room_name} qua Zalo`}
              title="Gửi qua Zalo"
            >
              <Send className="size-4" />
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={(e) => { e.stopPropagation(); onDelete(invoice) }}
            className="cursor-pointer text-muted-foreground hover:text-destructive"
            aria-label={`Xóa hóa đơn phòng ${invoice.room_name}`}
            title="Xóa hóa đơn"
          >
            <Trash2 className="size-4" />
          </Button>
        </div>
      </TableCell>
    </TableRow>
  )
})
