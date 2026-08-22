/// <reference lib="webworker" />
import { compressImageBlob, type CompressOptions } from './imageCompressionCore'

interface CompressRequest {
  id: number
  blob: Blob
  options: CompressOptions
}

// Nén trong worker để UI không bị đứng khi xử lý ảnh vài MB.
self.onmessage = async (event: MessageEvent<CompressRequest>) => {
  const { id, blob, options } = event.data
  try {
    const output = await compressImageBlob(blob, options)
    self.postMessage({ id, ok: true, blob: output })
  } catch (error) {
    self.postMessage({ id, ok: false, error: error instanceof Error ? error.message : String(error) })
  }
}
