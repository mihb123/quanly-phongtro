import { useEffect, useState, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { X, ZoomIn, Download, ExternalLink, RefreshCw, FileIcon } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { getFileName, isImagePath } from '@/utils/file'

interface ImageLightboxModalProps {
  isOpen: boolean
  imageUrl: string | null
  title?: string
  filename?: string
  onClose: () => void
}

interface ImageLightboxContentProps {
  imageUrl: string
  title?: string
  filename?: string
  onClose: () => void
}

function ImageLightboxContent({
  imageUrl,
  title,
  filename,
  onClose,
}: ImageLightboxContentProps) {
  const [zoom, setZoom] = useState(1)
  const [rotation, setRotation] = useState(0)

  // Lock body scroll while open
  useEffect(() => {
    const originalOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = originalOverflow
    }
  }, [])

  // Keyboard navigation (with capture phase so it intercepts Escape before any underlying Base UI Dialog)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        onClose()
      } else if (e.key === '+' || e.key === '=') {
        e.preventDefault()
        setZoom((prev) => Math.min(prev + 0.25, 3))
      } else if (e.key === '-') {
        e.preventDefault()
        setZoom((prev) => Math.max(prev - 0.25, 0.5))
      } else if (e.key === '0') {
        e.preventDefault()
        setZoom(1)
        setRotation(0)
      } else if (e.key.toLowerCase() === 'r') {
        e.preventDefault()
        setRotation((prev) => (prev + 90) % 360)
      }
    }

    window.addEventListener('keydown', handleKeyDown, true)
    return () => window.removeEventListener('keydown', handleKeyDown, true)
  }, [onClose])

  const handleZoomIn = useCallback(() => {
    setZoom((prev) => Math.min(Number((prev + 0.25).toFixed(2)), 3))
  }, [])

  const handleZoomOut = useCallback(() => {
    setZoom((prev) => Math.max(Number((prev - 0.25).toFixed(2)), 0.5))
  }, [])

  const handleResetZoom = useCallback(() => {
    setZoom(1)
    setRotation(0)
  }, [])

  const handleRotate = useCallback(() => {
    setRotation((prev) => (prev + 90) % 360)
  }, [])

  const handleDownload = useCallback(() => {
    if (!imageUrl) return
    const name = filename || title || (imageUrl.startsWith('blob:') ? 'image-preview.png' : getFileName(imageUrl))
    const a = document.createElement('a')
    a.href = imageUrl
    a.download = name
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }, [imageUrl, filename, title])

  const handleOpenNewTab = useCallback(() => {
    if (!imageUrl) return
    window.open(imageUrl, '_blank')
  }, [imageUrl])

  const isImg =
    imageUrl.startsWith('blob:') ||
    imageUrl.startsWith('data:image') ||
    imageUrl.includes('/files/') ||
    isImagePath(imageUrl)

  const displayName = title || (filename ? filename : getFileName(imageUrl))

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={displayName || 'Xem ảnh'}
      className="fixed inset-0 z-[9999] flex flex-col justify-between bg-black/90 backdrop-blur-md safe-fade-in select-none"
      onClick={onClose}
    >
      {/* Top Header Bar */}
      <header
        className="flex items-center justify-between gap-3 px-4 py-3 sm:px-6 sm:py-4 bg-gradient-to-b from-black/80 via-black/40 to-transparent z-10"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2 overflow-hidden text-white/90">
          <span className="text-sm sm:text-base font-medium truncate max-w-[200px] sm:max-w-md md:max-w-lg" title={displayName}>
            {displayName}
          </span>
          {zoom !== 1 && (
            <span className="text-xs font-mono px-2 py-0.5 rounded-full bg-white/10 text-white/80">
              {Math.round(zoom * 100)}%
            </span>
          )}
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-1.5 sm:gap-2">
          {isImg && (
            <>
              <button
                type="button"
                onClick={handleZoomOut}
                disabled={zoom <= 0.5}
                className="p-2 text-white/80 hover:text-white hover:bg-white/10 disabled:opacity-30 disabled:hover:bg-transparent rounded-lg transition-colors cursor-pointer"
                title="Thu nhỏ (-)"
                aria-label="Thu nhỏ"
              >
                <span className="text-lg leading-none font-bold px-0.5">−</span>
              </button>

              <button
                type="button"
                onClick={handleZoomIn}
                disabled={zoom >= 3}
                className="p-2 text-white/80 hover:text-white hover:bg-white/10 disabled:opacity-30 disabled:hover:bg-transparent rounded-lg transition-colors cursor-pointer"
                title="Phóng to (+)"
                aria-label="Phóng to"
              >
                <ZoomIn className="w-4 h-4" />
              </button>

              {(zoom !== 1 || rotation !== 0) && (
                <button
                  type="button"
                  onClick={handleResetZoom}
                  className="px-2.5 py-1 text-xs text-white/80 hover:text-white hover:bg-white/10 rounded-lg transition-colors cursor-pointer font-medium"
                  title="Đặt lại kích thước (0)"
                >
                  100%
                </button>
              )}

              <button
                type="button"
                onClick={handleRotate}
                className="p-2 text-white/80 hover:text-white hover:bg-white/10 rounded-lg transition-colors cursor-pointer"
                title="Xoay ảnh 90° (R)"
                aria-label="Xoay ảnh"
              >
                <RefreshCw className="w-4 h-4" />
              </button>
            </>
          )}

          <button
            type="button"
            onClick={handleDownload}
            className="p-2 text-white/80 hover:text-white hover:bg-white/10 rounded-lg transition-colors cursor-pointer"
            title="Tải về máy"
            aria-label="Tải về máy"
          >
            <Download className="w-4 h-4" />
          </button>

          <button
            type="button"
            onClick={handleOpenNewTab}
            className="p-2 text-white/80 hover:text-white hover:bg-white/10 rounded-lg transition-colors cursor-pointer"
            title="Mở tab mới"
            aria-label="Mở tab mới"
          >
            <ExternalLink className="w-4 h-4" />
          </button>

          <div className="h-5 w-px bg-white/20 mx-1" />

          <button
            type="button"
            onClick={onClose}
            className="p-2 text-white hover:text-white bg-white/10 hover:bg-white/20 rounded-full transition-colors cursor-pointer"
            title="Đóng (Esc)"
            aria-label="Đóng"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      </header>

      {/* Main Content Area */}
      <main
        className="flex-1 flex items-center justify-center p-4 sm:p-8 overflow-hidden relative"
        onClick={onClose}
      >
        {isImg ? (
          <div
            className="max-w-full max-h-full flex items-center justify-center transition-transform duration-150 ease-out"
            style={{
              transform: `scale(${zoom}) rotate(${rotation}deg)`,
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <img
              src={imageUrl}
              alt={displayName || 'Ảnh xem trước'}
              className="max-w-[92vw] max-h-[82vh] object-contain rounded-lg shadow-2xl border border-white/10 pointer-events-auto cursor-default"
              draggable={false}
            />
          </div>
        ) : (
          <div
            className="bg-card text-card-foreground p-8 rounded-xl shadow-2xl border border-border flex flex-col items-center gap-4 text-center max-w-md mx-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="w-16 h-16 rounded-full bg-primary/10 text-primary flex items-center justify-center">
              <FileIcon className="w-8 h-8" />
            </div>
            <div>
              <h3 className="font-semibold text-foreground text-lg">{displayName || 'Tài liệu'}</h3>
              <p className="text-sm text-muted-foreground mt-1">Định dạng file này không thể xem trước trực tiếp.</p>
            </div>
            <div className="flex gap-2 mt-2">
              <Button onClick={handleDownload} variant="default" className="gap-2">
                <Download className="w-4 h-4" /> Tải về máy
              </Button>
              <Button onClick={handleOpenNewTab} variant="outline" className="gap-2">
                <ExternalLink className="w-4 h-4" /> Mở tab mới
              </Button>
            </div>
          </div>
        )}
      </main>

      {/* Bottom Hint Bar */}
      <footer
        className="flex items-center justify-center py-2 px-4 bg-gradient-to-t from-black/60 to-transparent z-10 text-white/50 text-xs gap-4 hidden sm:flex"
        onClick={(e) => e.stopPropagation()}
      >
        <span>Nhấn <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white/80 font-mono text-[10px]">Esc</kbd> để đóng</span>
        {isImg && (
          <>
            <span>•</span>
            <span><kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white/80 font-mono text-[10px]">+</kbd> / <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white/80 font-mono text-[10px]">−</kbd> để phóng to/thu nhỏ</span>
            <span>•</span>
            <span><kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white/80 font-mono text-[10px]">R</kbd> để xoay</span>
          </>
        )}
      </footer>
    </div>
  )
}

export function ImageLightboxModal({
  isOpen,
  imageUrl,
  title,
  filename,
  onClose,
}: ImageLightboxModalProps) {
  if (!isOpen || !imageUrl) return null

  return createPortal(
    <ImageLightboxContent
      key={imageUrl}
      imageUrl={imageUrl}
      title={title}
      filename={filename}
      onClose={onClose}
    />,
    document.body
  )
}
