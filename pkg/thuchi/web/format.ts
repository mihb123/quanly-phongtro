import type { Freq } from './types'

const vnd = new Intl.NumberFormat('vi-VN')

export function money(n: number, opts: { sign?: boolean } = {}) {
  const s = `${vnd.format(Math.abs(Math.round(n)))} ₫`
  if (n < 0) return `−${s}`
  if (opts.sign && n > 0) return `+${s}`
  return s
}

export function compact(n: number) {
  const a = Math.abs(n)
  const sign = n < 0 ? '−' : ''
  if (a >= 1_000_000_000) return `${sign}${trim(a / 1_000_000_000)}tỷ`
  if (a >= 1_000_000) return `${sign}${trim(a / 1_000_000)}tr`
  if (a >= 1_000) return `${sign}${Math.round(a / 1_000)}k`
  return `${sign}${a}`
}

function trim(x: number) {
  return x >= 100 ? Math.round(x).toString() : x.toFixed(1).replace(/\.0$/, '').replace('.', ',')
}

export function monthKey(d: Date) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

export function dayKey(d: Date) {
  return `${monthKey(d)}-${String(d.getDate()).padStart(2, '0')}`
}

export function monthRange(month: string) {
  const [y, m] = month.split('-').map(Number)
  const start = new Date(y, m - 1, 1)
  const end = new Date(y, m, 1)
  return { start, end, startISO: start.toISOString(), endISO: end.toISOString() }
}

export function shiftMonth(month: string, delta: number) {
  const [y, m] = month.split('-').map(Number)
  return monthKey(new Date(y, m - 1 + delta, 1))
}

export function monthLabel(month: string) {
  const [y, m] = month.split('-').map(Number)
  return `Tháng ${m}/${y}`
}

const weekday = ['Chủ nhật', 'Thứ 2', 'Thứ 3', 'Thứ 4', 'Thứ 5', 'Thứ 6', 'Thứ 7']

export function dayLabel(key: string) {
  const [y, m, d] = key.split('-').map(Number)
  const date = new Date(y, m - 1, d)
  const today = new Date()
  const yesterday = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1)
  if (dayKey(date) === dayKey(today)) return 'Hôm nay'
  if (dayKey(date) === dayKey(yesterday)) return 'Hôm qua'
  return `${weekday[date.getDay()]}, ${d}/${m}${y !== today.getFullYear() ? `/${y}` : ''}`
}

export function timeLabel(iso: string) {
  const d = new Date(iso)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

export function relativeTime(ts: number | string | null) {
  if (!ts) return 'chưa'
  const t = typeof ts === 'number' ? ts : new Date(ts).getTime()
  const s = Math.round((Date.now() - t) / 1000)
  if (s < 60) return 'vừa xong'
  if (s < 3600) return `${Math.floor(s / 60)} phút trước`
  if (s < 86400) return `${Math.floor(s / 3600)} giờ trước`
  return `${Math.floor(s / 86400)} ngày trước`
}

export function parseAmount(input: string) {
  const s = input.trim().toLowerCase().replace(/\s/g, '')
  const m = s.match(/^([\d.,]+)(k|tr|m|t)?$/)
  if (!m) return NaN
  let num = m[2] ? Number(m[1].replace(',', '.')) : Number(m[1].replace(/[.,]/g, ''))
  if (m[2] === 'k') num *= 1_000
  if (m[2] === 'tr' || m[2] === 'm' || m[2] === 't') num *= 1_000_000
  return Math.round(num)
}

export const FREQS: { value: Freq; label: string; unit: string }[] = [
  { value: 'daily', label: 'Hằng ngày', unit: 'ngày' },
  { value: 'weekly', label: 'Hằng tuần', unit: 'tuần' },
  { value: 'monthly', label: 'Hằng tháng', unit: 'tháng' },
  { value: 'yearly', label: 'Hằng năm', unit: 'năm' },
]

export function freqInfo(freq: Freq) {
  return FREQS.find((f) => f.value === freq) ?? FREQS[2]
}

export function recurrenceLabel(r: { freq: Freq; interval: number }) {
  const { label, unit } = freqInfo(r.freq)
  return r.interval > 1 ? `Mỗi ${r.interval} ${unit}` : label
}

export const RECURRENCE_BACKFILL_MONTHS = 12

function occurrenceAt(start: Date, freq: Freq, step: number) {
  if (freq === 'daily' || freq === 'weekly') {
    const d = new Date(start)
    d.setDate(d.getDate() + step * (freq === 'weekly' ? 7 : 1))
    return d
  }
  const months = start.getMonth() + step * (freq === 'yearly' ? 12 : 1)
  const y = start.getFullYear() + Math.floor(months / 12)
  const m = ((months % 12) + 12) % 12
  const last = new Date(y, m + 1, 0).getDate()
  return new Date(y, m, Math.min(start.getDate(), last), start.getHours(), start.getMinutes(), start.getSeconds())
}

export function recurrenceBackfill(r: { freq: Freq; interval: number; start_at: string; end_at: string | null }, now = new Date()) {
  const start = new Date(r.start_at)
  const end = r.end_at ? new Date(r.end_at) : null
  const cutoff = new Date(now)
  cutoff.setMonth(cutoff.getMonth() - RECURRENCE_BACKFILL_MONTHS)
  const step = Math.max(1, r.interval)
  let count = 0
  for (let n = 0; n < 100_000; n++) {
    const at = occurrenceAt(start, r.freq, n * step)
    if (at > now || (end && at > end)) break
    if (at >= cutoff) count++
  }
  return { count, truncated: start < cutoff }
}
