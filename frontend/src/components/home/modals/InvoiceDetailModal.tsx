import { useState, useCallback, useEffect } from 'react'
import { X, Receipt, Loader2, CheckCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useInvoiceStore } from '@/data/invoiceData'
import type { Invoice } from '@/api/invoice'

const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
}

interface Props {
  invoice: Invoice
  onClose: () => void
  onEdit?: () => void
}

export function InvoiceDetailModal({ invoice, onClose, onEdit }: Props) {
  const { payInvoice } = useInvoiceStore()
  const [isPaying, setIsPaying] = useState(false)

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

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-0"
      onClick={onClose}
    >
      <div className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm transition-opacity" />
      
      <div 
        className="relative bg-white rounded-3xl shadow-2xl w-full max-w-md overflow-hidden animate-in fade-in zoom-in-95 duration-200"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex justify-between items-center p-6 border-b border-slate-100 bg-slate-50/50">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-purple-100 text-purple-600 flex items-center justify-center">
              <Receipt className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-xl font-extrabold text-slate-800">Phòng {invoice.room_name}</h2>
              <p className="text-sm font-medium text-slate-500 mt-1">Kỳ hóa đơn: {invoice.period}</p>
            </div>
          </div>
          <button 
            onClick={onClose}
            className="w-8 h-8 flex items-center justify-center rounded-full bg-slate-100 text-slate-500 hover:bg-slate-200 hover:text-slate-700 transition-colors cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          {/* Status Badge */}
          <div className="flex justify-center mb-6">
            <span className={`px-4 py-1.5 rounded-full text-sm font-bold border ${
              invoice.status === 'PAID' 
                ? 'bg-green-100 text-green-700 border-green-200' 
                : 'bg-red-100 text-red-700 border-red-200'
            }`}>
              {invoice.status === 'PAID' ? 'Đã thanh toán' : 'Chưa thanh toán'}
            </span>
          </div>

          <div className="space-y-3">
            <div className="flex justify-between items-center text-sm">
              <span className="font-semibold text-slate-500">Tiền phòng:</span>
              <span className="font-bold text-slate-800">{formatCurrency(invoice.room_fee)}</span>
            </div>
            
            <div className="h-px bg-slate-100 my-2" />

            <div className="flex justify-between items-center text-sm">
              <div className="flex flex-col">
                <span className="font-semibold text-slate-500">Tiền điện:</span>
                <span className="text-xs text-slate-400">({invoice.old_electricity_index} → {invoice.new_electricity_index})</span>
              </div>
              <span className="font-bold text-slate-800">{formatCurrency(invoice.electricity_fee)}</span>
            </div>

            <div className="h-px bg-slate-100 my-2" />

            <div className="flex justify-between items-center text-sm">
              <div className="flex flex-col">
                <span className="font-semibold text-slate-500">Tiền nước:</span>
                <span className="text-xs text-slate-400">({invoice.old_water_index} → {invoice.new_water_index})</span>
              </div>
              <span className="font-bold text-slate-800">{formatCurrency(invoice.water_fee)}</span>
            </div>

            <div className="h-px bg-slate-100 my-2" />

            <div className="flex justify-between items-center text-sm">
              <span className="font-semibold text-slate-500">WiFi:</span>
              <span className="font-bold text-slate-800">{formatCurrency(invoice.wifi_fee)}</span>
            </div>
            
            <div className="flex justify-between items-center text-sm">
              <span className="font-semibold text-slate-500">Gửi xe:</span>
              <span className="font-bold text-slate-800">{formatCurrency(invoice.parking_fee)}</span>
            </div>

            <div className="flex justify-between items-center text-sm">
              <span className="font-semibold text-slate-500">Dịch vụ:</span>
              <span className="font-bold text-slate-800">{formatCurrency(invoice.service_fee)}</span>
            </div>

            {invoice.extra_person_fee > 0 && (
              <div className="flex justify-between items-center text-sm">
                <div className="flex flex-col">
                  <span className="font-semibold text-slate-500">Phụ phí người:</span>
                  <span className="text-xs text-slate-400">(vượt {invoice.tenant_count - invoice.extra_person_threshold} người x {formatCurrency(invoice.extra_person_fee_unit)})</span>
                </div>
                <span className="font-bold text-slate-800">{formatCurrency(invoice.extra_person_fee)}</span>
              </div>
            )}

            {invoice.extra_vehicle_fee > 0 && (
              <div className="flex justify-between items-center text-sm">
                <div className="flex flex-col">
                  <span className="font-semibold text-slate-500">Phụ phí xe:</span>
                  <span className="text-xs text-slate-400">(vượt {invoice.vehicle_count - invoice.extra_vehicle_threshold} xe x {formatCurrency(invoice.extra_vehicle_fee_unit)})</span>
                </div>
                <span className="font-bold text-slate-800">{formatCurrency(invoice.extra_vehicle_fee)}</span>
              </div>
            )}

            {invoice.other_fee > 0 && (
              <div className="flex justify-between items-center text-sm">
                <span className="font-semibold text-slate-500">Phí khác:</span>
                <span className="font-bold text-slate-800">{formatCurrency(invoice.other_fee)}</span>
              </div>
            )}

            {invoice.discount > 0 && (
              <div className="flex justify-between items-center text-sm">
                <span className="font-semibold text-red-500">Giảm giá:</span>
                <span className="font-bold text-red-600">-{formatCurrency(invoice.discount)}</span>
              </div>
            )}

          </div>

          <div className="h-px bg-slate-200 mt-6 mb-4" />
          
          <div className="flex justify-between items-center bg-slate-50 p-4 rounded-xl border border-slate-100">
            <span className="font-extrabold text-slate-700">TỔNG CỘNG</span>
            <span className="text-xl font-black text-purple-600">{formatCurrency(invoice.total_amount)}</span>
          </div>

          {invoice.status === 'UNPAID' && (
            <div className="pt-4 space-y-3">
              <Button 
                onClick={handlePay}
                disabled={isPaying}
                className="w-full h-12 bg-green-600 hover:bg-green-700 text-white font-bold text-base rounded-xl flex items-center justify-center gap-2 transition-all active:scale-95 cursor-pointer"
              >
                {isPaying ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <>
                    <CheckCircle className="w-5 h-5" />
                    Đánh dấu đã thanh toán
                  </>
                )}
              </Button>

              {onEdit && (
                <Button 
                  onClick={onEdit}
                  variant="outline"
                  className="w-full h-12 border-slate-200 hover:bg-slate-50 text-slate-700 font-bold text-base rounded-xl flex items-center justify-center gap-2 transition-all active:scale-95 cursor-pointer"
                >
                  Sửa thông tin
                </Button>
              )}
            </div>
          )}

          {invoice.status === 'PAID' && (
            <div className="pt-4">
              <Button 
                onClick={async () => {
                  setIsPaying(true)
                  const res = await useInvoiceStore.getState().unpayInvoice(invoice.id)
                  setIsPaying(false)
                  if (!res.success) {
                    alert(res.message)
                  }
                }}
                disabled={isPaying}
                variant="outline"
                className="w-full h-12 border-orange-200 hover:bg-orange-50 text-orange-700 font-bold text-base rounded-xl flex items-center justify-center gap-2 transition-all active:scale-95 cursor-pointer"
              >
                {isPaying ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <>
                    <X className="w-5 h-5" />
                    Hoàn tác thanh toán
                  </>
                )}
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
