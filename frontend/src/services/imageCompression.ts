import {
  DEFAULT_COMPRESS_OPTIONS,
  compressImageBlob,
  type CompressOptions,
} from './imageCompressionCore'

/**
 * Dưới ngưỡng này ảnh được upload nguyên bản: nén thêm không đáng so với
 * thời gian giải mã. Ngưỡng thấp vì đường lên của điện thoại chỉ ~50-350 KB/s.
 */
export const COMPRESS_THRESHOLD_BYTES = 150 * 1024

/** Chỉ nhận bản nén khi nhỏ hơn bản gốc ít nhất 10%. */
const MIN_SAVING_RATIO = 0.9

/** PNG (ảnh chụp màn hình, scan) hay có chữ nhỏ nên encode ở chất lượng cao hơn. */
const PNG_START_QUALITY = 0.9

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
  resolve: (blob: Blob | null) => void
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
  worker.onmessage = (event: MessageEvent<{ id: number; ok: boolean; blob?: Blob | null; error?: string }>) => {
    const { id, ok, blob, error } = event.data
    const job = pendingJobs.get(id)
    if (!job) return
    pendingJobs.delete(id)
    if (ok) {
      job.resolve(blob ?? null)
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

function compressViaWorker(blob: Blob, options: CompressOptions): Promise<Blob | null> {
  return new Promise<Blob | null>((resolve, reject) => {
    const id = ++jobSeq
    pendingJobs.set(id, { resolve, reject })
    if (idleTimer) clearTimeout(idleTimer)
    getWorker().postMessage({ id, blob, options })
  })
}

export function isCompressibleImage(file: File): boolean {
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

  const isPng = /^image\/png$/i.test(file.type)
  const options: CompressOptions = {
    ...DEFAULT_COMPRESS_OPTIONS,
    ...(isPng ? { startQuality: PNG_START_QUALITY } : {}),
    ...optionOverrides,
  }

  try {
    const blob = workerSupported()
      ? await compressViaWorker(file, options)
      : await compressImageBlob(file, options)

    // null = ảnh đã gọn; nén ra không tiết kiệm đủ thì giữ bản gốc.
    if (!blob || blob.size > file.size * MIN_SAVING_RATIO) return untouched

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

export interface OptimizedImage {
  /** Ảnh sau khi nén, hoặc chính file gốc khi nén ra không nhỏ hơn. */
  blob: Blob
  originalSize: number
  optimizedSize: number
  optimized: boolean
}

/**
 * Nén một ảnh theo yêu cầu trực tiếp của người dùng (trang tối ưu ảnh).
 *
 * Khác `compressImageForUpload`: không bỏ qua ảnh nhỏ và không đoán "ảnh đã gọn"
 * — người dùng đã chủ động chọn nén nên luôn thử encode lại, chỉ trả về bản gốc
 * khi kết quả không nhỏ hơn thật.
 */
export async function optimizeImage(
  file: File,
  overrides: Partial<CompressOptions> = {},
): Promise<OptimizedImage> {
  const isPng = /^image\/png$/i.test(file.type)
  const options: CompressOptions = {
    ...DEFAULT_COMPRESS_OPTIONS,
    ...(isPng ? { startQuality: PNG_START_QUALITY } : {}),
    // Hai ngưỡng này là bộ lọc "có đáng nén không" của luồng upload; ở đây bỏ qua.
    maxBytesPerPixel: 0,
    alwaysCompressOverBytes: 0,
    // Ảnh của người dùng có thể cần nền trong suốt, không được tô trắng thành JPEG.
    preserveTransparency: true,
    ...overrides,
  }

  const blob = workerSupported()
    ? await compressViaWorker(file, options)
    : await compressImageBlob(file, options)

  if (!blob || blob.size >= file.size) {
    return { blob: file, originalSize: file.size, optimizedSize: file.size, optimized: false }
  }

  return { blob, originalSize: file.size, optimizedSize: blob.size, optimized: true }
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
