import { X, Download } from 'lucide-react'
import { Button } from '@/components/ui/button'

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
  }

  return (
    <div 
      className="fixed inset-0 z-[60] flex items-center justify-center p-4 sm:p-0 bg-slate-900/80 backdrop-blur-sm"
      onClick={onClose}
    >
      <div 
        className="relative bg-white rounded-3xl shadow-2xl max-w-3xl w-full max-h-[90vh] flex flex-col animate-in zoom-in-95 duration-200"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex justify-between items-center p-4 border-b border-slate-100">
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-bold text-slate-800">{title}</h2>
            <span className="text-sm font-medium text-indigo-600 bg-indigo-50 px-2.5 py-1 rounded-md">Click vào ảnh để tải về</span>
          </div>
          <button 
            onClick={onClose}
            className="w-8 h-8 flex items-center justify-center rounded-full bg-slate-100 text-slate-500 hover:bg-slate-200 hover:text-slate-700 transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        
        <div className="flex-1 overflow-auto p-4 bg-slate-50 flex items-center justify-center min-h-[50vh]">
          <div className="relative group cursor-pointer" onClick={handleDownload} title="Click để tải về máy">
            <img 
              src={imageUrl} 
              alt="Hóa đơn preview" 
              className="max-w-full h-auto shadow-md border border-slate-200 rounded-lg transition-transform duration-200 group-hover:scale-[1.01]"
            />
            <div className="absolute inset-0 bg-indigo-900/0 group-hover:bg-indigo-900/5 transition-colors rounded-lg flex items-end justify-center pb-6">
              <div className="opacity-0 group-hover:opacity-100 bg-indigo-600 text-white p-3.5 rounded-full shadow-xl transform translate-y-4 group-hover:translate-y-0 transition-all duration-200 hover:bg-indigo-700">
                <Download className="w-6 h-6" />
              </div>
            </div>
          </div>
        </div>

        <div className="p-4 border-t border-slate-100 flex justify-end gap-3 bg-white rounded-b-3xl">
          <Button variant="outline" onClick={onClose} className="rounded-xl font-bold">
            Đóng
          </Button>
          <Button 
            onClick={handleDownload}
            className="bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl font-bold flex items-center gap-2"
          >
            <Download className="w-4 h-4" /> Tải về máy
          </Button>
        </div>
      </div>
    </div>
  )
}
