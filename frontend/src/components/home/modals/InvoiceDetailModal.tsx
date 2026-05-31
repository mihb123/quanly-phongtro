import { useState, useCallback, useEffect } from 'react'
import { X, Receipt, Loader2, CheckCircle, Zap, Droplets, Download } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useInvoiceStore } from '@/data/invoiceData'
import type { Invoice } from '@/api/invoice'
import { apiClient } from '@/api/client'

const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
}

interface Props {
  invoice: Invoice
  onClose: () => void
  onEdit?: () => void
}

export function InvoiceDetailModal({ invoice, onClose, onEdit }: Props) {
  const { payInvoice, unpayInvoice } = useInvoiceStore()
  const [isPaying, setIsPaying] = useState(false)
  const [isDownloading, setIsDownloading] = useState(false)

  // Handle Escape key
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose()
  }, [onClose])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  const handlePay = async () => {
    setIsPaying(true)
    const res = await payInvoice(invoice.id)
    setIsPaying(false)
    if (!res.success) {
      alert(res.message)
    }
  }

  const handleUnpay = async () => {
    setIsPaying(true)
    const res = await unpayInvoice(invoice.id)
    setIsPaying(false)
    if (!res.success) {
      alert(res.message)
    }
  }

  const handleDownload = async () => {
    try {
      setIsDownloading(true)
      const res = await apiClient.get(`/invoice/${invoice.id}/download`, {
        responseType: 'blob'
      })
      const url = window.URL.createObjectURL(new Blob([res.data]))
      const link = document.createElement('a')
      link.href = url
      link.setAttribute('download', `phongtro_hoadon_${invoice.room_name}_${invoice.period.replace('/', '_')}.png`)
      document.body.appendChild(link)
      link.click()
      link.remove()
    } catch (error) {
      console.error('Failed to download invoice:', error)
      alert('Không thể tải xuống hóa đơn')
    } finally {
      setIsDownloading(false)
    }
  }

  // Calculate dynamic usages and unit prices
  const elecUsage = invoice.new_electricity_index - invoice.old_electricity_index;
  const elecUnitPrice = elecUsage > 0 ? invoice.electricity_fee / elecUsage : 0;
  
  const waterUsage = invoice.new_water_index - invoice.old_water_index;
  const waterUnitPrice = waterUsage > 0 ? invoice.water_fee / waterUsage : 0;
  
  const showUtilityTable = invoice.electricity_fee > 0 || invoice.water_fee > 0;

  // Build statement lines
  const lines = []
  if (invoice.room_fee > 0) {
    lines.push({ label: 'Tiền thuê phòng', desc: 'Giá thuê phòng cố định', value: invoice.room_fee, isDiscount: false })
  }
  if (invoice.electricity_fee > 0) {
    lines.push({ label: 'Tiền điện', desc: elecUsage > 0 ? `Tiêu thụ: ${elecUsage} kWh (Xem bảng chỉ số)` : 'Định mức điện cố định', value: invoice.electricity_fee, isDiscount: false })
  }
  if (invoice.water_fee > 0) {
    lines.push({ label: 'Tiền nước', desc: waterUsage > 0 ? `Tiêu thụ: ${waterUsage} m³ (Xem bảng chỉ số)` : 'Định mức nước cố định', value: invoice.water_fee, isDiscount: false })
  }
  if (invoice.wifi_fee > 0) {
    lines.push({ label: 'Tiền mạng Wifi', desc: 'Trọn gói / phòng', value: invoice.wifi_fee, isDiscount: false })
  }
  if (invoice.parking_fee > 0) {
    lines.push({ label: 'Tiền gửi xe', desc: `Gửi ${invoice.vehicle_count} xe máy`, value: invoice.parking_fee, isDiscount: false })
  }
  if (invoice.service_fee > 0) {
    lines.push({ label: 'Phí dịch vụ chung', desc: 'Vệ sinh, rác, thang máy', value: invoice.service_fee, isDiscount: false })
  }
  if (invoice.extra_person_fee > 0) {
    lines.push({ label: 'Phụ thu người thêm', desc: `${invoice.tenant_count - invoice.extra_person_threshold} người vượt đ.mức (${formatCurrency(invoice.extra_person_fee_unit)}/ng)`, value: invoice.extra_person_fee, isDiscount: false })
  }
  if (invoice.extra_vehicle_fee > 0) {
    lines.push({ label: 'Phụ thu xe thêm', desc: `${invoice.vehicle_count - invoice.extra_vehicle_threshold} xe vượt đ.mức (${formatCurrency(invoice.extra_vehicle_fee_unit)}/xe)`, value: invoice.extra_vehicle_fee, isDiscount: false })
  }
  if (invoice.other_fee > 0) {
    lines.push({ label: 'Chi phí phát sinh', desc: 'Chi phí khác phát sinh', value: invoice.other_fee, isDiscount: false })
  }
  if (invoice.discount > 0) {
    lines.push({ label: 'Giảm trừ khuyến mại', desc: 'Khấu trừ đặc biệt', value: invoice.discount, isDiscount: true })
  }

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-0"
      onClick={onClose}
    >
      <div className="absolute inset-0 bg-slate-900/60 backdrop-blur-sm transition-opacity" />
      
      <div 
        className="relative bg-white rounded-3xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col animate-in fade-in zoom-in-95 duration-200"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex justify-between items-center p-6 border-b border-slate-100 bg-slate-50/80 sticky top-0 z-10 backdrop-blur-md">
          <div className="flex items-center gap-4">
            <div className="w-12 h-12 rounded-2xl bg-indigo-100 text-indigo-600 flex items-center justify-center shadow-inner">
              <Receipt className="w-6 h-6" />
            </div>
            <div>
              <h2 className="text-2xl font-black text-slate-800 tracking-tight">Phòng {invoice.room_name}</h2>
              <div className="flex items-center gap-3 mt-1">
                <p className="text-sm font-semibold text-slate-500">Kỳ: {invoice.period}</p>
                <span className={`px-2.5 py-0.5 rounded-full text-xs font-bold border ${
                  invoice.status === 'PAID' 
                    ? 'bg-green-100 text-green-700 border-green-200' 
                    : 'bg-red-100 text-red-700 border-red-200'
                }`}>
                  {invoice.status === 'PAID' ? 'Đã thanh toán' : 'Chưa thanh toán'}
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button 
              onClick={handleDownload}
              disabled={isDownloading}
              title="Tải ảnh hóa đơn"
              className="w-10 h-10 flex items-center justify-center rounded-full bg-indigo-50 text-indigo-600 hover:bg-indigo-100 hover:text-indigo-700 transition-colors cursor-pointer disabled:opacity-50"
            >
              {isDownloading ? <Loader2 className="w-5 h-5 animate-spin" /> : <Download className="w-5 h-5" />}
            </button>
            <button 
              onClick={onClose}
              className="w-10 h-10 flex items-center justify-center rounded-full bg-slate-100 text-slate-500 hover:bg-slate-200 hover:text-slate-700 transition-colors cursor-pointer"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto space-y-8 flex-1">
          
          {/* Table 1: Utility Statement */}
          {showUtilityTable && (
            <div className="space-y-4">
              <h3 className="text-sm font-bold text-indigo-600 flex items-center gap-2 uppercase tracking-wider">
                <span className="w-6 h-6 rounded-full bg-indigo-100 flex items-center justify-center text-xs">1</span>
                Bảng kê chỉ số Điện / Nước
              </h3>
              
              <div className="border border-slate-200 rounded-2xl overflow-hidden bg-white shadow-sm">
                <table className="w-full text-sm text-left">
                  <thead className="bg-slate-50 text-slate-500 font-bold border-b border-slate-200">
                    <tr>
                      <th className="px-4 py-3">Dịch vụ</th>
                      <th className="px-4 py-3 text-center">Chỉ số cũ</th>
                      <th className="px-4 py-3 text-center">Chỉ số mới</th>
                      <th className="px-4 py-3 text-center">Tiêu thụ</th>
                      <th className="px-4 py-3 text-right">Đơn giá</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {invoice.electricity_fee > 0 && elecUsage > 0 && (
                      <tr className="hover:bg-slate-50/50 transition-colors">
                        <td className="px-4 py-3 font-semibold text-slate-700 flex items-center gap-2">
                          <Zap className="w-4 h-4 text-yellow-500" />
                          Điện
                        </td>
                        <td className="px-4 py-3 text-center text-slate-600 font-medium">{invoice.old_electricity_index}</td>
                        <td className="px-4 py-3 text-center text-slate-600 font-medium">{invoice.new_electricity_index}</td>
                        <td className="px-4 py-3 text-center font-bold text-slate-800">{elecUsage} kWh</td>
                        <td className="px-4 py-3 text-right font-medium text-slate-600">{formatCurrency(elecUnitPrice)}</td>
                      </tr>
                    )}
                    {invoice.water_fee > 0 && waterUsage > 0 && (
                      <tr className="hover:bg-slate-50/50 transition-colors">
                        <td className="px-4 py-3 font-semibold text-slate-700 flex items-center gap-2">
                          <Droplets className="w-4 h-4 text-blue-500" />
                          Nước
                        </td>
                        <td className="px-4 py-3 text-center text-slate-600 font-medium">{invoice.old_water_index}</td>
                        <td className="px-4 py-3 text-center text-slate-600 font-medium">{invoice.new_water_index}</td>
                        <td className="px-4 py-3 text-center font-bold text-slate-800">{waterUsage} m³</td>
                        <td className="px-4 py-3 text-right font-medium text-slate-600">{formatCurrency(waterUnitPrice)}</td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Table 2: Services Detail */}
          <div className="space-y-4">
            <h3 className="text-sm font-bold text-indigo-600 flex items-center gap-2 uppercase tracking-wider">
              <span className="w-6 h-6 rounded-full bg-indigo-100 flex items-center justify-center text-xs">{showUtilityTable ? '2' : '1'}</span>
              Chi tiết các chi phí dịch vụ
            </h3>

            <div className="border border-slate-200 rounded-2xl overflow-hidden bg-white shadow-sm">
              <table className="w-full text-sm text-left">
                <thead className="bg-slate-50 text-slate-500 font-bold border-b border-slate-200">
                  <tr>
                    <th className="px-4 py-3">Khoản mục dịch vụ</th>
                    <th className="px-4 py-3 hidden sm:table-cell">Mô tả chi tiết</th>
                    <th className="px-4 py-3 text-right">Thành tiền</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {lines.map((line, i) => (
                    <tr key={i} className="hover:bg-slate-50/50 transition-colors">
                      <td className="px-4 py-3.5">
                        <div className="font-bold text-slate-800">{line.label}</div>
                        <div className="text-xs text-slate-500 mt-0.5 sm:hidden">{line.desc}</div>
                      </td>
                      <td className="px-4 py-3.5 text-slate-500 font-medium hidden sm:table-cell">{line.desc}</td>
                      <td className={`px-4 py-3.5 text-right font-bold whitespace-nowrap ${line.isDiscount ? 'text-red-600' : 'text-slate-800'}`}>
                        {line.isDiscount ? '-' : ''}{formatCurrency(line.value)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
          
        </div>

        {/* Footer (Total and Actions) */}
        <div className="p-6 bg-slate-50 border-t border-slate-100 space-y-4 rounded-b-3xl mt-auto">
          <div className="flex justify-between items-center bg-indigo-50/50 p-4 rounded-2xl border border-indigo-100">
            <span className="font-black text-indigo-900">TỔNG TIỀN CẦN THANH TOÁN</span>
            <span className="text-2xl font-black text-indigo-600">{formatCurrency(invoice.total_amount)}</span>
          </div>

          <div className="flex gap-3">
            {invoice.status === 'UNPAID' && (
              <>
                {onEdit && (
                  <Button 
                    onClick={onEdit}
                    variant="outline"
                    className="flex-1 h-12 border-slate-200 hover:bg-white text-slate-700 font-bold text-base rounded-xl transition-all cursor-pointer"
                  >
                    Sửa thông tin
                  </Button>
                )}
                <Button 
                  onClick={handlePay}
                  disabled={isPaying}
                  className="flex-[2] h-12 bg-green-600 hover:bg-green-700 text-white font-bold text-base rounded-xl transition-all shadow-sm shadow-green-600/20 cursor-pointer"
                >
                  {isPaying ? (
                    <Loader2 className="w-5 h-5 animate-spin" />
                  ) : (
                    <>
                      <CheckCircle className="w-5 h-5 mr-2" />
                      Đánh dấu đã thanh toán
                    </>
                  )}
                </Button>
              </>
            )}

            {invoice.status === 'PAID' && (
              <Button 
                onClick={handleUnpay}
                disabled={isPaying}
                variant="outline"
                className="w-full h-12 border-orange-200 hover:bg-orange-50 text-orange-700 font-bold text-base rounded-xl transition-all cursor-pointer"
              >
                {isPaying ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <>
                    <X className="w-5 h-5 mr-2" />
                    Hoàn tác thanh toán
                  </>
                )}
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
