// Log tiến độ ra stderr (stdout giữ cho kết quả / --json) để người chạy biết tool đang ở bước nào.
const START = Date.now()
let quiet = false

export function setQuiet(value) {
  quiet = Boolean(value)
}

export function elapsed(since = START) {
  return `${((Date.now() - since) / 1000).toFixed(1)}s`
}

export function progress(message) {
  if (quiet) return
  process.stderr.write(`[${elapsed().padStart(6)}] ${message}\n`)
}
