import { useEffect, useState, useCallback, useRef } from 'react'
import { createPortal } from 'react-dom'
import { X, ZoomIn, Download, ExternalLink, RefreshCw, FileIcon, ChevronLeft, ChevronRight, Loader2 } from '@/components/icons'
import { Button } from '@/components/ui/button'
import { getFileName, isImagePath } from '@/utils/file'
import { getProtectedFileObjectUrl } from '@/api/files'

export interface LightboxImageItem {
  url?: string
  path?: string
  file?: File
  title?: string
  filename?: string
}

export interface ImageLightboxModalProps {
  isOpen: boolean
  imageUrl?: string | null
  title?: string
  filename?: string
  images?: LightboxImageItem[]
  initialIndex?: number
  onClose: () => void
}

interface ImageLightboxContentProps {
  items: LightboxImageItem[]
  initialIndex: number
  onClose: () => void
}

function ImageLightboxContent({
  items,
  initialIndex,
  onClose,
}: ImageLightboxContentProps) {
  const [currentIndex, setCurrentIndex] = useState(() => Math.max(0, Math.min(initialIndex, items.length - 1)))
  const [zoom, setZoom] = useState(1)
  const [rotation, setRotation] = useState(0)

  // Cache resolved URLs (for paths or files) by index
  const [resolvedUrls, setResolvedUrls] = useState<Record<number, string>>({})
  const [loadingIndex, setLoadingIndex] = useState<number | null>(null)

  // Track created blob URLs to revoke on unmount
  const createdBlobUrlsRef = useRef<Set<string>>(new Set())

  const currentItem = items[currentIndex] || items[0]

  // Resolve URL for a given item index
  const resolveItemUrl = useCallback(async (index: number) => {
    const item = items[index]
    if (!item) return ''

    if (item.url) {
      return item.url
    }

    if (item.file) {
      const objectUrl = URL.createObjectURL(item.file)
      createdBlobUrlsRef.current.add(objectUrl)
      setResolvedUrls((prev) => ({ ...prev, [index]: objectUrl }))
      return objectUrl
    }

    if (item.path) {
      try {
        const objectUrl = await getProtectedFileObjectUrl(item.path)
        createdBlobUrlsRef.current.add(objectUrl)
        setResolvedUrls((prev) => ({ ...prev, [index]: objectUrl }))
        return objectUrl
      } catch (err) {
        console.error('Lỗi khi tải ảnh lightbox:', err)
        return ''
      }
    }

    return ''
  }, [items])

  // Load current image URL and prefetch neighbors
  useEffect(() => {
    let isCancelled = false

    const loadCurrentAndNeighbors = async () => {
      const currentUrl = resolvedUrls[currentIndex]
      if (!currentUrl) {
        setLoadingIndex(currentIndex)
        const url = await resolveItemUrl(currentIndex)
        if (!isCancelled && url) {
          setResolvedUrls((prev) => ({ ...prev, [currentIndex]: url }))
        }
        if (!isCancelled) {
          setLoadingIndex(null)
        }
      }

      // Prefetch next and previous
      if (items.length > 1) {
        const nextIdx = (currentIndex + 1) % items.length
        const prevIdx = (currentIndex - 1 + items.length) % items.length
        if (!resolvedUrls[nextIdx]) {
          void resolveItemUrl(nextIdx)
        }
        if (!resolvedUrls[prevIdx]) {
          void resolveItemUrl(prevIdx)
        }
      }
    }

    void loadCurrentAndNeighbors()

    return () => {
      isCancelled = true
    }
  }, [currentIndex, items.length, resolveItemUrl, resolvedUrls])

  // Cleanup all created blob URLs on unmount
  useEffect(() => {
    const createdBlobs = createdBlobUrlsRef.current
    return () => {
      createdBlobs.forEach((url) => {
        if (url.startsWith('blob:')) {
          URL.revokeObjectURL(url)
        }
      })
      createdBlobs.clear()
    }
  }, [])

  // Lock body scroll while open
  useEffect(() => {
    const originalOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = originalOverflow
    }
  }, [])

  const handlePrev = useCallback(() => {
    if (items.length <= 1) return
    setZoom(1)
    setRotation(0)
    setCurrentIndex((prev) => (prev > 0 ? prev - 1 : items.length - 1))
  }, [items.length])

  const handleNext = useCallback(() => {
    if (items.length <= 1) return
    setZoom(1)
    setRotation(0)
    setCurrentIndex((prev) => (prev < items.length - 1 ? prev + 1 : 0))
  }, [items.length])

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        onClose()
      } else if (e.key === 'ArrowLeft') {
        e.preventDefault()
        e.stopPropagation()
        handlePrev()
      } else if (e.key === 'ArrowRight') {
        e.preventDefault()
        e.stopPropagation()
        handleNext()
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
  }, [onClose, handlePrev, handleNext])

  // Touch swipe support for mobile
  const touchStartXRef = useRef<number | null>(null)
  const handleTouchStart = (e: React.TouchEvent) => {
    touchStartXRef.current = e.touches[0].clientX
  }
  const handleTouchEnd = (e: React.TouchEvent) => {
    if (touchStartXRef.current === null) return
    const touchEndX = e.changedTouches[0].clientX
    const diff = touchEndX - touchStartXRef.current
    if (diff > 50) {
      handlePrev()
    } else if (diff < -50) {
      handleNext()
    }
    touchStartXRef.current = null
  }

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

  const activeUrl = resolvedUrls[currentIndex] || currentItem?.url || ''
  const activeFilename = currentItem?.filename || (currentItem?.path ? getFileName(currentItem.path) : currentItem?.file?.name) || (activeUrl ? getFileName(activeUrl) : 'image.png')
  const activeTitle = currentItem?.title || activeFilename

  const handleDownload = useCallback(() => {
    if (!activeUrl) return
    const a = document.createElement('a')
    a.href = activeUrl
    a.download = activeFilename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }, [activeUrl, activeFilename])

  const handleOpenNewTab = useCallback(() => {
    if (!activeUrl) return
    window.open(activeUrl, '_blank')
  }, [activeUrl])

  const isImg =
    activeUrl.startsWith('blob:') ||
    activeUrl.startsWith('data:image') ||
    activeUrl.includes('/files/') ||
    isImagePath(activeUrl) ||
    isImagePath(currentItem?.path) ||
    Boolean(currentItem?.file?.type.startsWith('image/'))

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={activeTitle || 'Xem ảnh'}
      className="fixed inset-0 z-[9999] flex flex-col justify-between bg-black/90 backdrop-blur-md safe-fade-in select-none"
      onClick={onClose}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
    >
      {/* Top Header Bar */}
      <header
        className="flex items-center justify-between gap-3 px-4 py-3 sm:px-6 sm:py-4 bg-gradient-to-b from-black/80 via-black/40 to-transparent z-10"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2 overflow-hidden text-white/90">
          {items.length > 1 && (
            <span className="text-xs sm:text-sm font-semibold px-2.5 py-1 rounded-full bg-white/15 text-white shrink-0">
              {currentIndex + 1} / {items.length}
            </span>
          )}
          <span className="text-sm sm:text-base font-medium truncate max-w-[180px] sm:max-w-md md:max-w-lg" title={activeTitle}>
            {activeTitle}
          </span>
          {zoom !== 1 && (
            <span className="text-xs font-mono px-2 py-0.5 rounded-full bg-white/10 text-white/80 shrink-0">
              {Math.round(zoom * 100)}%
            </span>
          )}
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
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
            disabled={!activeUrl}
            className="p-2 text-white/80 hover:text-white hover:bg-white/10 disabled:opacity-30 rounded-lg transition-colors cursor-pointer"
            title="Tải về máy"
            aria-label="Tải về máy"
          >
            <Download className="w-4 h-4" />
          </button>

          <button
            type="button"
            onClick={handleOpenNewTab}
            disabled={!activeUrl}
            className="p-2 text-white/80 hover:text-white hover:bg-white/10 disabled:opacity-30 rounded-lg transition-colors cursor-pointer"
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

      {/* Navigation Arrow Previous */}
      {items.length > 1 && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            handlePrev()
          }}
          className="fixed left-3 sm:left-6 top-1/2 -translate-y-1/2 z-20 p-3 sm:p-3.5 rounded-full bg-black/40 hover:bg-white/20 text-white/90 hover:text-white backdrop-blur-md border border-white/10 transition-all hover:scale-110 active:scale-95 cursor-pointer shadow-xl group"
          title="Ảnh trước (Mũi tên trái ←)"
          aria-label="Ảnh trước"
        >
          <ChevronLeft className="w-6 h-6 sm:w-7 sm:h-7 transition-transform group-hover:-translate-x-0.5" />
        </button>
      )}

      {/* Navigation Arrow Next */}
      {items.length > 1 && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            handleNext()
          }}
          className="fixed right-3 sm:right-6 top-1/2 -translate-y-1/2 z-20 p-3 sm:p-3.5 rounded-full bg-black/40 hover:bg-white/20 text-white/90 hover:text-white backdrop-blur-md border border-white/10 transition-all hover:scale-110 active:scale-95 cursor-pointer shadow-xl group"
          title="Ảnh tiếp theo (Mũi tên phải →)"
          aria-label="Ảnh tiếp theo"
        >
          <ChevronRight className="w-6 h-6 sm:w-7 sm:h-7 transition-transform group-hover:translate-x-0.5" />
        </button>
      )}

      {/* Main Content Area */}
      <main
        className="flex-1 flex items-center justify-center p-4 sm:p-8 overflow-hidden relative"
        onClick={onClose}
      >
        {loadingIndex === currentIndex ? (
          <div className="flex flex-col items-center gap-3 text-white">
            <Loader2 className="w-10 h-10 animate-spin text-primary" />
            <span className="text-sm font-medium">Đang tải ảnh...</span>
          </div>
        ) : isImg && activeUrl ? (
          <div
            className="max-w-full max-h-full flex items-center justify-center transition-transform duration-150 ease-out"
            style={{
              transform: `scale(${zoom}) rotate(${rotation}deg)`,
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <img
              key={activeUrl}
              src={activeUrl}
              alt={activeTitle || 'Ảnh xem trước'}
              className="max-w-[92vw] max-h-[82vh] object-contain rounded-lg shadow-2xl border border-white/10 pointer-events-auto cursor-default safe-fade-in"
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
              <h3 className="font-semibold text-foreground text-lg">{activeTitle || 'Tài liệu'}</h3>
              <p className="text-sm text-muted-foreground mt-1">Định dạng file này không thể xem trước trực tiếp.</p>
            </div>
            <div className="flex gap-2 mt-2">
              <Button onClick={handleDownload} disabled={!activeUrl} variant="default" className="gap-2">
                <Download className="w-4 h-4" /> Tải về máy
              </Button>
              <Button onClick={handleOpenNewTab} disabled={!activeUrl} variant="outline" className="gap-2">
                <ExternalLink className="w-4 h-4" /> Mở tab mới
              </Button>
            </div>
          </div>
        )}
      </main>

      {/* Bottom Hint Bar */}
      <footer
        className="flex items-center justify-center py-2 px-4 bg-gradient-to-t from-black/60 to-transparent z-10 text-white/60 text-xs gap-3 sm:gap-4 select-none"
        onClick={(e) => e.stopPropagation()}
      >
        {items.length > 1 && (
          <>
            <span className="flex items-center gap-1">
              <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white font-mono text-[10px]">←</kbd>
              <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white font-mono text-[10px]">→</kbd>
              <span>chuyển ảnh ({currentIndex + 1}/{items.length})</span>
            </span>
            <span className="hidden sm:inline">•</span>
          </>
        )}
        <span className="hidden sm:inline">Nhấn <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white/80 font-mono text-[10px]">Esc</kbd> để đóng</span>
        {isImg && (
          <>
            <span className="hidden sm:inline">•</span>
            <span className="hidden sm:inline"><kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white/80 font-mono text-[10px]">+</kbd> / <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white/80 font-mono text-[10px]">−</kbd> để thu phóng</span>
            <span className="hidden sm:inline">•</span>
            <span className="hidden sm:inline"><kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white/80 font-mono text-[10px]">R</kbd> để xoay</span>
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
  images,
  initialIndex = 0,
  onClose,
}: ImageLightboxModalProps) {
  if (!isOpen) return null

  // Normalize single image or gallery list
  const items: LightboxImageItem[] =
    images && images.length > 0
      ? images
      : imageUrl
      ? [{ url: imageUrl, title, filename }]
      : []

  if (items.length === 0) return null

  return createPortal(
    <ImageLightboxContent
      items={items}
      initialIndex={initialIndex}
      onClose={onClose}
    />,
    document.body
  )
}
