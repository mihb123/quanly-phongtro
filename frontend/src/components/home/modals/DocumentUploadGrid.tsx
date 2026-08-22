import { Upload, X, FileIcon, ZoomIn, Loader2 } from '@/components/icons'
import { getFileName, isImagePath } from '@/utils/file'
import { ProtectedFileImage } from './ProtectedFileImage'
import type { LightboxImageItem } from '@/components/shared/ImageLightboxModal'

interface DocumentUploadGridProps {
  /** Nhãn dùng cho tooltip/badge của từng file, vd: "Hợp đồng". */
  itemLabel: string
  paths: string[]
  isBusy: boolean
  emptyActionLabel: string
  onAddFiles: (files: FileList | null) => void
  onRequestDelete: (path: string) => void
  onPreview: (items: LightboxImageItem[], initialIndex: number) => void
}

// Lưới ô vuông xem trước / tải lên / xóa các file hồ sơ (CCCD, hợp đồng...). File lưu ngay khi thao tác, do component cha xử lý.
export function DocumentUploadGrid({
  itemLabel,
  paths,
  isBusy,
  emptyActionLabel,
  onAddFiles,
  onRequestDelete,
  onPreview,
}: DocumentUploadGridProps) {
  const gallery: LightboxImageItem[] = paths.map((path, idx) => ({
    path,
    title: `${itemLabel} ${idx + 1}`,
    filename: getFileName(path),
  }))

  return (
    <div className="flex flex-wrap gap-2.5 items-center">
      {paths.map((path, idx) => {
        const fileName = getFileName(path)
        return (
          <div
            key={`${itemLabel}-${idx}`}
            className="relative group w-[100px] h-[100px] rounded-lg overflow-hidden border border-border bg-muted/40 cursor-pointer shadow-xs hover:border-primary/60 transition-all flex items-center justify-center shrink-0"
            onClick={() => onPreview(gallery, idx)}
            title={`${itemLabel} ${idx + 1}: ${fileName}`}
          >
            {isImagePath(path) ? (
              <ProtectedFileImage
                path={path}
                alt={fileName}
                className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
              />
            ) : (
              <div className="flex flex-col items-center justify-center p-2 text-center text-muted-foreground">
                <FileIcon className="w-6 h-6 mb-1 text-primary/70" />
                <span className="text-[10px] font-medium leading-tight line-clamp-2 break-all">{fileName}</span>
              </div>
            )}

            <div className="absolute inset-0 bg-black/30 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center text-white pointer-events-none">
              <ZoomIn className="w-5 h-5" />
            </div>

            <button
              type="button"
              disabled={isBusy}
              onClick={(e) => {
                e.stopPropagation()
                onRequestDelete(path)
              }}
              className="absolute top-1 right-1 size-5 rounded-full bg-background/80 hover:bg-destructive text-muted-foreground hover:text-white backdrop-blur-xs flex items-center justify-center shadow transition-colors z-10 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-background/80 disabled:hover:text-muted-foreground"
              title={`Xóa file ${itemLabel.toLowerCase()}`}
            >
              <X className="w-3 h-3" />
            </button>

            <div className="absolute bottom-1 left-1 px-1.5 py-0.5 rounded bg-black/60 backdrop-blur-xs text-[9px] font-semibold text-white pointer-events-none">
              #{idx + 1}
            </div>
          </div>
        )
      })}

      {isBusy && (
        <div className="w-[100px] h-[100px] rounded-lg border border-dashed border-primary/50 bg-primary/5 flex flex-col items-center justify-center gap-1 text-primary text-[10px] font-medium shrink-0 animate-pulse">
          <Loader2 className="w-5 h-5 animate-spin" />
          <span>Đang lưu...</span>
        </div>
      )}

      <label
        className={`flex flex-col items-center justify-center w-[100px] h-[100px] rounded-lg border-2 border-dashed border-border/80 hover:border-primary hover:bg-primary/5 cursor-pointer text-muted-foreground hover:text-primary transition-all shrink-0 ${
          isBusy ? 'opacity-50 pointer-events-none' : ''
        }`}
        title={`Tải lên thêm file ${itemLabel.toLowerCase()}`}
      >
        <Upload className="w-5 h-5 mb-1" />
        <span className="text-[11px] font-medium text-center px-1 leading-tight">
          {paths.length === 0 ? emptyActionLabel : 'Thêm file'}
        </span>
        <input
          type="file"
          accept=".pdf,image/*"
          multiple
          disabled={isBusy}
          className="hidden"
          onChange={e => {
            onAddFiles(e.target.files)
            e.target.value = ''
          }}
        />
      </label>
    </div>
  )
}
