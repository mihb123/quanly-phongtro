import { apiClient } from './client'

const apiPrefix = '/api/v1'

const toApiClientPath = (path: string) => {
  if (path.startsWith(apiPrefix)) {
    return path.slice(apiPrefix.length)
  }
  return path
}

export const getProtectedFileObjectUrl = async (path: string) => {
  const response = await apiClient.get<Blob>(toApiClientPath(path), {
    responseType: 'blob',
  })
  return URL.createObjectURL(response.data)
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
