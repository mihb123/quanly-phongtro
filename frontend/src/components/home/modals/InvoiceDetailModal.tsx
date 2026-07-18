import { useState, useEffect, useRef } from 'react'
import { Receipt, Loader2, Zap, Droplets, Download, Eye } from '@/components/icons'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'
import { useInvoiceStore } from '@/data/invoiceData'
import type { Invoice } from '@/api/invoice'
import { useHouseStore } from '@/data/houseData'
import * as htmlToImage from 'html-to-image'
import { toast } from 'sonner'
import { BackendImagePreviewModal } from './BackendImagePreviewModal'
import { getProtectedFileObjectUrl } from '@/api/files'

const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
}

interface Props {
  invoice: Invoice
  onClose: () => void
  onEdit?: () => void
}

// Modal xem chi tiết hóa đơn (chỉ đọc). Vỏ dùng AppModal; giữ nguyên printRef/contentRef cho việc xuất ảnh hóa đơn.
export function InvoiceDetailModal({ invoice, onClose, onEdit }: Props) {
  const { payInvoice, unpayInvoice } = useInvoiceStore()
  const [isPaying, setIsPaying] = useState(false)
  const [isDownloading, setIsDownloading] = useState(false)
  const [isPreviewing, setIsPreviewing] = useState(false)
  const [previewData, setPreviewData] = useState<{ url: string, filename: string } | null>(null)
  const [transactionImageUrl, setTransactionImageUrl] = useState<string | null>(null)
  const [transactionImageError, setTransactionImageError] = useState(false)
  
  const printRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let isMounted = true
    let objectUrl: string | null = null

    setTransactionImageUrl(null)
    setTransactionImageError(false)

    if (!invoice.transaction_image_path) {
      return
    }

    getProtectedFileObjectUrl(invoice.transaction_image_path)
      .then((url) => {
        objectUrl = url
        if (isMounted) {
          setTransactionImageUrl(url)
        } else {
          URL.revokeObjectURL(url)
        }
      })
      .catch(() => {
        if (isMounted) setTransactionImageError(true)
      })

    return () => {
      isMounted = false
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [invoice.transaction_image_path])

  const handlePay = async () => {
    setIsPaying(true)
    const res = await payInvoice(invoice.id)
    setIsPaying(false)
    if (!res.success) {
      alert(res.message)
    }
  }

  const generateFrontendImage = async () => {
    if (!printRef.current) return null;
    const printEl = printRef.current;
    const contentEl = contentRef.current;
    
    // Tính toán chiều cao thực tế cần thiết để hiển thị toàn bộ nội dung
    let totalHeight = printEl.offsetHeight;
    if (contentEl) {
      const hiddenHeight = contentEl.scrollHeight - contentEl.clientHeight;
      if (hiddenHeight > 0) {
        totalHeight += hiddenHeight;
      }
    }

    const url = await htmlToImage.toPng(printEl, {
      pixelRatio: 2,
      backgroundColor: '#ffffff',
      width: printEl.offsetWidth,
      height: totalHeight,
      style: {
        maxHeight: 'none',
        height: `${totalHeight}px`
      },
      filter: (node) => {
        if (node.nodeType === 3) return true; // Keep text nodes
        const el = node as HTMLElement;
        return !el.classList || !el.classList.contains('no-print');
      }
    });

    const houseNameRaw = useHouseStore.getState().houses.find(h => h.id === invoice.house_id)?.name || 'NhaTro';
    const houseName = houseNameRaw.replace(/\s+/g, '');
    const filename = `${invoice.room_name}_${invoice.period.replace('-', '_')}_${houseName}.png`;

    return { url, filename };
  }

  const handlePreviewFrontend = async () => {
    try {
      setIsPreviewing(true)
      toast.loading('Đang tạo ảnh xem trước...', { id: 'preview-invoice' });
      
      const data = await generateFrontendImage();
      if (data) {
        setPreviewData(data);
      }
      toast.dismiss('preview-invoice');
    } catch (error) {
      console.error('Lỗi tạo ảnh preview:', error);
      toast.error('Lỗi khi tạo ảnh xem trước', { id: 'preview-invoice' });
    } finally {
      setIsPreviewing(false)
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
      const data = await generateFrontendImage();
      if (data) {
        const link = document.createElement('a')
        link.href = data.url
        link.setAttribute('download', data.filename)
        document.body.appendChild(link)
        link.click()
        link.remove()
      }
    } catch (error) {
      console.error('Failed to download invoice:', error)
      alert('Không thể tải xuống hóa đơn')
    } finally {
      setIsDownloading(false)
    }
  }

  // Calculate dynamic usages and unit prices
  const elecUsage = invoice.new_electricity_index - invoice.old_electricity_index;
  const waterUsage = invoice.new_water_index - invoice.old_water_index;
  
  const showUtilityTable = invoice.electricity_fee > 0 || invoice.water_fee > 0;

  const getUtilDisplay = (fee: number, usage: number, unit: string) => {
    if (usage > 0) {
      return {
        usageStr: `${usage} ${unit}`,
        priceStr: formatCurrency(fee / usage)
      }
    }
    if (fee > 0) {
      if (invoice.tenant_count > 0 && fee % invoice.tenant_count === 0 && (fee / invoice.tenant_count) >= 1000) {
        return {
          usageStr: `${invoice.tenant_count} người`,
          priceStr: formatCurrency(fee / invoice.tenant_count)
        }
      }
      return {
        usageStr: `Khoán`,
        priceStr: formatCurrency(fee)
      }
    }
    return { usageStr: '-', priceStr: '-' }
  }

  const elecDisplay = getUtilDisplay(invoice.electricity_fee, elecUsage, 'kWh');
  const waterDisplay = getUtilDisplay(invoice.water_fee, waterUsage, 'm³');

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
    <>
    <AppModal
      open
      onClose={onClose}
      title={
        <span className="flex items-center gap-3">
          <span className="w-10 h-10 rounded-lg bg-primary/10 text-primary flex items-center justify-center shadow-inner">
            <Receipt className="w-5 h-5" />
          </span>
          <span className="flex flex-col">
            <span className="text-lg font-semibold text-foreground tracking-tight">Hóa đơn Phòng {invoice.room_name}</span>
            <span className="text-xs font-semibold text-muted-foreground mt-0.5">Kỳ: {invoice.period}</span>
          </span>
        </span>
      }
      contentClassName="sm:max-w-2xl"
      footer={
        <div className="flex justify-between items-center gap-3 w-full no-print">
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={handlePreviewFrontend}
              disabled={isPreviewing}
              title="Xem trước hóa đơn"
              className="h-10 w-10 shrink-0 border-border bg-card text-muted-foreground hover:text-primary transition-colors cursor-pointer rounded-lg"
            >
               {isPreviewing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Eye className="w-4 h-4" />}
            </Button>
            <Button
              variant="outline"
              size="icon"
              onClick={handleDownload}
              disabled={isDownloading}
              title="Tải hóa đơn"
              className="h-10 w-10 shrink-0 border-border bg-card text-muted-foreground hover:text-primary transition-colors cursor-pointer rounded-lg"
            >
               {isDownloading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Download className="w-4 h-4" />}
            </Button>
          </div>

          <div className="flex gap-2 flex-1 justify-end">
            {(invoice.status === 'UNPAID' || invoice.status === 'PENDING_VERIFICATION') && (
              <>
                {onEdit && (
                  <Button
                    onClick={onEdit}
                    variant="outline"
                    className="h-10 border-border bg-card hover:bg-background text-foreground font-medium rounded-lg transition-colors cursor-pointer"
                  >
                    Sửa
                  </Button>
                )}
                <Button
                  onClick={handlePay}
                  disabled={isPaying}
                  className="h-10 bg-primary hover:bg-primary/90 text-primary-foreground font-medium rounded-lg transition-all shadow-sm shadow-primary/20 cursor-pointer active:scale-95 px-4"
                >
                  {isPaying ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Xác nhận Đã thu'}
                </Button>
              </>
            )}

            {invoice.status === 'PAID' && (
              <Button
                onClick={handleUnpay}
                disabled={isPaying}
                variant="outline"
                className="h-10 border-destructive/20 hover:bg-destructive/10 text-destructive font-medium rounded-lg transition-colors cursor-pointer px-4"
              >
                {isPaying ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Hoàn tác'}
              </Button>
            )}
          </div>
        </div>
      }
    >
      <div ref={printRef} className="bg-card text-card-foreground">
        {/* Content */}
        <div ref={contentRef} className="space-y-6">
          
          {/* Table 1: Utility Statement */}
          {showUtilityTable && (
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
                <span className="size-6 rounded-full bg-muted text-muted-foreground flex items-center justify-center text-xs font-medium">1</span>
                Bảng kê chỉ số Điện / Nước
              </h3>
              
              <div className="border border-border rounded-lg overflow-hidden bg-card shadow-sm">
                <table className="w-full text-sm text-left">
                  <thead className="bg-muted/30 text-muted-foreground font-medium border-b border-border">
                    <tr>
                      <th className="px-4 py-3">Dịch vụ</th>
                      <th className="px-4 py-3 text-center">Chỉ số cũ</th>
                      <th className="px-4 py-3 text-center">Chỉ số mới</th>
                      <th className="px-4 py-3 text-center">Tiêu thụ</th>
                      <th className="px-4 py-3 text-right">Đơn giá</th>
                      <th className="px-4 py-3 text-right">Thành tiền</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border/40">
                    {invoice.electricity_fee > 0 && (
                      <tr className="hover:bg-muted/30 transition-colors">
                        <td className="px-4 py-3 font-semibold text-foreground flex items-center gap-2">
                          <Zap className="w-4 h-4 text-warning" />
                          Điện
                        </td>
                        <td className="px-4 py-3 text-center text-muted-foreground font-medium">{elecUsage > 0 ? invoice.old_electricity_index : '-'}</td>
                        <td className="px-4 py-3 text-center text-muted-foreground font-medium">{elecUsage > 0 ? invoice.new_electricity_index : '-'}</td>
                        <td className="px-4 py-3 text-center font-medium text-foreground">{elecDisplay.usageStr}</td>
                        <td className="px-4 py-3 text-right font-medium text-muted-foreground">{elecDisplay.priceStr}</td>
                        <td className="px-4 py-3 text-right font-medium text-foreground">{formatCurrency(invoice.electricity_fee)}</td>
                      </tr>
                    )}
                    {invoice.water_fee > 0 && (
                      <tr className="hover:bg-muted/30 transition-colors">
                        <td className="px-4 py-3 font-semibold text-foreground flex items-center gap-2">
                          <Droplets className="w-4 h-4 text-info" />
                          Nước
                        </td>
                        <td className="px-4 py-3 text-center text-muted-foreground font-medium">{waterUsage > 0 ? invoice.old_water_index : '-'}</td>
                        <td className="px-4 py-3 text-center text-muted-foreground font-medium">{waterUsage > 0 ? invoice.new_water_index : '-'}</td>
                        <td className="px-4 py-3 text-center font-medium text-foreground">{waterDisplay.usageStr}</td>
                        <td className="px-4 py-3 text-right font-medium text-muted-foreground">{waterDisplay.priceStr}</td>
                        <td className="px-4 py-3 text-right font-medium text-foreground">{formatCurrency(invoice.water_fee)}</td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Table 2: Services Detail */}
          <div className="space-y-4">
            <h3 className="text-sm font-semibold text-foreground flex items-center gap-2">
              <span className="size-6 rounded-full bg-muted text-muted-foreground flex items-center justify-center text-xs font-medium">{showUtilityTable ? '2' : '1'}</span>
              Chi tiết các chi phí dịch vụ
            </h3>

            <div className="border border-border rounded-lg overflow-hidden bg-card shadow-sm">
              <table className="w-full text-sm text-left">
                <thead className="bg-muted/30 text-muted-foreground font-medium border-b border-border">
                  <tr>
                    <th className="px-4 py-3">Khoản mục dịch vụ</th>
                    <th className="px-4 py-3 text-right">Thành tiền</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border/40">
                  {lines.map((line, i) => (
                    <tr key={i} className="hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-3.5">
                        <div className="font-medium text-foreground">{line.label}</div>
                      </td>
                      <td className={`px-4 py-3.5 text-right font-medium whitespace-nowrap ${line.isDiscount ? 'text-destructive' : 'text-foreground'}`}>
                        {line.isDiscount ? '-' : ''}{formatCurrency(line.value)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Summary / Total Card inside content (bottom) */}
          <div className="bg-primary/5 p-5 rounded-lg border border-primary/20 flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4 mt-6">
             <div>
               <p className="text-xs font-medium text-muted-foreground mb-1">Tổng tiền cần thanh toán</p>
               <h3 className="text-2xl font-semibold tabular-nums text-foreground sm:text-3xl">{formatCurrency(invoice.total_amount)}</h3>
             </div>
             <div>
               <span className={`px-3 py-1 rounded-full text-sm font-medium border ${
                  invoice.status === 'PAID' 
                    ? 'bg-success/10 text-success border-success/20'
                    : invoice.status === 'PENDING_VERIFICATION'
                      ? 'bg-warning/10 text-warning border-warning/20'
                      : 'bg-destructive/10 text-destructive border-destructive/20'
                }`}>
                  {invoice.status === 'PAID' ? 'Đã thanh toán' : invoice.status === 'PENDING_VERIFICATION' ? 'Chờ xác nhận CK' : 'Chưa thanh toán'}
                </span>
             </div>
          </div>

          {invoice.transaction_image_path && (
            <div className="mt-6 border border-border rounded-lg p-4 bg-muted/10">
              <h3 className="text-sm font-medium text-foreground mb-4">Ảnh bằng chứng chuyển khoản Zalo</h3>
              <div className="flex justify-center">
                {transactionImageError ? (
                  <div className="text-sm text-muted-foreground">Không tải được ảnh giao dịch</div>
                ) : transactionImageUrl ? (
                  <img
                    src={transactionImageUrl}
                    alt="Bằng chứng giao dịch"
                    className="max-h-[300px] rounded-lg object-contain border border-border shadow-sm"
                  />
                ) : (
                  <div className="text-sm text-muted-foreground">Đang tải ảnh...</div>
                )}
              </div>
            </div>
          )}

        </div>
      </div>
    </AppModal>

    {previewData && (
      <BackendImagePreviewModal
        imageUrl={previewData.url}
        filename={previewData.filename}
        title="Xem trước hóa đơn (Mẫu 2)"
        onClose={() => {
          if (previewData.url.startsWith('blob:')) {
            URL.revokeObjectURL(previewData.url)
          }
          setPreviewData(null)
        }}
      />
    )}
    </>
  )
}
