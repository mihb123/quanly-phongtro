import { X, Download } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { createPortal } from 'react-dom'

interface Props {
  imageUrl: string
  filename: string
  title?: string
  onClose: () => void
}

export function BackendImagePreviewModal({ imageUrl, filename, title = "Xem trước hóa đơn", onClose }: Props) {
  const handleDownload = () => {
    const a = document.createElement('a');
    a.href = imageUrl;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    document.body.removeChild(a);
  }

  return createPortal(
    <div 
      className="fixed inset-0 z-[60] flex items-center justify-center p-4 sm:p-0 bg-background/80 backdrop-blur-sm"
      onClick={onClose}
    >
      <div 
        className="relative bg-card text-card-foreground rounded-3xl shadow-2xl max-w-3xl w-full max-h-[90vh] flex flex-col safe-fade-in overflow-y-auto"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex justify-between items-center p-4 border-b border-border/40">
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-bold text-foreground">{title}</h2>
            <span className="text-sm font-medium text-primary bg-primary/10 px-2.5 py-1 rounded-md">Click vào ảnh để tải về</span>
          </div>
          <button 
            onClick={onClose}
            className="w-8 h-8 flex items-center justify-center rounded-full bg-secondary text-muted-foreground hover:bg-secondary/80 hover:text-foreground transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        
        <div className="flex-1 overflow-hidden p-4 bg-muted/30 flex items-center justify-center min-h-0">
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

        <div className="p-4 border-t border-border/40 flex justify-end gap-3 bg-card rounded-b-3xl">
          <Button variant="outline" onClick={onClose} className="rounded-xl font-bold">
            Đóng
          </Button>
          <Button 
            onClick={handleDownload}
            className="shadow-sm font-bold flex items-center gap-2"
          >
            <Download className="w-4 h-4" /> Tải về máy
          </Button>
        </div>
      </div>
    </div>,
    document.body
  )
}
