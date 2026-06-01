import { useState, useEffect, useMemo } from 'react'
import { Plus, Receipt, Zap, TrendingUp, CheckCircle2, Clock, FilterX, Download, Pencil, Loader2, Trash2 } from 'lucide-react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useInvoiceStore } from '@/data/invoiceData'
import { useHouseStore } from '@/data/houseData'
import { useRoomStore } from '@/data/roomData'
import { useSelectedStore } from '@/data/selectedData'
import { type Invoice } from '@/api/invoice'
import { CreateInvoiceModal } from './modals/CreateInvoiceModal'
import { InvoiceDetailModal } from './modals/InvoiceDetailModal'
import { QuickCreateInvoiceModal } from './modals/QuickCreateInvoiceModal'
import { EditInvoiceModal } from './modals/EditInvoiceModal'
import { BackendImagePreviewModal } from './modals/BackendImagePreviewModal'
import { toast } from 'sonner'
import { apiClient } from '@/api/client'

const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
}

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
      
      const response = await apiClient.get(`/invoice/${invoice.id}/image`, {
        responseType: 'blob', // Bắt buộc để tải file nhị phân (ảnh)
      });

      const houseNameRaw = houses.find(h => h.id === invoice.house_id)?.name || 'NhaTro';
      const houseName = houseNameRaw.replace(/\s+/g, '');

      const blob = new Blob([response.data], { type: 'image/png' });
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
          const response = await apiClient.get(`/invoice/${invoice.id}/image`, {
            responseType: 'blob',
          });
          
          const houseNameRaw = houses.find(h => h.id === invoice.house_id)?.name || 'NhaTro';
          const houseName = houseNameRaw.replace(/\s+/g, '');
          const periodStr = invoice.period.replace('-', '_');
          
          const blob = new Blob([response.data], { type: 'image/png' });
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

  return (
    <div className="space-y-6 animate-in fade-in zoom-in duration-300">
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
      <div className="flex flex-col md:flex-row md:justify-between md:items-end gap-4 border-b border-slate-200 pb-6">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-slate-800">Quản lý hóa đơn</h1>
          <p className="text-slate-500 mt-1">Danh sách hóa đơn điện, nước, dịch vụ hàng tháng</p>
        </div>
        <div className="flex items-center gap-3">
           <Button onClick={() => setShowQuickCreateModal(true)} className="bg-blue-600 hover:bg-blue-700 shadow-md text-white rounded-xl cursor-pointer h-10 px-6 font-bold transition-all active:scale-95 flex items-center gap-2">
             <Zap className="w-4 h-4" /> Ghi điện nước nhanh
           </Button>
           <Button onClick={() => setShowCreateModal(true)} className="bg-purple-600 hover:bg-purple-700 shadow-md text-white rounded-xl cursor-pointer h-10 px-6 font-bold transition-all active:scale-95 flex items-center gap-2">
             <Plus className="w-4 h-4" /> Tạo hóa đơn
           </Button>
        </div>
      </div>

      {/* Top Section: Filters & Compact Stats */}
      <Card className="bg-white/80 border-slate-200/60 shadow-sm flex flex-col">
        {/* Filters */}
        <div className="p-4 border-b border-slate-100 flex flex-wrap items-end gap-4">
          <div className="flex-1 min-w-[200px]">
            <label className="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Nhà trọ</label>
            <select 
              className="w-full h-9 px-3 rounded-lg border border-slate-200 focus:border-purple-500 focus:ring focus:ring-purple-200 outline-none transition-all text-sm font-semibold bg-white"
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

          <div className="w-[140px] md:w-[160px]">
            <label className="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Phòng</label>
            <select 
              className="w-full h-9 px-3 rounded-lg border border-slate-200 focus:border-purple-500 focus:ring focus:ring-purple-200 outline-none transition-all text-sm font-semibold disabled:bg-slate-50 disabled:text-slate-400 bg-white"
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

          <div className="w-[140px] md:w-[160px]">
            <label className="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Kỳ hóa đơn</label>
            <input 
              type="month"
              className="w-full h-9 px-3 rounded-lg border border-slate-200 focus:border-purple-500 focus:ring focus:ring-purple-200 outline-none transition-all text-sm font-semibold bg-white"
              value={invoiceFilter.period || ''}
              onChange={(e) => setInvoiceFilter({ period: e.target.value })}
            />
          </div>

          <div className="w-[140px] md:w-[160px]">
            <label className="block text-[10px] font-bold text-slate-500 uppercase tracking-wider mb-1.5">Trạng thái</label>
            <select 
              className="w-full h-9 px-3 rounded-lg border border-slate-200 focus:border-purple-500 focus:ring focus:ring-purple-200 outline-none transition-all text-sm font-semibold bg-white"
              value={invoiceFilter.status || ''}
              onChange={(e) => setInvoiceFilter({ status: e.target.value })}
            >
              <option value="">Tất cả</option>
              <option value="UNPAID">Chưa thanh toán</option>
              <option value="PAID">Đã thanh toán</option>
            </select>
          </div>

          <Button variant="ghost" onClick={handleClearFilter} className="h-9 px-3 text-xs font-bold text-slate-500 hover:text-slate-800 hover:bg-slate-100 rounded-lg flex items-center gap-1.5 transition-all ml-auto lg:ml-0">
            <FilterX className="w-3.5 h-3.5" />
            Xóa lọc
          </Button>
        </div>

        {/* Stats */}
        <div className="p-4 bg-slate-50/50 flex flex-wrap items-center gap-x-8 gap-y-4 rounded-b-xl border-t border-slate-100/50">
          <div className="flex items-center gap-3">
             <div className="w-8 h-8 rounded-full bg-slate-100 border border-slate-200/60 flex items-center justify-center shadow-sm">
               <Receipt className="w-4 h-4 text-slate-600" />
             </div>
             <div>
               <p className="text-[10px] font-bold text-slate-500 uppercase tracking-wider">Tổng hóa đơn</p>
               <p className="text-base font-extrabold text-slate-800 leading-tight">{stats.totalInvoices}</p>
             </div>
          </div>
          
          <div className="w-px h-8 bg-slate-200 hidden sm:block" />
          
          <div className="flex items-center gap-3">
             <div className="w-8 h-8 rounded-full bg-blue-100 border border-blue-200/60 flex items-center justify-center shadow-sm">
               <TrendingUp className="w-4 h-4 text-blue-600" />
             </div>
             <div>
               <p className="text-[10px] font-bold text-blue-600/80 uppercase tracking-wider">Tổng dự kiến</p>
               <p className="text-base font-extrabold text-blue-700 leading-tight">{formatCurrency(stats.expectedRevenue)}</p>
             </div>
          </div>
          
          <div className="w-px h-8 bg-slate-200 hidden sm:block" />
          
          <div className="flex items-center gap-3">
             <div className="w-8 h-8 rounded-full bg-green-100 border border-green-200/60 flex items-center justify-center shadow-sm">
               <CheckCircle2 className="w-4 h-4 text-green-600" />
             </div>
             <div>
               <p className="text-[10px] font-bold text-green-600/80 uppercase tracking-wider">Đã thu</p>
               <p className="text-base font-extrabold text-green-700 leading-tight">{formatCurrency(stats.collectedRevenue)}</p>
             </div>
          </div>
          
          <div className="w-px h-8 bg-slate-200 hidden sm:block" />
          
          <div className="flex items-center gap-3">
             <div className="w-8 h-8 rounded-full bg-orange-100 border border-orange-200/60 flex items-center justify-center shadow-sm">
               <Clock className="w-4 h-4 text-orange-600" />
             </div>
             <div>
               <p className="text-[10px] font-bold text-orange-600/80 uppercase tracking-wider">Chưa thu</p>
               <p className="text-base font-extrabold text-orange-700 leading-tight">{formatCurrency(stats.unpaidRevenue)}</p>
             </div>
          </div>
        </div>
      </Card>
      
      {/* List */}
      {isLoading ? (
        <div className="text-center py-20">
          <div className="w-8 h-8 border-4 border-purple-200 border-t-purple-600 rounded-full animate-spin mx-auto"></div>
          <p className="text-slate-500 mt-4 font-medium">Đang tải hóa đơn...</p>
        </div>
      ) : invoices.length === 0 ? (
        <div className="text-center py-20">
          <Receipt className="w-12 h-12 text-slate-300 mx-auto mb-3" />
          <p className="text-slate-500 italic">Không tìm thấy hóa đơn nào.</p>
        </div>
      ) : (
        <div className="bg-white/80 rounded-2xl border border-slate-200/60 shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-slate-50 border-b border-slate-100">
                  <th className="py-4 px-6 font-bold text-slate-600 text-sm whitespace-nowrap">Kỳ</th>
                  <th className="py-4 px-6 font-bold text-slate-600 text-sm whitespace-nowrap">Phòng</th>
                  <th className="py-4 px-6 font-bold text-slate-600 text-sm whitespace-nowrap text-right">Tổng tiền</th>
                  <th className="py-4 px-6 font-bold text-slate-600 text-sm whitespace-nowrap text-center">Trạng thái</th>
                  <th className="py-4 px-6 font-bold text-slate-600 text-sm whitespace-nowrap">Ngày tạo</th>
                  <th className="py-4 px-6 font-bold text-slate-600 text-sm whitespace-nowrap text-center">
                    <div className="flex items-center justify-center gap-2">
                      Hành động
                      <Button 
                        onClick={handleDownloadAll}
                        disabled={invoices.length === 0 || isDownloadingAll}
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-purple-600 hover:text-purple-700 hover:bg-purple-100 rounded-full"
                        title="Tải tất cả hóa đơn (Mẫu 1)"
                      >
                        {isDownloadingAll ? <Loader2 className="w-4 h-4 animate-spin" /> : <Download className="w-4 h-4" />}
                      </Button>
                    </div>
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {invoices.map(invoice => (
                  <tr 
                    key={invoice.id} 
                    onClick={() => setSelectedInvoiceId(invoice.id)}
                    className="hover:bg-slate-50 transition-colors cursor-pointer group"
                  >
                    <td className="py-4 px-6 font-bold text-slate-800 whitespace-nowrap">
                      {invoice.period}
                    </td>
                    <td className="py-4 px-6 font-bold text-purple-600 whitespace-nowrap" title={!invoiceFilter.house_id ? houses.find(h => h.id === invoice.house_id)?.name : undefined}>
                      {(() => {
                        if (invoiceFilter.house_id) return invoice.room_name;
                        const hName = houses.find(h => h.id === invoice.house_id)?.name || 'Không rõ';
                        return `${invoice.room_name} (${hName.length > 20 ? hName.substring(0, 20) + '...' : hName})`;
                      })()}
                    </td>
                    <td className="py-4 px-6 font-bold text-slate-800 text-right whitespace-nowrap">
                      {formatCurrency(invoice.total_amount)}
                    </td>
                    <td className="py-4 px-6 text-center whitespace-nowrap">
                      <span className={`text-xs font-bold px-3 py-1 rounded-full border ${
                        invoice.status === 'PAID' 
                          ? 'bg-green-100 text-green-700 border-green-200' 
                          : 'bg-red-100 text-red-700 border-red-200'
                      }`}>
                        {invoice.status === 'PAID' ? 'Đã thu' : 'Chưa thu'}
                      </span>
                    </td>
                    <td className="py-4 px-6 text-sm font-medium text-slate-500 whitespace-nowrap">
                      {new Date(invoice.created_at).toLocaleDateString('en-GB', { day: '2-digit', month: '2-digit', year: 'numeric' })}
                    </td>
                    <td className="py-4 px-6 text-center whitespace-nowrap">
                      <div className="flex items-center justify-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button 
                          variant="ghost" 
                          size="sm" 
                          onClick={(e) => {
                            e.stopPropagation();
                            setSelectedInvoiceId(invoice.id);
                            setShowEditModal(true);
                          }}
                          className="text-slate-500 hover:text-purple-600 hover:bg-purple-50 cursor-pointer h-8 w-8 p-0 rounded-full"
                          title="Sửa hóa đơn"
                        >
                          <Pencil className="w-4 h-4" />
                        </Button>
                        <Button 
                          variant="ghost" 
                          size="sm" 
                          onClick={(e) => handleDownload(e, invoice)}
                          className="text-slate-500 hover:text-blue-600 hover:bg-blue-50 cursor-pointer h-8 w-8 p-0 rounded-full"
                          title="Tải hóa đơn (Mẫu 1)"
                        >
                          <Download className="w-4 h-4" />
                        </Button>
                        <Button 
                          variant="ghost" 
                          size="sm" 
                          onClick={(e) => handleDelete(e, invoice)}
                          className="text-slate-500 hover:text-red-600 hover:bg-red-50 cursor-pointer h-8 w-8 p-0 rounded-full"
                          title="Xóa hóa đơn"
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
