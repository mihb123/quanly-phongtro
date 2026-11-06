const VN_OFFSET = 7 * 3600_000
const pad = (n) => String(n).padStart(2, '0')

export function vnDay(offsetDays = 0) {
  const t = new Date(Date.now() + VN_OFFSET + offsetDays * 86400_000)
  return { y: t.getUTCFullYear(), m: t.getUTCMonth() + 1, d: t.getUTCDate() }
}

export function vnMonth(offsetMonths = 0, day = 1) {
  const { y, m } = vnDay()
  const t = new Date(Date.UTC(y, m - 1 + offsetMonths, 1))
  const last = new Date(Date.UTC(t.getUTCFullYear(), t.getUTCMonth() + 1, 0)).getUTCDate()
  return { y: t.getUTCFullYear(), m: t.getUTCMonth() + 1, d: Math.min(day, last) }
}

export const vnInput = ({ y, m, d }, hh = 12, mm = 0) => `${y}-${pad(m)}-${pad(d)}T${pad(hh)}:${pad(mm)}`
export const vnDate = ({ y, m, d }) => `${y}-${pad(m)}-${pad(d)}`
export const vnInstant = ({ y, m, d }, hh = 12, mm = 0, ss = 0) => new Date(Date.UTC(y, m - 1, d, hh - 7, mm, ss))
export const monthKey = ({ y, m }) => `${y}-${pad(m)}`
export const monthLabel = ({ y, m }) => `Tháng ${m}/${y}`
export const shortMonth = ({ y, m }) => `T${m}/${y}`
export const sepayTime = (date) => {
  const t = new Date(date.getTime() + VN_OFFSET)
  return `${t.getUTCFullYear()}-${pad(t.getUTCMonth() + 1)}-${pad(t.getUTCDate())} ${pad(t.getUTCHours())}:${pad(t.getUTCMinutes())}:${pad(t.getUTCSeconds())}`
}
