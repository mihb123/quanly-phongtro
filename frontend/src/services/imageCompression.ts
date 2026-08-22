import {
  DEFAULT_COMPRESS_OPTIONS,
  compressImageBlob,
  type CompressOptions,
} from './imageCompressionCore'

/** Dưới ngưỡng này ảnh được upload nguyên bản, không cần nén. */
export const COMPRESS_THRESHOLD_BYTES = 1024 * 1024

/** Số ảnh được nén song song, giữ thấp để không ngốn RAM trên điện thoại. */
const MAX_PARALLEL = 2

const WORKER_IDLE_MS = 15_000

const COMPRESSIBLE_MIME = /^image\/(jpeg|jpg|png|webp)$/i
const COMPRESSIBLE_EXT = /\.(jpe?g|png|webp)$/i

export interface CompressImageOptions extends Partial<CompressOptions> {
  /** Ảnh nhỏ hơn ngưỡng này được giữ nguyên. */
  skipUnderBytes?: number
}

export interface CompressedFile {
  /** File để đưa vào FormData: bản đã nén, hoặc file gốc nếu không nén được. */
  file: File
  originalSize: number
  compressed: boolean
}

interface PendingJob {
  resolve: (blob: Blob) => void
  reject: (error: Error) => void
}

let worker: Worker | null = null
let jobSeq = 0
const pendingJobs = new Map<number, PendingJob>()
let idleTimer: ReturnType<typeof setTimeout> | null = null

function workerSupported(): boolean {
  return (
    typeof Worker !== 'undefined' &&
    typeof OffscreenCanvas !== 'undefined' &&
    typeof createImageBitmap !== 'undefined'
  )
}

function failAllPending(message: string) {
  for (const job of pendingJobs.values()) {
    job.reject(new Error(message))
  }
  pendingJobs.clear()
}

function scheduleWorkerShutdown() {
  if (idleTimer) clearTimeout(idleTimer)
  idleTimer = setTimeout(() => {
    if (pendingJobs.size === 0) {
      worker?.terminate()
      worker = null
    }
  }, WORKER_IDLE_MS)
}

function getWorker(): Worker {
  if (worker) return worker
  worker = new Worker(new URL('./imageCompression.worker.ts', import.meta.url), { type: 'module' })
  worker.onmessage = (event: MessageEvent<{ id: number; ok: boolean; blob?: Blob; error?: string }>) => {
    const { id, ok, blob, error } = event.data
    const job = pendingJobs.get(id)
    if (!job) return
    pendingJobs.delete(id)
    if (ok && blob) {
      job.resolve(blob)
    } else {
      job.reject(new Error(error || 'nén ảnh thất bại'))
    }
    if (pendingJobs.size === 0) scheduleWorkerShutdown()
  }
  worker.onerror = () => {
    failAllPending('worker nén ảnh gặp lỗi')
    worker?.terminate()
    worker = null
  }
  return worker
}

function compressViaWorker(blob: Blob, options: CompressOptions): Promise<Blob> {
  return new Promise<Blob>((resolve, reject) => {
    const id = ++jobSeq
    pendingJobs.set(id, { resolve, reject })
    if (idleTimer) clearTimeout(idleTimer)
    getWorker().postMessage({ id, blob, options })
  })
}

function isCompressibleImage(file: File): boolean {
  if (COMPRESSIBLE_MIME.test(file.type)) return true
  return !file.type && COMPRESSIBLE_EXT.test(file.name)
}

function toJpegName(name: string): string {
  const base = name.replace(/\.[^./\\]+$/, '')
  return `${base || 'image'}.jpg`
}

/**
 * Nén một ảnh trước khi upload. File không phải ảnh, ảnh nhỏ hơn ngưỡng,
 * hoặc ảnh nén ra không nhỏ hơn bản gốc đều được trả về nguyên bản.
 */
export async function compressImageForUpload(
  file: File,
  overrides: CompressImageOptions = {},
): Promise<CompressedFile> {
  const { skipUnderBytes = COMPRESS_THRESHOLD_BYTES, ...optionOverrides } = overrides
  const untouched: CompressedFile = { file, originalSize: file.size, compressed: false }

  if (!isCompressibleImage(file) || file.size <= skipUnderBytes) {
    return untouched
  }

  const options: CompressOptions = { ...DEFAULT_COMPRESS_OPTIONS, ...optionOverrides }
  try {
    const blob = workerSupported()
      ? await compressViaWorker(file, options)
      : await compressImageBlob(file, options)

    if (blob.size >= file.size) return untouched

    return {
      file: new File([blob], toJpegName(file.name), {
        type: blob.type || 'image/jpeg',
        lastModified: file.lastModified,
      }),
      originalSize: file.size,
      compressed: true,
    }
  } catch (error) {
    // Nén lỗi (HEIC, ảnh hỏng, hết RAM...) thì vẫn cho upload file gốc.
    console.warn('Không nén được ảnh, upload bản gốc:', file.name, error)
    return untouched
  }
}

/** Nén danh sách file với giới hạn song song. */
export async function compressImagesForUpload(
  files: File[],
  overrides: CompressImageOptions = {},
): Promise<CompressedFile[]> {
  const results: CompressedFile[] = new Array(files.length)
  let cursor = 0

  const runners = Array.from({ length: Math.min(MAX_PARALLEL, files.length) }, async () => {
    while (cursor < files.length) {
      const index = cursor++
      results[index] = await compressImageForUpload(files[index], overrides)
    }
  })

  await Promise.all(runners)
  return results
}
