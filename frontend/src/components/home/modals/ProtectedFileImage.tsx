import { useEffect, useState } from 'react'
import { getProtectedFileObjectUrl } from '@/api/files'

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

  useEffect(() => {
    let isMounted = true
    let objectUrl: string | null = null

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
  }, [path])

  const current = state.path === path ? state : { imageUrl: null, hasError: false }

  if (current.hasError) {
    return <div className={className}>Không tải được ảnh</div>
  }

  if (!current.imageUrl) {
    return <div className={className}>Đang tải...</div>
  }

  return <img src={current.imageUrl} alt={alt} className={className} />
}
