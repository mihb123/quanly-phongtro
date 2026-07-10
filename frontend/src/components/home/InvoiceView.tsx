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

  const handleDownload = async (e: React.MouseEvent, invoice: Invoice) => {
    e.stopPropagation();
    
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

  const handleDelete = async (e: React.MouseEvent, invoice: Invoice) => {
    e.stopPropagation();
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

  const handleSendZalo = async (e: React.MouseEvent, invoice: Invoice) => {
    e.stopPropagation();
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
    <div className="space-y-6 safe-fade-in">
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
            <Button onClick={() => setShowQuickCreateModal(true)} variant="secondary" className="font-bold touch-target">
              <Zap className="size-4" /> Ghi điện nước nhanh
            </Button>
            <Button onClick={() => setShowCreateModal(true)} className="font-bold touch-target">
              <Plus className="size-4" /> Tạo hóa đơn
            </Button>
          </>
        }
      />

      {/* Top Section: Filters & Compact Stats */}
      <Card className="gap-0 p-0 flex flex-col">
        {/* Filters */}
        <div className="p-4 border-b border-border/40 grid grid-cols-1 sm:grid-cols-2 lg:flex lg:flex-wrap items-end gap-4">
          <div className="w-full lg:flex-1 lg:min-w-[200px]">
            <label className="block text-[10px] font-bold text-muted-foreground uppercase tracking-wider mb-1.5">Nhà trọ</label>
            <select 
              className="w-full h-9 px-3 rounded-lg border border-input focus:border-primary focus:ring focus:ring-primary/20 outline-none transition-all text-sm font-semibold bg-background"
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
            <label className="block text-[10px] font-bold text-muted-foreground uppercase tracking-wider mb-1.5">Phòng</label>
            <select 
              className="w-full h-9 px-3 rounded-lg border border-input focus:border-primary focus:ring focus:ring-primary/20 outline-none transition-all text-sm font-semibold disabled:bg-muted disabled:text-muted-foreground bg-background"
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
            <label className="block text-[10px] font-bold text-muted-foreground uppercase tracking-wider mb-1.5">Kỳ hóa đơn</label>
            <input 
              type="month"
              className="w-full h-9 px-3 rounded-lg border border-input focus:border-primary focus:ring focus:ring-primary/20 outline-none transition-all text-sm font-semibold bg-background"
              value={invoiceFilter.period || ''}
              onChange={(e) => setInvoiceFilter({ period: e.target.value })}
            />
          </div>

          <div className="w-full lg:w-[160px]">
            <label className="block text-[10px] font-bold text-muted-foreground uppercase tracking-wider mb-1.5">Trạng thái</label>
            <select 
              className="w-full h-9 px-3 rounded-lg border border-input focus:border-primary focus:ring focus:ring-primary/20 outline-none transition-all text-sm font-semibold bg-background"
              value={invoiceFilter.status || ''}
              onChange={(e) => setInvoiceFilter({ status: e.target.value })}
            >
              <option value="">Tất cả</option>
              <option value="UNPAID">Chưa thanh toán</option>
              <option value="PENDING_VERIFICATION">Chờ xác nhận CK</option>
              <option value="PAID">Đã thanh toán</option>
            </select>
          </div>

          <Button variant="ghost" onClick={handleClearFilter} className="w-full sm:col-span-2 lg:col-span-1 lg:w-auto h-9 px-3 text-xs font-bold text-muted-foreground hover:text-foreground hover:bg-secondary rounded-lg flex items-center justify-center gap-1.5 transition-all">
            <FilterX className="w-3.5 h-3.5" />
            Xóa lọc
          </Button>
        </div>

        {/* Stats */}
        <div className="p-4 bg-muted/30 grid grid-cols-2 lg:flex lg:items-center gap-4 lg:gap-8 rounded-b-xl border-t border-border/40">
          <div className="flex items-center gap-3">
             <div className="w-8 h-8 rounded-full bg-muted text-muted-foreground flex items-center justify-center shadow-sm shrink-0">
               <Receipt className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider truncate">Tổng hóa đơn</p>
               <p className="text-sm sm:text-base font-extrabold text-foreground leading-tight truncate">{stats.totalInvoices}</p>
             </div>
          </div>

          <div className="w-px h-8 bg-border hidden lg:block" />

          <div className="flex items-center gap-3">
             <div className="w-8 h-8 rounded-full bg-info/15 text-info flex items-center justify-center shadow-sm shrink-0">
               <TrendingUp className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="text-[10px] font-bold text-info/80 uppercase tracking-wider truncate">Tổng dự kiến</p>
               <p className="text-sm sm:text-base font-extrabold text-info leading-tight truncate">{formatCurrency(stats.expectedRevenue)}</p>
             </div>
          </div>

          <div className="w-px h-8 bg-border hidden lg:block" />

          <div className="flex items-center gap-3">
             <div className="w-8 h-8 rounded-full bg-success/15 text-success flex items-center justify-center shadow-sm shrink-0">
               <CheckCircle2 className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="text-[10px] font-bold text-success/80 uppercase tracking-wider truncate">Đã thu</p>
               <p className="text-sm sm:text-base font-extrabold text-success leading-tight truncate">{formatCurrency(stats.collectedRevenue)}</p>
             </div>
          </div>

          <div className="w-px h-8 bg-border hidden lg:block" />

          <div className="flex items-center gap-3">
             <div className="w-8 h-8 rounded-full bg-warning/15 text-warning flex items-center justify-center shadow-sm shrink-0">
               <Clock className="size-4" />
             </div>
             <div className="min-w-0">
               <p className="text-[10px] font-bold text-warning/80 uppercase tracking-wider truncate">Chưa thu</p>
               <p className="text-sm sm:text-base font-extrabold text-warning leading-tight truncate">{formatCurrency(stats.unpaidRevenue)}</p>
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
        <Card className="gap-0 p-0 overflow-hidden">
          {/* Mobile Card View */}
          <div className="block md:hidden divide-y divide-border/40">
            {invoices.map(invoice => (
              <div 
                key={invoice.id} 
                onClick={() => setSelectedInvoiceId(invoice.id)}
                className="p-4 hover:bg-secondary/40 transition-colors cursor-pointer"
              >
                <div className="flex justify-between items-start mb-2">
                  <div>
                    <p className="font-bold text-primary">{(() => {
                        if (invoiceFilter.house_id) return invoice.room_name;
                        const hName = houses.find(h => h.id === invoice.house_id)?.name || 'Không rõ';
                        return `${invoice.room_name} (${hName.length > 20 ? hName.substring(0, 20) + '...' : hName})`;
                      })()}
                    </p>
                    <p className="text-sm font-bold text-foreground mt-0.5">Kỳ: {invoice.period}</p>
                  </div>
                  <div className="text-right">
                    <p className="font-bold text-foreground">{formatCurrency(invoice.total_amount)}</p>
                    <StatusBadge status={invoice.status} className="mt-1" />
                  </div>
                </div>
                <div className="flex justify-between items-center mt-3 pt-3 border-t border-border/40">
                  <span className="text-xs font-medium text-muted-foreground">
                    {new Date(invoice.created_at).toLocaleDateString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric' })}
                  </span>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => { e.stopPropagation(); setSelectedInvoiceId(invoice.id); setShowEditModal(true); }}
                      className="text-muted-foreground hover:text-primary h-8 w-8 p-0 rounded-full touch-target"
                    >
                      <Pencil className="size-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => handleDownload(e, invoice)}
                      className="text-muted-foreground hover:text-info h-8 w-8 p-0 rounded-full touch-target"
                    >
                      <Download className="size-4" />
                    </Button>
                    {invoice.status === 'UNPAID' && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => handleSendZalo(e, invoice)}
                        className="text-muted-foreground hover:text-success h-8 w-8 p-0 rounded-full touch-target"
                      >
                        <Send className="size-4" />
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => handleDelete(e, invoice)}
                      className="text-muted-foreground hover:text-destructive h-8 w-8 p-0 rounded-full touch-target"
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Desktop Table View */}
          <div className="hidden md:block">
            <Table>
              <TableHeader>
                <TableRow className="bg-secondary/30">
                  <TableHead className="px-6 py-4 font-bold text-muted-foreground">Kỳ</TableHead>
                  <TableHead className="px-6 py-4 font-bold text-muted-foreground">Phòng</TableHead>
                  <TableHead className="px-6 py-4 font-bold text-muted-foreground text-right">Tổng tiền</TableHead>
                  <TableHead className="px-6 py-4 font-bold text-muted-foreground text-center">Trạng thái</TableHead>
                  <TableHead className="px-6 py-4 font-bold text-muted-foreground">Ngày tạo</TableHead>
                  <TableHead className="px-6 py-4 font-bold text-muted-foreground text-center">
                    <div className="flex items-center justify-center gap-2">
                      Hành động
                      <Button
                        onClick={handleDownloadAll}
                        disabled={invoices.length === 0 || isDownloadingAll}
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-primary hover:text-primary hover:bg-primary/10 rounded-full touch-target"
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
                    <TableCell className="px-6 py-4 font-bold text-foreground">
                      {invoice.period}
                    </TableCell>
                    <TableCell className="px-6 py-4 font-bold text-primary" title={!invoiceFilter.house_id ? houses.find(h => h.id === invoice.house_id)?.name : undefined}>
                      {(() => {
                        if (invoiceFilter.house_id) return invoice.room_name;
                        const hName = houses.find(h => h.id === invoice.house_id)?.name || 'Không rõ';
                        return `${invoice.room_name} (${hName.length > 20 ? hName.substring(0, 20) + '...' : hName})`;
                      })()}
                    </TableCell>
                    <TableCell className="px-6 py-4 font-bold text-foreground text-right">
                      {formatCurrency(invoice.total_amount)}
                    </TableCell>
                    <TableCell className="px-6 py-4 text-center">
                      <StatusBadge status={invoice.status} />
                    </TableCell>
                    <TableCell className="px-6 py-4 text-sm font-medium text-muted-foreground">
                      {new Date(invoice.created_at).toLocaleDateString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric' })}
                    </TableCell>
                    <TableCell className="px-6 py-4 text-center">
                      <div className="flex items-center justify-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation();
                            setSelectedInvoiceId(invoice.id);
                            setShowEditModal(true);
                          }}
                          className="text-muted-foreground hover:text-primary h-8 w-8 p-0 rounded-full"
                          title="Sửa hóa đơn"
                        >
                          <Pencil className="size-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => handleDownload(e, invoice)}
                          className="text-muted-foreground hover:text-info h-8 w-8 p-0 rounded-full"
                          title="Tải hóa đơn (Mẫu 1)"
                        >
                          <Download className="size-4" />
                        </Button>
                        {invoice.status === 'UNPAID' && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={(e) => handleSendZalo(e, invoice)}
                            className="text-muted-foreground hover:text-success h-8 w-8 p-0 rounded-full"
                            title="Gửi qua Zalo"
                          >
                            <Send className="size-4" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => handleDelete(e, invoice)}
                          className="text-muted-foreground hover:text-destructive h-8 w-8 p-0 rounded-full"
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
          </div>
        </Card>
      )}
    </div>
  )
}
