import { Download } from '@/components/icons'
import { AppModal } from '@/components/shared/AppModal'
import { Button } from '@/components/ui/button'

interface Props {
  imageUrl: string
  filename: string
  title?: string
  onClose: () => void
}

// Modal xem trước ảnh hóa đơn: vỏ dùng AppModal (Esc/click nền/X tự xử lý), giữ nguyên logic tải ảnh & click-to-download.
export function BackendImagePreviewModal({ imageUrl, filename, title = "Xem trước hóa đơn", onClose }: Props) {
  const handleDownload = () => {
    const a = document.createElement('a');
    a.href = imageUrl;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  }

  return (
    <AppModal
      open
      onClose={onClose}
      title={
        <span className="flex items-center gap-3">
          {title}
          <span className="text-sm font-medium text-primary bg-primary/10 px-2.5 py-1 rounded-md">Click vào ảnh để tải về</span>
        </span>
      }
      contentClassName="sm:max-w-3xl"
      footer={
        <>
          <Button variant="outline" onClick={onClose} className="rounded-xl font-bold">
            Đóng
          </Button>
          <Button
            onClick={handleDownload}
            className="shadow-sm font-bold flex items-center gap-2"
          >
            <Download className="w-4 h-4" /> Tải về máy
          </Button>
        </>
      }
    >
      <div className="bg-muted/30 flex items-center justify-center min-h-0 -mx-6 -my-4 px-4 py-4">
        <div className="relative group cursor-pointer max-w-full max-h-full flex items-center justify-center h-full" onClick={handleDownload} title="Click để tải về máy">
          <img
            src={imageUrl}
            alt="Hóa đơn preview"
            className="max-w-full max-h-full object-contain shadow-md border border-border/40 rounded-lg transition-transform duration-200 group-hover:scale-[1.02]"
          />
          <div className="absolute inset-0 bg-primary/0 group-hover:bg-primary/5 transition-colors rounded-lg flex items-end justify-center pb-6">
            <div className="opacity-0 group-hover:opacity-100 bg-primary text-primary-foreground p-3.5 rounded-full shadow-xl transform translate-y-4 group-hover:translate-y-0 transition-all duration-200 hover:bg-primary/90">
              <Download className="w-6 h-6" />
            </div>
          </div>
        </div>
      </div>
    </AppModal>
  )
}
