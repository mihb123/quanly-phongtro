import { useCallback, useEffect, useId, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import {
  ArrowRight,
  Check,
  Copy,
  Download,
  Eye,
  FileIcon,
  ImageIcon,
  LayoutDashboard,
  Loader2,
  Plus,
  Shield,
  Trash2,
  Upload,
  X,
  Zap,
  ZoomIn,
} from '@/components/icons'
import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { ImageLightboxModal } from '@/components/shared/ImageLightboxModal'
import { PageHeader } from '@/components/shared/PageHeader'
import { isCompressibleImage, optimizeImage } from '@/services/imageCompression'
import type { CompressOptions } from '@/services/imageCompressionCore'
import { cn } from '@/lib/utils'
import { formatFileSize } from '@/utils/file'

// Giới hạn số file tối đa mỗi lần xử lý để đảm bảo bộ nhớ trình duyệt ổn định.
const MAX_FILES = 20

// 2 mức tối ưu rõ ràng theo nhu cầu sử dụng thực tế.
const PRESETS = [
  {
    id: 'auto',
    label: 'Tự động',
    badge: 'Khuyên dùng',
    description: 'Cân bằng tối ưu giữa dung lượng và độ sắc nét',
    detail: 'Tối đa 2000px · Phù hợp cho hầu hết ảnh chụp, phòng trọ, avatar, mạng xã hội',
    overrides: {} as Partial<CompressOptions>,
  },
  {
    id: 'quality',
    label: 'Giữ nét',
    badge: 'Chất lượng cao',
    description: 'Ưu tiên độ sắc nét cao cho văn bản, hóa đơn, giấy tờ',
    detail: 'Tối đa 2400px · Giữ rõ từng chi tiết chữ nhỏ trên CCCD, hợp đồng thuê phòng',
    overrides: {
      maxDimension: 2400,
      startQuality: 0.92,
      minQuality: 0.82,
      targetBytes: 1500 * 1024,
    } as Partial<CompressOptions>,
  },
] as const

type PresetId = (typeof PRESETS)[number]['id']

interface OptimizeItem {
  id: string
  file: File
  previewUrl: string
  optimizedUrl?: string
  status: 'pending' | 'processing' | 'done' | 'error'
  originalSize: number
  optimizedSize?: number
  /** false khi bản nén không nhỏ hơn ảnh gốc nên giữ nguyên bản gốc */
  optimized?: boolean
  blob?: Blob
  downloadName?: string
  error?: string
}

let itemSeq = 0

function outputName(name: string, blob: Blob): string {
  const base = name.replace(/\.[^./\\]+$/, '') || 'image'
  const ext = blob.type === 'image/png' ? 'png' : 'jpg'
  return `${base}-optimized.${ext}`
}

function savingPercent(original: number, optimized: number): number {
  if (original <= 0) return 0
  return Math.round(((original - optimized) / original) * 100)
}

function downloadBlob(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = name
  anchor.click()
  // Revoke sau khi trình duyệt đã kích hoạt tải xuống
  window.setTimeout(() => URL.revokeObjectURL(url), 10_000)
}

export default function OptimizeImagePage() {
  const fileInputId = useId()
  const [items, setItems] = useState<OptimizeItem[]>([])
  const [preset, setPreset] = useState<PresetId>('auto')
  const [isDropzoneDragging, setIsDropzoneDragging] = useState(false)
  const [isWindowDragging, setIsWindowDragging] = useState(false)
  const [isProcessing, setIsProcessing] = useState(false)
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('list')
  const [filterMode, setFilterMode] = useState<'all' | 'saved' | 'untouched'>('all')
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null)

  // Lưu trữ mọi Object URL để thu hồi (revoke) khi unmount hoặc xóa ảnh tránh rò rỉ RAM
  const objectUrlsRef = useRef<Set<string>>(new Set())

  const trackObjectUrl = useCallback((url: string) => {
    objectUrlsRef.current.add(url)
    return url
  }, [])

  const untrackAndRevokeUrl = useCallback((url?: string) => {
    if (url && objectUrlsRef.current.has(url)) {
      URL.revokeObjectURL(url)
      objectUrlsRef.current.delete(url)
    }
  }, [])

  useEffect(() => {
    const urls = objectUrlsRef.current
    return () => {
      urls.forEach((url) => URL.revokeObjectURL(url))
      urls.clear()
    }
  }, [])

  // Nén tuần tự các ảnh trong queue để giữ RAM ổn định trên thiết bị di động
  const runQueue = useCallback(
    async (queue: OptimizeItem[], presetId: PresetId) => {
      const overrides = PRESETS.find((p) => p.id === presetId)?.overrides ?? {}
      setIsProcessing(true)

      for (const item of queue) {
        setItems((prev) =>
          prev.map((it) => (it.id === item.id ? { ...it, status: 'processing' } : it))
        )
        try {
          const result = await optimizeImage(item.file, overrides)
          const optUrl = trackObjectUrl(URL.createObjectURL(result.blob))

          setItems((prev) =>
            prev.map((it) => {
              if (it.id !== item.id) return it
              // Thu hồi URL nén cũ nếu có
              untrackAndRevokeUrl(it.optimizedUrl)
              return {
                ...it,
                status: 'done',
                blob: result.blob,
                optimizedUrl: optUrl,
                optimizedSize: result.optimizedSize,
                optimized: result.optimized,
                downloadName: outputName(item.file.name, result.blob),
                error: undefined,
              }
            })
          )
        } catch (error) {
          console.warn('Không nén được ảnh:', item.file.name, error)
          setItems((prev) =>
            prev.map((it) =>
              it.id === item.id
                ? {
                    ...it,
                    status: 'error',
                    error: 'Không thể đọc ảnh này (định dạng không hỗ trợ hoặc file hỏng)',
                  }
                : it
            )
          )
        }
      }

      setIsProcessing(false)
    },
    [trackObjectUrl, untrackAndRevokeUrl]
  )

  const addFiles = useCallback(
    (selected: FileList | File[] | null) => {
      const incoming = selected ? Array.from(selected) : []
      if (incoming.length === 0) return

      const images = incoming.filter(isCompressibleImage)
      if (images.length < incoming.length) {
        toast.error(`Đã bỏ qua ${incoming.length - images.length} file không phải định dạng JPG/PNG/WebP`)
      }
      if (images.length === 0) return

      const room = Math.max(0, MAX_FILES - items.length)
      if (images.length > room) {
        toast.error(`Mỗi lần chỉ tối ưu tối đa ${MAX_FILES} ảnh (${room} vị trí còn trống)`)
      }
      const accepted = images.slice(0, room)
      if (accepted.length === 0) return

      const newItems: OptimizeItem[] = accepted.map((file) => {
        const previewUrl = trackObjectUrl(URL.createObjectURL(file))
        return {
          id: `item-${++itemSeq}`,
          file,
          previewUrl,
          status: 'pending',
          originalSize: file.size,
        }
      })

      setItems((prev) => [...prev, ...newItems])
      void runQueue(newItems, preset)
    },
    [items.length, preset, runQueue, trackObjectUrl]
  )

  // Lắng nghe sự kiện Paste ảnh từ Clipboard (Ctrl + V / Cmd + V)
  useEffect(() => {
    const handlePaste = (event: ClipboardEvent) => {
      const clipboardFiles: File[] = []
      if (event.clipboardData?.items) {
        for (const item of Array.from(event.clipboardData.items)) {
          if (item.type.startsWith('image/')) {
            const file = item.getAsFile()
            if (file) clipboardFiles.push(file)
          }
        }
      }
      if (clipboardFiles.length > 0) {
        event.preventDefault()
        toast.success(`Đã nhận ${clipboardFiles.length} ảnh từ bộ nhớ tạm`)
        addFiles(clipboardFiles)
      }
    }

    window.addEventListener('paste', handlePaste)
    return () => window.removeEventListener('paste', handlePaste)
  }, [addFiles])

  // Lắng nghe kéo thả toàn trang để người dùng có thể thả file ở bất kỳ đâu
  useEffect(() => {
    let dragCounter = 0

    const handleDragEnter = (event: DragEvent) => {
      event.preventDefault()
      if (event.dataTransfer?.types?.includes('Files')) {
        dragCounter++
        setIsWindowDragging(true)
      }
    }

    const handleDragLeave = (event: DragEvent) => {
      event.preventDefault()
      dragCounter--
      if (dragCounter <= 0) {
        dragCounter = 0
        setIsWindowDragging(false)
      }
    }

    const handleDragOver = (event: DragEvent) => {
      event.preventDefault()
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = 'copy'
      }
    }

    const handleDrop = (event: DragEvent) => {
      event.preventDefault()
      dragCounter = 0
      setIsWindowDragging(false)
      if (event.dataTransfer?.files) {
        addFiles(event.dataTransfer.files)
      }
    }

    window.addEventListener('dragenter', handleDragEnter)
    window.addEventListener('dragleave', handleDragLeave)
    window.addEventListener('dragover', handleDragOver)
    window.addEventListener('drop', handleDrop)

    return () => {
      window.removeEventListener('dragenter', handleDragEnter)
      window.removeEventListener('dragleave', handleDragLeave)
      window.removeEventListener('dragover', handleDragOver)
      window.removeEventListener('drop', handleDrop)
    }
  }, [addFiles])

  const changePreset = useCallback(
    (next: PresetId) => {
      if (next === preset) return
      setPreset(next)
      if (items.length === 0) return

      // Nén lại toàn bộ ảnh đang có theo mức mới
      const reset: OptimizeItem[] = items.map((item) => {
        untrackAndRevokeUrl(item.optimizedUrl)
        return {
          ...item,
          status: 'pending',
          blob: undefined,
          optimizedUrl: undefined,
          downloadName: undefined,
          optimizedSize: undefined,
          optimized: undefined,
          error: undefined,
        }
      })
      setItems(reset)
      void runQueue(reset, next)
    },
    [items, preset, runQueue, untrackAndRevokeUrl]
  )

  const removeItem = useCallback(
    (id: string) => {
      setItems((prev) => {
        const target = prev.find((it) => it.id === id)
        if (target) {
          untrackAndRevokeUrl(target.previewUrl)
          untrackAndRevokeUrl(target.optimizedUrl)
        }
        return prev.filter((it) => it.id !== id)
      })
    },
    [untrackAndRevokeUrl]
  )

  const clearAll = useCallback(() => {
    items.forEach((it) => {
      untrackAndRevokeUrl(it.previewUrl)
      untrackAndRevokeUrl(it.optimizedUrl)
    })
    setItems([])
  }, [items, untrackAndRevokeUrl])

  const copyBlobToClipboard = useCallback(async (blob: Blob, name: string) => {
    try {
      if (!navigator.clipboard || !window.ClipboardItem) {
        toast.error('Trình duyệt hiện tại không hỗ trợ sao chép ảnh vào clipboard')
        return
      }
      const type = blob.type.includes('png') ? 'image/png' : 'image/jpeg'
      const item = new ClipboardItem({ [type]: blob })
      await navigator.clipboard.write([item])
      toast.success(`Đã sao chép ảnh "${name}" vào bộ nhớ tạm`)
    } catch (err) {
      console.warn('Lỗi copy clipboard:', err)
      toast.error('Không thể sao chép ảnh trên trình duyệt này')
    }
  }, [])

  const doneItems = items.filter((it) => it.status === 'done' && it.blob)
  const processingCount = items.filter((it) => it.status === 'processing').length
  const pendingCount = items.filter((it) => it.status === 'pending').length
  const totalOriginal = doneItems.reduce((sum, it) => sum + it.originalSize, 0)
  const totalOptimized = doneItems.reduce((sum, it) => sum + (it.optimizedSize ?? it.originalSize), 0)
  const totalSaving = savingPercent(totalOriginal, totalOptimized)
  const totalSavedBytes = Math.max(0, totalOriginal - totalOptimized)

  const downloadAll = useCallback(() => {
    if (doneItems.length === 0) return
    toast.info(`Bắt đầu tải xuống ${doneItems.length} ảnh...`)
    doneItems.forEach((item, index) => {
      window.setTimeout(() => {
        if (item.blob && item.downloadName) {
          downloadBlob(item.blob, item.downloadName)
        }
      }, index * 350)
    })
  }, [doneItems])

  // Lọc danh sách hiển thị
  const filteredItems = items.filter((item) => {
    if (filterMode === 'saved') return item.status === 'done' && item.optimized
    if (filterMode === 'untouched') return item.status === 'done' && !item.optimized
    return true
  })

  // Danh sách ảnh truyền vào Lightbox modal để xem full màn hình
  const lightboxImages = useMemo(
    () =>
      items.map((item) => ({
        url: item.optimizedUrl || item.previewUrl,
        title: `${item.file.name} · ${
          item.status === 'done'
            ? `${formatFileSize(item.optimizedSize ?? item.originalSize)} (${
                item.optimized && savingPercent(item.originalSize, item.optimizedSize ?? item.originalSize) > 0
                  ? `Giảm ${savingPercent(item.originalSize, item.optimizedSize ?? item.originalSize)}%`
                  : 'Đã tối ưu sẵn'
              })`
            : item.status === 'processing'
            ? 'Đang xử lý...'
            : 'Chờ xử lý'
        }`,
        filename: item.downloadName || item.file.name,
      })),
    [items]
  )

  const handleOpenLightbox = (itemId: string) => {
    const index = items.findIndex((it) => it.id === itemId)
    if (index !== -1) {
      setLightboxIndex(index)
    }
  }

  return (
    <div className="min-h-[100dvh] bg-background">
      {/* Overlay khi kéo thả file vào toàn màn hình */}
      {isWindowDragging && (
        <div className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-background/85 p-6 backdrop-blur-md transition-all">
          <div className="flex max-w-md flex-col items-center gap-4 rounded-2xl border-2 border-dashed border-primary bg-card/90 p-8 text-center shadow-2xl animate-in fade-in zoom-in-95">
            <div className="flex size-16 items-center justify-center rounded-full bg-primary/10 text-primary animate-bounce">
              <Upload className="size-8" />
            </div>
            <div>
              <h3 className="text-xl font-bold text-foreground">Thả ảnh vào đây để nén ngay</h3>
              <p className="mt-1 text-sm text-muted-foreground">
                Hỗ trợ JPG, PNG, WebP · Xử lý tức thì không tải lên máy chủ
              </p>
            </div>
          </div>
        </div>
      )}

      <div className="mx-auto w-full max-w-4xl space-y-6 px-4 py-6 pb-28 sm:py-10 md:pb-12">
        {/* Header trang */}
        <div className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <PageHeader
              title="Tối ưu ảnh"
              description="Giảm dung lượng ảnh thông minh mà vẫn giữ độ nét chữ và chi tiết. Xử lý 100% trên trình duyệt của bạn."
            />
            <Link
              to="/"
              className={cn(
                buttonVariants({ variant: 'outline', size: 'sm' }),
                'min-h-10 shrink-0 touch-manipulation'
              )}
            >
              Về trang chủ
            </Link>
          </div>

          <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <Badge variant="secondary" className="gap-1.5 py-1 font-normal">
              <Shield className="size-3.5 text-primary" />
              100% Riêng tư
            </Badge>
          </div>
        </div>

        {/* Cấu hình Mức Nén (2 Lựa Chọn) */}
        <Card className="gap-4 p-4 sm:p-5">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <h2 className="text-sm font-semibold text-foreground">Mức tối ưu</h2>
              <p className="text-xs text-muted-foreground">
                Chọn cấu hình phù hợp với loại ảnh bạn cần xử lý
              </p>
            </div>
            {isProcessing && (
              <div className="flex items-center gap-1.5 text-xs text-primary">
                <Loader2 className="size-3.5 animate-spin" />
                <span>Đang nén...</span>
              </div>
            )}
          </div>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {PRESETS.map((option) => {
              const isSelected = preset === option.id
              return (
                <button
                  key={option.id}
                  type="button"
                  aria-pressed={isSelected}
                  disabled={isProcessing}
                  onClick={() => changePreset(option.id)}
                  className={cn(
                    'relative flex flex-col items-start gap-1.5 rounded-xl border p-4 text-left transition-all touch-manipulation',
                    'hover:border-primary/50 hover:bg-muted/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
                    isSelected
                      ? 'border-primary bg-primary/5 shadow-sm ring-1 ring-primary/20'
                      : 'border-border bg-card'
                  )}
                >
                  <div className="flex w-full items-center justify-between">
                    <span className="text-sm font-semibold text-foreground">
                      {option.label}
                    </span>
                    <div className="flex items-center gap-1.5">
                      <Badge
                        variant={isSelected ? 'default' : 'outline'}
                        className="text-[11px] font-normal"
                      >
                        {option.badge}
                      </Badge>
                      {isSelected && (
                        <span className="flex size-5 items-center justify-center rounded-full bg-primary text-primary-foreground">
                          <Check className="size-3.5" />
                        </span>
                      )}
                    </div>
                  </div>
                  <p className="text-xs font-medium text-muted-foreground">
                    {option.description}
                  </p>
                  <p className="text-[11px] text-muted-foreground/80">{option.detail}</p>
                </button>
              )
            })}
          </div>
        </Card>

        {/* Khung Tải ảnh (Dropzone) */}
        {items.length === 0 ? (
          <Card className="overflow-hidden p-0">
            <label
              htmlFor={fileInputId}
              onDragOver={(e) => {
                e.preventDefault()
                setIsDropzoneDragging(true)
              }}
              onDragLeave={() => setIsDropzoneDragging(false)}
              onDrop={(e) => {
                e.preventDefault()
                setIsDropzoneDragging(false)
                addFiles(e.dataTransfer.files)
              }}
              className={cn(
                'flex min-h-64 cursor-pointer flex-col items-center justify-center gap-3 p-8 text-center transition-all touch-manipulation',
                isDropzoneDragging
                  ? 'border-2 border-dashed border-primary bg-primary/10'
                  : 'hover:bg-muted/30 active:bg-muted/50'
              )}
            >
              <div className="relative flex size-16 items-center justify-center rounded-2xl bg-primary/10 text-primary ring-8 ring-primary/5 transition-transform group-hover:scale-105">
                <Upload className="size-8" />
              </div>

              <div className="space-y-1">
                <p className="text-base font-semibold text-foreground">
                  Chọn ảnh hoặc kéo thả vào đây
                </p>
                <p className="text-xs text-muted-foreground">
                  Hỗ trợ JPG, PNG, WebP · Tối đa {MAX_FILES} ảnh mỗi lần
                </p>
              </div>

              <div className="flex flex-wrap items-center justify-center gap-2 pt-2">
                <span className="rounded-md border bg-muted/60 px-2.5 py-1 text-xs font-medium text-muted-foreground">
                  Phím tắt: Dán ảnh trực tiếp với <kbd className="font-sans font-semibold text-foreground">Ctrl + V</kbd>
                </span>
              </div>

              <input
                id={fileInputId}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                multiple
                className="sr-only"
                onChange={(e) => {
                  addFiles(e.target.files)
                  e.target.value = ''
                }}
              />
            </label>
          </Card>
        ) : (
          /* Thanh thêm ảnh nhanh khi đã có danh sách */
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border bg-card p-3 shadow-sm sm:px-4">
            <div className="flex items-center gap-2">
              <span className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <ImageIcon className="size-4" />
              </span>
              <div>
                <p className="text-sm font-medium text-foreground">
                  Đã chọn {items.length}/{MAX_FILES} ảnh
                </p>
                <p className="text-xs text-muted-foreground">
                  Kéo thả thêm ảnh hoặc nhấn Ctrl + V để dán
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <label
                htmlFor={fileInputId}
                className={cn(
                  buttonVariants({ variant: 'outline', size: 'sm' }),
                  'cursor-pointer gap-1.5 touch-manipulation'
                )}
              >
                <Plus className="size-4" />
                <span>Thêm ảnh</span>
                <input
                  id={fileInputId}
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  multiple
                  className="sr-only"
                  onChange={(e) => {
                    addFiles(e.target.files)
                    e.target.value = ''
                  }}
                />
              </label>
            </div>
          </div>
        )}

        {/* Kết quả xử lý */}
        {items.length > 0 && (
          <div className="space-y-4">
            {/* Dashboard Thống Kê Tiết Kiệm */}
            <Card className="gap-0 overflow-hidden p-0">
              <div className="flex flex-wrap items-center justify-between gap-3 border-b bg-muted/20 px-4 py-3 sm:px-5">
                <div className="flex items-center gap-2">
                  <span className="flex size-6 items-center justify-center rounded-md bg-primary/10 text-primary">
                    <Zap className="size-3.5" />
                  </span>
                  <h3 className="text-sm font-semibold text-foreground">
                    Kết quả ({doneItems.length}/{items.length} hoàn thành)
                  </h3>
                </div>

                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={clearAll}
                    disabled={isProcessing}
                    className="h-8 gap-1.5 text-muted-foreground hover:text-destructive"
                  >
                    <Trash2 className="size-3.5" />
                    <span className="hidden sm:inline">Xóa tất cả</span>
                  </Button>
                  <Button
                    size="sm"
                    onClick={downloadAll}
                    disabled={doneItems.length === 0}
                    className="h-8 gap-1.5"
                  >
                    <Download className="size-3.5" />
                    <span>Tải tất cả ({doneItems.length})</span>
                  </Button>
                </div>
              </div>

              {/* Tiến trình khi đang xử lý batch */}
              {isProcessing && (
                <div className="border-b bg-primary/5 px-4 py-2 text-xs text-primary sm:px-5">
                  <div className="flex items-center justify-between gap-2 pb-1.5">
                    <span className="flex items-center gap-1.5 font-medium">
                      <Loader2 className="size-3.5 animate-spin" />
                      Đang xử lý {processingCount + pendingCount} ảnh còn lại...
                    </span>
                    <span className="tabular-nums font-semibold">
                      {Math.round((doneItems.length / items.length) * 100)}%
                    </span>
                  </div>
                  <div className="h-1.5 w-full overflow-hidden rounded-full bg-primary/20">
                    <div
                      className="h-full bg-primary transition-all duration-300"
                      style={{
                        width: `${Math.max(5, (doneItems.length / items.length) * 100)}%`,
                      }}
                    />
                  </div>
                </div>
              )}

              {/* Số liệu thống kê */}
              <div className="grid grid-cols-3 divide-x border-b bg-card text-center">
                <div className="p-3 sm:p-4">
                  <p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                    Ban đầu
                  </p>
                  <p className="mt-0.5 text-base font-bold tabular-nums text-foreground sm:text-lg">
                    {formatFileSize(totalOriginal)}
                  </p>
                </div>
                <div className="p-3 sm:p-4">
                  <p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                    Sau tối ưu
                  </p>
                  <p className="mt-0.5 text-base font-bold tabular-nums text-primary sm:text-lg">
                    {formatFileSize(totalOptimized)}
                  </p>
                </div>
                <div className="p-3 sm:p-4">
                  <p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                    Tiết kiệm
                  </p>
                  <div className="mt-0.5 flex flex-wrap items-center justify-center gap-1">
                    <span className="text-base font-bold tabular-nums text-emerald-600 dark:text-emerald-400 sm:text-lg">
                      -{totalSaving}%
                    </span>
                    {totalSavedBytes > 0 && (
                      <span className="hidden text-xs text-muted-foreground lg:inline">
                        ({formatFileSize(totalSavedBytes)})
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* Bộ lọc và chuyển chế độ xem */}
              <div className="flex flex-wrap items-center justify-between gap-2 bg-muted/10 px-4 py-2.5 sm:px-5">
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={() => setFilterMode('all')}
                    className={cn(
                      'rounded-lg px-2.5 py-1 text-xs font-medium transition-colors',
                      filterMode === 'all'
                        ? 'bg-secondary text-secondary-foreground shadow-xs'
                        : 'text-muted-foreground hover:bg-muted'
                    )}
                  >
                    Tất cả ({items.length})
                  </button>
                  <button
                    type="button"
                    onClick={() => setFilterMode('saved')}
                    className={cn(
                      'rounded-lg px-2.5 py-1 text-xs font-medium transition-colors',
                      filterMode === 'saved'
                        ? 'bg-secondary text-secondary-foreground shadow-xs'
                        : 'text-muted-foreground hover:bg-muted'
                    )}
                  >
                    Đã giảm ({items.filter((it) => it.optimized).length})
                  </button>
                  <button
                    type="button"
                    onClick={() => setFilterMode('untouched')}
                    className={cn(
                      'rounded-lg px-2.5 py-1 text-xs font-medium transition-colors',
                      filterMode === 'untouched'
                        ? 'bg-secondary text-secondary-foreground shadow-xs'
                        : 'text-muted-foreground hover:bg-muted'
                    )}
                  >
                    Đã gọn ({items.filter((it) => it.status === 'done' && !it.optimized).length})
                  </button>
                </div>

                <div className="flex items-center gap-1 rounded-lg border bg-muted/40 p-0.5">
                  <button
                    type="button"
                    onClick={() => setViewMode('grid')}
                    aria-label="Chế độ lưới"
                    className={cn(
                      'flex size-7 items-center justify-center rounded-md text-xs transition-colors',
                      viewMode === 'grid'
                        ? 'bg-background text-foreground shadow-xs'
                        : 'text-muted-foreground hover:text-foreground'
                    )}
                  >
                    <LayoutDashboard className="size-3.5" />
                  </button>
                  <button
                    type="button"
                    onClick={() => setViewMode('list')}
                    aria-label="Chế độ danh sách"
                    className={cn(
                      'flex size-7 items-center justify-center rounded-md text-xs transition-colors',
                      viewMode === 'list'
                        ? 'bg-background text-foreground shadow-xs'
                        : 'text-muted-foreground hover:text-foreground'
                    )}
                  >
                    <FileIcon className="size-3.5" />
                  </button>
                </div>
              </div>
            </Card>

            {/* Danh sách ảnh dạng GRID */}
            {viewMode === 'grid' ? (
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {filteredItems.map((item) => {
                  const saving =
                    item.optimizedSize !== undefined
                      ? savingPercent(item.originalSize, item.optimizedSize)
                      : 0
                  const displayUrl = item.optimizedUrl || item.previewUrl

                  return (
                    <Card
                      key={item.id}
                      className="group relative flex flex-col justify-between overflow-hidden p-3 transition-all hover:shadow-md"
                    >
                      {/* Vùng Preview ảnh: Click vào để xem full ảnh trong Lightbox */}
                      <div className="relative aspect-4/3 w-full overflow-hidden rounded-lg bg-muted">
                        <button
                          type="button"
                          onClick={() => handleOpenLightbox(item.id)}
                          className="relative size-full cursor-zoom-in text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                          aria-label={`Xem phóng to ${item.file.name}`}
                        >
                          <img
                            src={displayUrl}
                            alt={item.file.name}
                            loading="lazy"
                            decoding="async"
                            className="size-full object-cover transition-transform duration-300 group-hover:scale-105"
                          />

                          {/* Lớp phủ hover gợi ý click phóng to */}
                          <div className="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 backdrop-blur-xs transition-opacity group-hover:opacity-100">
                            <span className="flex items-center gap-1.5 rounded-full bg-background/90 px-3 py-1.5 text-xs font-semibold text-foreground shadow-lg">
                              <ZoomIn className="size-3.5" />
                              Xem phóng to
                            </span>
                          </div>
                        </button>

                        {/* Badge phần trăm tiết kiệm trên ảnh */}
                        <div className="absolute top-2 left-2 pointer-events-none">
                          {item.status === 'done' && (
                            item.optimized && saving > 0 ? (
                              <Badge className="bg-emerald-600 text-white shadow-xs">
                                -{saving}%
                              </Badge>
                            ) : (
                              <Badge variant="secondary" className="bg-background/90 backdrop-blur-xs">
                                Đã gọn
                              </Badge>
                            )
                          )}
                          {item.status === 'processing' && (
                            <Badge variant="secondary" className="gap-1 bg-background/90 backdrop-blur-xs">
                              <Loader2 className="size-3 animate-spin" />
                              Đang nén
                            </Badge>
                          )}
                          {item.status === 'error' && (
                            <Badge variant="destructive">Lỗi</Badge>
                          )}
                        </div>

                        {/* Nút xóa nhanh */}
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation()
                            removeItem(item.id)
                          }}
                          className="absolute top-2 right-2 flex size-7 items-center justify-center rounded-full bg-background/80 text-muted-foreground shadow-xs backdrop-blur-xs transition-colors hover:bg-destructive hover:text-white"
                          aria-label={`Xóa ${item.file.name}`}
                        >
                          <X className="size-3.5" />
                        </button>
                      </div>

                      {/* Thông tin chi tiết */}
                      <div className="mt-3 space-y-2">
                        <div className="min-w-0">
                          <p
                            className="truncate text-sm font-medium text-foreground"
                            title={item.file.name}
                          >
                            {item.file.name}
                          </p>

                          {item.status === 'done' ? (
                            <div className="mt-1 flex items-center justify-between text-xs text-muted-foreground">
                              <span className="tabular-nums">{formatFileSize(item.originalSize)}</span>
                              <ArrowRight className="size-3 text-muted-foreground/50" />
                              <span className="font-semibold tabular-nums text-foreground">
                                {formatFileSize(item.optimizedSize ?? item.originalSize)}
                              </span>
                            </div>
                          ) : item.status === 'error' ? (
                            <p className="mt-1 text-xs text-destructive">{item.error}</p>
                          ) : (
                            <p className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
                              <Loader2 className="size-3 animate-spin" />
                              {item.status === 'processing' ? 'Đang tối ưu ảnh...' : 'Đang trong hàng đợi'}
                            </p>
                          )}
                        </div>

                        {/* Thao tác nút bấm */}
                        {item.status === 'done' && item.blob && item.downloadName && (
                          <div className="flex items-center gap-1.5 pt-1">
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-8 flex-1 gap-1 text-xs touch-manipulation"
                              onClick={() => downloadBlob(item.blob!, item.downloadName!)}
                            >
                              <Download className="size-3.5" />
                              Tải về
                            </Button>
                            <Button
                              size="icon-sm"
                              variant="ghost"
                              className="size-8 shrink-0 touch-manipulation text-muted-foreground hover:text-foreground"
                              title="Sao chép ảnh vào clipboard"
                              onClick={() => copyBlobToClipboard(item.blob!, item.downloadName!)}
                            >
                              <Copy className="size-3.5" />
                            </Button>
                            <Button
                              size="icon-sm"
                              variant="ghost"
                              className="size-8 shrink-0 touch-manipulation text-muted-foreground hover:text-foreground"
                              title="Xem toàn màn hình"
                              onClick={() => handleOpenLightbox(item.id)}
                            >
                              <Eye className="size-3.5" />
                            </Button>
                          </div>
                        )}
                      </div>
                    </Card>
                  )
                })}
              </div>
            ) : (
              /* Danh sách ảnh dạng LIST */
              <Card className="gap-0 overflow-hidden p-0">
                <ul className="divide-y">
                  {filteredItems.map((item) => {
                    const saving =
                      item.optimizedSize !== undefined
                        ? savingPercent(item.originalSize, item.optimizedSize)
                        : 0
                    const displayUrl = item.optimizedUrl || item.previewUrl

                    return (
                      <li
                        key={item.id}
                        className="flex items-center gap-3 p-3 transition-colors hover:bg-muted/20 sm:p-4"
                      >
                        {/* Thumbnail có click mở Lightbox */}
                        <button
                          type="button"
                          onClick={() => handleOpenLightbox(item.id)}
                          className="group relative size-14 shrink-0 overflow-hidden rounded-lg border bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary sm:size-16"
                          title="Bấm để xem ảnh phóng to"
                        >
                          <img
                            src={displayUrl}
                            alt={item.file.name}
                            loading="lazy"
                            decoding="async"
                            className="size-full object-cover transition-transform group-hover:scale-110"
                          />
                          <div className="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 transition-opacity group-hover:opacity-100">
                            <ZoomIn className="size-4 text-white" />
                          </div>
                        </button>

                        <div className="min-w-0 flex-1">
                          <p className="truncate text-sm font-medium text-foreground">
                            {item.file.name}
                          </p>

                          {item.status === 'done' ? (
                            <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                              <span className="tabular-nums">{formatFileSize(item.originalSize)}</span>
                              <span aria-hidden>→</span>
                              <span className="font-semibold tabular-nums text-foreground">
                                {formatFileSize(item.optimizedSize ?? item.originalSize)}
                              </span>
                              {item.optimized && saving > 0 ? (
                                <Badge className="bg-emerald-600 text-white tabular-nums">
                                  -{saving}%
                                </Badge>
                              ) : (
                                <Badge variant="outline">Đã gọn</Badge>
                              )}
                            </div>
                          ) : item.status === 'error' ? (
                            <p className="mt-1 text-xs text-destructive">{item.error}</p>
                          ) : (
                            <p className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
                              <Loader2 className="size-3.5 animate-spin" />
                              {item.status === 'processing' ? 'Đang tối ưu...' : 'Đang chờ'}
                            </p>
                          )}
                        </div>

                        {/* Cụm nút thao tác */}
                        <div className="flex shrink-0 items-center gap-1">
                          {item.status === 'done' && item.blob && item.downloadName && (
                            <>
                              <Button
                                size="icon-sm"
                                variant="ghost"
                                className="size-9 text-muted-foreground hover:text-foreground touch-manipulation"
                                title="Sao chép ảnh"
                                onClick={() => copyBlobToClipboard(item.blob!, item.downloadName!)}
                              >
                                <Copy className="size-4" />
                              </Button>
                              <Button
                                size="icon-sm"
                                variant="outline"
                                className="size-9 touch-manipulation"
                                title={`Tải ${item.downloadName}`}
                                onClick={() => downloadBlob(item.blob!, item.downloadName!)}
                              >
                                <Download className="size-4" />
                              </Button>
                            </>
                          )}
                          <Button
                            size="icon-sm"
                            variant="ghost"
                            className="size-9 text-muted-foreground hover:text-destructive touch-manipulation"
                            title={`Xóa ${item.file.name}`}
                            onClick={() => removeItem(item.id)}
                          >
                            <X className="size-4" />
                          </Button>
                        </div>
                      </li>
                    )
                  })}
                </ul>
              </Card>
            )}
          </div>
        )}
      </div>

      {/* Thanh tải về cố định dưới cùng cho thiết bị di động */}
      {doneItems.length > 0 && (
        <div className="fixed inset-x-0 bottom-0 z-40 border-t bg-background/95 p-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] backdrop-blur-md md:hidden">
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="text-xs text-muted-foreground">
                Đã tối ưu {doneItems.length} ảnh
              </p>
              <p className="text-sm font-bold text-primary">
                Tiết kiệm {totalSaving}% ({formatFileSize(totalSavedBytes)})
              </p>
            </div>
            <Button
              className="min-h-11 flex-1 font-semibold touch-manipulation"
              onClick={downloadAll}
            >
              <Download className="size-4" />
              Tải tất cả ({doneItems.length})
            </Button>
          </div>
        </div>
      )}

      {/* Modal Lightbox xem full ảnh đã tối ưu khi người dùng click vào ảnh */}
      <ImageLightboxModal
        isOpen={lightboxIndex !== null}
        images={lightboxImages}
        initialIndex={lightboxIndex ?? 0}
        onClose={() => setLightboxIndex(null)}
      />
    </div>
  )
}
