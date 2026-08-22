import { useEffect, useRef, useState } from 'react'
import { getProtectedFileObjectUrl } from '@/api/files'
import { Loader2, FileIcon } from '@/components/icons'

interface ProtectedFileImageProps {
  path: string
  alt: string
  className?: string
}

export function ProtectedFileImage({ path, alt, className }: ProtectedFileImageProps) {
  const [state, setState] = useState<{ path: string; imageUrl: string | null; hasError: boolean }>({
    path: '',
    imageUrl: null,
    hasError: false,
  })
  const placeholderRef = useRef<HTMLDivElement | null>(null)
  // Môi trường không hỗ trợ IntersectionObserver thì tải ngay như trước
  const [isVisible, setIsVisible] = useState(() => typeof IntersectionObserver === 'undefined')

  // Ảnh là file full-size nên chỉ tải khi tile sắp vào viewport, tránh nổ hàng loạt request khi mở danh sách dài
  useEffect(() => {
    const node = placeholderRef.current
    if (!node || typeof IntersectionObserver === 'undefined') return

    const observer = new IntersectionObserver(
      entries => {
        if (entries.some(entry => entry.isIntersecting)) {
          setIsVisible(true)
          observer.disconnect()
        }
      },
      { rootMargin: '200px' }
    )
    observer.observe(node)

    return () => observer.disconnect()
  }, [path])

  useEffect(() => {
    let isMounted = true
    let objectUrl: string | null = null

    if (!path || !isVisible) return

    getProtectedFileObjectUrl(path)
      .then((url) => {
        objectUrl = url
        if (isMounted) {
          setState({ path, imageUrl: url, hasError: false })
        } else {
          URL.revokeObjectURL(url)
        }
      })
      .catch(() => {
        if (isMounted) setState({ path, imageUrl: null, hasError: true })
      })

    return () => {
      isMounted = false
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [path, isVisible])

  const current = path
    ? (state.path === path ? state : { imageUrl: null, hasError: false })
    : { imageUrl: null, hasError: true }

  if (current.hasError) {
    return (
      <div className={`flex flex-col items-center justify-center bg-muted/60 text-muted-foreground p-1 text-center ${className || ''}`}>
        <FileIcon className="w-5 h-5 text-muted-foreground/60 mb-0.5" />
        <span className="text-[10px]">Lỗi ảnh</span>
      </div>
    )
  }

  if (!current.imageUrl) {
    return (
      <div
        ref={placeholderRef}
        className={`flex items-center justify-center bg-muted/40 text-muted-foreground ${className || ''}`}
      >
        {isVisible && <Loader2 className="w-4 h-4 animate-spin text-primary/60" />}
      </div>
    )
  }

  return <img src={current.imageUrl} alt={alt} className={className} />
}
