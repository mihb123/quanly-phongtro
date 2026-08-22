import { useCallback, useRef, useState } from 'react'
import { compressImagesForUpload } from '@/services/imageCompression'

// Quản lý danh sách file chờ upload: ảnh nặng được nén trước khi vào state
// nên lúc submit chỉ còn việc gửi đi.
export function useUploadFiles() {
  const [files, setFiles] = useState<File[]>([])
  const [isOptimizing, setIsOptimizing] = useState(false)
  const pendingCount = useRef(0)

  const addFiles = useCallback(async (selected: FileList | File[] | null) => {
    const incoming = selected ? Array.from(selected) : []
    if (incoming.length === 0) return

    pendingCount.current += 1
    setIsOptimizing(true)
    try {
      const results = await compressImagesForUpload(incoming)
      setFiles(prev => [...prev, ...results.map(r => r.file)])
    } finally {
      pendingCount.current -= 1
      if (pendingCount.current === 0) setIsOptimizing(false)
    }
  }, [])

  const removeFile = useCallback((index: number) => {
    setFiles(prev => prev.filter((_, i) => i !== index))
  }, [])

  return { files, addFiles, removeFile, isOptimizing }
}
