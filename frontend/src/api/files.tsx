import { apiClient } from './client'

const apiPrefix = '/api/v1'

const toApiClientPath = (path: string) => {
  if (path.startsWith(apiPrefix)) {
    return path.slice(apiPrefix.length)
  }
  return path
}

// Backend chưa có thumbnail nên mỗi request là ảnh full-size. Cache Blob theo path để danh sách
// và lightbox dùng chung một lần tải; mỗi caller vẫn tự tạo/revoke object URL riêng nên vòng đời không đổi.
const MAX_CACHED_BLOBS = 40
const blobCache = new Map<string, Promise<Blob>>()

const fetchProtectedBlob = (path: string) => {
  const cached = blobCache.get(path)
  if (cached) {
    // Đưa lên cuối Map để giữ thứ tự LRU
    blobCache.delete(path)
    blobCache.set(path, cached)
    return cached
  }

  const request = apiClient
    .get<Blob>(toApiClientPath(path), { responseType: 'blob' })
    .then(response => response.data)
    .catch(error => {
      blobCache.delete(path)
      throw error
    })

  blobCache.set(path, request)
  while (blobCache.size > MAX_CACHED_BLOBS) {
    const oldest = blobCache.keys().next().value
    if (oldest === undefined) break
    blobCache.delete(oldest)
  }

  return request
}

export const clearProtectedFileCache = (path?: string) => {
  if (path) {
    blobCache.delete(path)
  } else {
    blobCache.clear()
  }
}

export const getProtectedFileObjectUrl = async (path: string) => {
  const blob = await fetchProtectedBlob(path)
  return URL.createObjectURL(blob)
}

export const openProtectedFile = async (path: string) => {
  const previewWindow = window.open('', '_blank')

  try {
    const objectUrl = await getProtectedFileObjectUrl(path)
    if (previewWindow) {
      previewWindow.location.href = objectUrl
    } else {
      window.open(objectUrl, '_blank')
    }
    window.setTimeout(() => URL.revokeObjectURL(objectUrl), 60_000)
  } catch (error) {
    previewWindow?.close()
    throw error
  }
}
