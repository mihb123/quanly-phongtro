// Thuật toán nén ảnh dùng chung cho Web Worker và main thread (luồng upload
// CCCD/hợp đồng và trang /optimize-img).
// Mục tiêu: giảm số byte của ảnh mà vẫn đọc rõ chữ.

export interface CompressOptions {
  /** Cạnh dài nhất sau khi resize (px). */
  maxDimension: number
  /** Dung lượng mong muốn sau khi nén (byte). */
  targetBytes: number
  /** Chất lượng JPEG bắt đầu. */
  startQuality: number
  /** Ngưỡng chất lượng thấp nhất được phép, tránh làm mờ chữ trên CCCD. */
  minQuality: number
  /**
   * Ngưỡng byte/pixel để coi là ảnh "encode dư". JPEG chụp/scan bình thường
   * khoảng 0.1-0.15 B/px; ảnh chụp màn hình dạng PNG có thể tới hơn 1 B/px.
   */
  maxBytesPerPixel: number
  /** Trên mức này thì luôn nén, không cần xét byte/pixel. */
  alwaysCompressOverBytes: number
  /**
   * Ảnh còn vùng trong suốt thì xuất PNG thay vì JPEG. Luồng upload để false
   * (CCCD/hợp đồng luôn cần ảnh nhẹ); trang tối ưu ảnh bật lên để không làm
   * mất nền trong suốt của ảnh người dùng.
   */
  preserveTransparency?: boolean
}

export const DEFAULT_COMPRESS_OPTIONS: CompressOptions = {
  maxDimension: 2000,
  targetBytes: 900 * 1024,
  startQuality: 0.85,
  minQuality: 0.6,
  maxBytesPerPixel: 0.25,
  alwaysCompressOverBytes: 1024 * 1024,
}

export const OUTPUT_MIME = 'image/jpeg'
export const PNG_MIME = 'image/png'

type AnyCanvas = OffscreenCanvas | HTMLCanvasElement

function createCanvas(width: number, height: number): AnyCanvas {
  if (typeof OffscreenCanvas !== 'undefined') {
    return new OffscreenCanvas(width, height)
  }
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  return canvas
}

async function encode(canvas: AnyCanvas, quality: number, type = OUTPUT_MIME): Promise<Blob> {
  if ('convertToBlob' in canvas) {
    return canvas.convertToBlob({ type, quality })
  }
  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob(
      blob => (blob ? resolve(blob) : reject(new Error('canvas.toBlob trả về null'))),
      type,
      quality,
    )
  })
}

type AnyCanvasContext = CanvasRenderingContext2D | OffscreenCanvasRenderingContext2D

// Quét kênh alpha tìm pixel chưa đục hoàn toàn. Chỉ gọi khi cần biết có nên giữ PNG.
function hasTransparency(ctx: AnyCanvasContext, width: number, height: number): boolean {
  const { data } = ctx.getImageData(0, 0, width, height)
  for (let i = 3; i < data.length; i += 4) {
    if (data[i] !== 255) return true
  }
  return false
}

// imageOrientation: 'from-image' để ảnh chụp từ điện thoại không bị xoay sau khi vẽ lại.
async function decode(blob: Blob): Promise<ImageBitmap> {
  return createImageBitmap(blob, { imageOrientation: 'from-image' })
}

async function resize(source: ImageBitmap, width: number, height: number): Promise<ImageBitmap> {
  try {
    const resized = await createImageBitmap(source, {
      resizeWidth: width,
      resizeHeight: height,
      resizeQuality: 'high',
    })
    source.close()
    return resized
  } catch {
    // Trình duyệt không hỗ trợ resize khi decode: để canvas tự scale bên dưới.
    return source
  }
}

/**
 * Resize + re-encode ảnh về JPEG, hạ dần chất lượng cho tới khi đạt targetBytes
 * nhưng không bao giờ xuống dưới minQuality. Ảnh còn vùng trong suốt ra PNG khi
 * bật preserveTransparency.
 *
 * Trả về null khi ảnh đã gọn so với kích thước thật của nó (không quá
 * maxDimension và byte/pixel thấp) — lúc đó nén lại chỉ mất CPU mà không lợi.
 */
export async function compressImageBlob(blob: Blob, options: CompressOptions): Promise<Blob | null> {
  const source = await decode(blob)
  const scale = Math.min(1, options.maxDimension / Math.max(source.width, source.height))
  const bytesPerPixel = blob.size / (source.width * source.height)
  const worthCompressing =
    scale < 1 ||
    blob.size > options.alwaysCompressOverBytes ||
    bytesPerPixel > options.maxBytesPerPixel

  if (!worthCompressing) {
    source.close()
    return null
  }

  const width = Math.max(1, Math.round(source.width * scale))
  const height = Math.max(1, Math.round(source.height * scale))
  const drawable = scale < 1 ? await resize(source, width, height) : source
  const canvas = createCanvas(width, height)
  const ctx = (canvas as HTMLCanvasElement).getContext('2d') as
    | CanvasRenderingContext2D
    | OffscreenCanvasRenderingContext2D
    | null

  if (!ctx) {
    drawable.close()
    throw new Error('không lấy được canvas 2d context')
  }

  ctx.imageSmoothingEnabled = true
  ctx.imageSmoothingQuality = 'high'
  ctx.drawImage(drawable, 0, 0, width, height)
  drawable.close()

  if (options.preserveTransparency === true && hasTransparency(ctx, width, height)) {
    // PNG giữ được alpha và không có tham số chất lượng nên chỉ encode một lần.
    return encode(canvas, 1, PNG_MIME)
  }

  // JPEG không có alpha: chèn nền trắng xuống dưới để vùng trong suốt không ra nền đen.
  ctx.globalCompositeOperation = 'destination-over'
  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, width, height)
  ctx.globalCompositeOperation = 'source-over'

  let quality = options.startQuality
  let output = await encode(canvas, quality)
  while (output.size > options.targetBytes && quality > options.minQuality) {
    quality = Math.max(options.minQuality, Number((quality - 0.1).toFixed(2)))
    output = await encode(canvas, quality)
  }
  return output
}
