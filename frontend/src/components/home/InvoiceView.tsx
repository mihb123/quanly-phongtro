import { useState, useEffect, useMemo } from 'react'
import { Plus, Receipt, Zap, TrendingUp, CheckCircle2, Clock, FilterX, Download, Pencil, Loader2, Trash2, Send } from '@/components/icons'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { PageHeader } from '@/components/shared/PageHeader'
import { EmptyState } from '@/components/shared/EmptyState'
import { StatusBadge } from '@/components/shared/StatusBadge'
import { useInvoiceStore } from '@/data/invoiceData'
import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'
import { type Invoice, getInvoiceImageBlob } from '@/api/invoice'
import { sendInvoiceViaZalo } from '@/api/zalo'
import { CreateInvoiceModal } from './modals/CreateInvoiceModal'
import { InvoiceDetailModal } from './modals/InvoiceDetailModal'
import { QuickCreateInvoiceModal } from './modals/QuickCreateInvoiceModal'
import { EditInvoiceModal } from './modals/EditInvoiceModal'
import { BackendImagePreviewModal } from './modals/BackendImagePreviewModal'
import { toast } from 'sonner'
import { formatCurrency } from '@/utils/format'
import { InvoiceMobileCard } from './invoice/InvoiceMobileCard'

// View quản lý hóa đơn: lọc, thống kê nhanh, danh sách (card mobile + table desktop) và các modal tạo/sửa/xem.
export function InvoicesView() {
  const { houses } = useHouseStore()
  const { getRoomsByHouse } = useRoomStore()
  const selectedHouse = useSelectedStore(state => state.selectedHouse)
  const selectHouse = useSelectedStore(state => state.selectHouse)
  
  const { invoices, isLoading, invoiceFilter, setInvoiceFilter, fetchInvoices, deleteInvoice } = useInvoiceStore()

  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showQuickCreateModal, setShowQuickCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [isDownloadingAll, setIsDownloadingAll] = useState(false)
  const [previewData, setPreviewData] = useState<{ url: string, filename: string } | null>(null)
  const [selectedInvoiceId, setSelectedInvoiceId] = useState<string | null>(null)
  const selectedInvoice = invoices.find(inv => inv.id === selectedInvoiceId) || null

  useEffect(() => {
    if (selectedHouse && invoiceFilter.house_id !== selectedHouse.id) {
      setInvoiceFilter({ house_id: selectedHouse.id, room_id: '' })
    } else {
      fetchInvoices()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedHouse])

  const handleClearFilter = () => {
    const currentMonth = `${new Date().getFullYear()}-${String(new Date().getMonth() + 1).padStart(2, '0')}`;
    selectHouse(null);
    setInvoiceFilter({ house_id: '', room_id: '', period: currentMonth, status: '' });
  };

  const [filterRooms, setFilterRooms] = useState<{ id: string; name: string }[]>([])

  // Fetch rooms when a house is selected in filter
  useEffect(() => {
    if (invoiceFilter.house_id) {
      getRoomsByHouse(invoiceFilter.house_id).then(res => setFilterRooms(res)).catch(console.error)
    } else {
      setFilterRooms([])
    }
  }, [invoiceFilter.house_id, getRoomsByHouse])

  const stats = useMemo(() => {
    const totalInvoices = invoices.length;
    let expectedRevenue = 0;
    let collectedRevenue = 0;
    let unpaidRevenue = 0;

    invoices.forEach(inv => {
      expectedRevenue += inv.total_amount;
      if (inv.status === 'PAID') {
        collectedRevenue += inv.total_amount;
      } else {
        unpaidRevenue += inv.total_amount;
      }
    });

    return { totalInvoices, expectedRevenue, collectedRevenue, unpaidRevenue };
  }, [invoices]);

  const handleDownload = async (invoice: Invoice) => {
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
  };

  const handleDelete = async (invoice: Invoice) => {
    if (invoice.status === 'PAID') {
      toast.error('Không thể xóa hóa đơn đã thanh toán', { id: 'delete-invoice' });
      return;
    }
    if (confirm(`Bạn có chắc muốn xóa hóa đơn phòng ${invoice.room_name} kỳ ${invoice.period}?`)) {
      toast.loading('Đang xóa hóa đơn...', { id: 'delete-invoice' });
      const res = await deleteInvoice(invoice.id);
      if (res.success) {
        toast.success(res.message, { id: 'delete-invoice' });
      } else {
        toast.error(res.message, { id: 'delete-invoice' });
      }
    }
  };

  const handleDownloadAll = async () => {
    if (invoices.length === 0) return;
    
    setIsDownloadingAll(true);
    toast.loading(`Đang tải ${invoices.length} ảnh hóa đơn...`, { id: 'download-all' });
    
    try {
      for (let i = 0; i < invoices.length; i++) {
        const invoice = invoices[i];
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
      
      toast.success('Thành công', { id: 'download-all', description: `Đã hoàn tất tải ${invoices.length} ảnh hóa đơn.` });
    } catch (error) {
      console.error('Lỗi khi tải tất cả:', error);
      toast.error('Có lỗi xảy ra', { id: 'download-all', description: 'Không thể tải toàn bộ hóa đơn.' });
    } finally {
      setIsDownloadingAll(false);
    }
  };

  const handleSendZalo = async (invoice: Invoice) => {
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
  };

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
          onClose={() => setSelectedInvoiceId(null)} 
          onEdit={() => setShowEditModal(true)}
        />
      )}
      {showEditModal && selectedInvoice && (
        <EditInvoiceModal
          invoice={selectedInvoice}
          onClose={() => {
            setShowEditModal(false)
            setSelectedInvoiceId(null)
          }}
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
        {/* Filters */}
        <div className="grid grid-cols-2 items-end gap-3 border-b border-border/40 p-3 sm:gap-4 sm:p-4 lg:flex lg:flex-wrap">
          <div className="col-span-2 w-full sm:col-span-1 lg:min-w-[200px] lg:flex-1">
            <label className="mb-1.5 block text-xs font-medium text-muted-foreground">Nhà trọ</label>
            <select 
              className="h-11 w-full rounded-md border border-input bg-background px-3 text-base shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 sm:h-9 sm:text-sm"
              value={invoiceFilter.house_id || ''}
              onChange={(e) => {
                const newHouseId = e.target.value;
                setInvoiceFilter({ house_id: newHouseId, room_id: '' });
                const house = houses.find(h => h.id === newHouseId) || null;
                selectHouse(house);
              }}
            >
              <option value="">Tất cả nhà trọ</option>
              {houses.map(h => (
                <option key={h.id} value={h.id}>{h.name}</option>
              ))}
            </select>
          </div>

          <div className="w-full lg:w-[160px]">
            <label className="mb-1.5 block text-xs font-medium text-muted-foreground">Phòng</label>
            <select 
              className="h-11 w-full rounded-md border border-input bg-background px-3 text-base shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground sm:h-9 sm:text-sm"
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
            <label className="mb-1.5 block text-xs font-medium text-muted-foreground">Kỳ hóa đơn</label>
            <input 
              type="month"
              className="h-11 w-full rounded-md border border-input bg-background px-3 text-base shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 sm:h-9 sm:text-sm"
              value={invoiceFilter.period || ''}
              onChange={(e) => setInvoiceFilter({ period: e.target.value })}
            />
          </div>

          <div className="w-full lg:w-[160px]">
            <label className="mb-1.5 block text-xs font-medium text-muted-foreground">Trạng thái</label>
            <select 
              className="h-11 w-full rounded-md border border-input bg-background px-3 text-base shadow-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 sm:h-9 sm:text-sm"
              value={invoiceFilter.status || ''}
              onChange={(e) => setInvoiceFilter({ status: e.target.value })}
            >
              <option value="">Tất cả</option>
              <option value="UNPAID">Chưa thanh toán</option>
              <option value="PENDING_VERIFICATION">Chờ xác nhận CK</option>
              <option value="PAID">Đã thanh toán</option>
            </select>
          </div>

          <Button variant="ghost" size="lg" onClick={handleClearFilter} className="w-full cursor-pointer text-muted-foreground lg:w-auto">
            <FilterX data-icon="inline-start" />
            Xóa lọc
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
               <p className="truncate text-sm font-semibold leading-tight tabular-nums text-foreground sm:text-base">{stats.totalInvoices}</p>
             </div>
          </div>

          <div className="w-px h-8 bg-border hidden lg:block" />

          <div className="flex items-center gap-3">
             <div className="size-8 rounded-full bg-info/15 text-info flex items-center justify-center shrink-0">
               <TrendingUp className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="truncate text-xs font-medium text-muted-foreground">Tổng dự kiến</p>
               <p className="truncate text-sm font-semibold leading-tight tabular-nums text-foreground sm:text-base">{formatCurrency(stats.expectedRevenue)}</p>
             </div>
          </div>

          <div className="w-px h-8 bg-border hidden lg:block" />

          <div className="flex items-center gap-3">
             <div className="size-8 rounded-full bg-success/15 text-success flex items-center justify-center shrink-0">
               <CheckCircle2 className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="truncate text-xs font-medium text-muted-foreground">Đã thu</p>
               <p className="truncate text-sm font-semibold leading-tight tabular-nums text-foreground sm:text-base">{formatCurrency(stats.collectedRevenue)}</p>
             </div>
          </div>

          <div className="w-px h-8 bg-border hidden lg:block" />

          <div className="flex items-center gap-3">
             <div className="size-8 rounded-full bg-warning/15 text-warning flex items-center justify-center shrink-0">
               <Clock className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="truncate text-xs font-medium text-muted-foreground">Chưa thu</p>
               <p className="truncate text-sm font-semibold leading-tight tabular-nums text-foreground sm:text-base">{formatCurrency(stats.unpaidRevenue)}</p>
             </div>
          </div>
        </div>
      </Card>
      
      {/* List */}
      {isLoading ? (
        <div className="text-center py-20">
          <Loader2 className="size-8 text-primary animate-spin mx-auto" />
          <p className="text-muted-foreground mt-4 font-medium">Đang tải hóa đơn...</p>
        </div>
      ) : invoices.length === 0 ? (
        <Card className="p-0">
          <EmptyState icon={Receipt} title="Không tìm thấy hóa đơn nào." className="py-20" />
        </Card>
      ) : (
        <>
          {/* Mobile Card View */}
          <div className="flex flex-col gap-2 md:hidden">
            {invoices.map(invoice => {
              const houseName = invoiceFilter.house_id
                ? undefined
                : houses.find(house => house.id === invoice.house_id)?.name || 'Không rõ nhà trọ'

              return (
                <InvoiceMobileCard
                  key={invoice.id}
                  invoice={invoice}
                  houseName={houseName}
                  onOpen={() => setSelectedInvoiceId(invoice.id)}
                  onEdit={() => {
                    setSelectedInvoiceId(invoice.id)
                    setShowEditModal(true)
                  }}
                  onDownload={() => handleDownload(invoice)}
                  onSendZalo={() => handleSendZalo(invoice)}
                  onDelete={() => handleDelete(invoice)}
                />
              )
            })}
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
                        disabled={invoices.length === 0 || isDownloadingAll}
                        variant="ghost"
                        size="icon"
                        className="text-muted-foreground"
                        title="Tải tất cả hóa đơn (Mẫu 1)"
                      >
                        {isDownloadingAll ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
                      </Button>
                    </div>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {invoices.map(invoice => (
                  <TableRow
                    key={invoice.id}
                    onClick={() => setSelectedInvoiceId(invoice.id)}
                    className="cursor-pointer group"
                  >
                    <TableCell className="px-6 py-3 font-medium text-foreground">
                      {invoice.period}
                    </TableCell>
                    <TableCell className="px-6 py-3 font-medium text-foreground" title={!invoiceFilter.house_id ? houses.find(h => h.id === invoice.house_id)?.name : undefined}>
                      {(() => {
                        if (invoiceFilter.house_id) return invoice.room_name;
                        const hName = houses.find(h => h.id === invoice.house_id)?.name || 'Không rõ';
                        return `${invoice.room_name} (${hName.length > 20 ? hName.substring(0, 20) + '...' : hName})`;
                      })()}
                    </TableCell>
                    <TableCell className="px-6 py-3 text-right font-semibold tabular-nums text-foreground">
                      {formatCurrency(invoice.total_amount)}
                    </TableCell>
                    <TableCell className="px-6 py-3 text-center">
                      <StatusBadge status={invoice.status} />
                    </TableCell>
                    <TableCell className="px-6 py-3 text-sm text-muted-foreground">
                      {new Date(invoice.created_at).toLocaleDateString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric' })}
                    </TableCell>
                    <TableCell className="px-6 py-3 text-center">
                      <div className="flex items-center justify-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={(e) => {
                            e.stopPropagation();
                            setSelectedInvoiceId(invoice.id);
                            setShowEditModal(true);
                          }}
                          className="text-muted-foreground"
                          title="Sửa hóa đơn"
                        >
                          <Pencil className="size-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={(e) => {
                            e.stopPropagation()
                            handleDownload(invoice)
                          }}
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
                            onClick={(e) => {
                              e.stopPropagation()
                              handleSendZalo(invoice)
                            }}
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
                          onClick={(e) => {
                            e.stopPropagation()
                            handleDelete(invoice)
                          }}
                          className="cursor-pointer text-muted-foreground hover:text-destructive"
                          aria-label={`Xóa hóa đơn phòng ${invoice.room_name}`}
                          title="Xóa hóa đơn"
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>
        </>
      )}
    </div>
  )
}
