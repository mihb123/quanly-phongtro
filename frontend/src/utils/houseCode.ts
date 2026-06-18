export const HOUSE_CODE_PATTERN = /^[a-zA-Z0-9_-]+$/

// generateHouseCode creates a short readable code from a house name.
export function generateHouseCode(name: string) {
  const normalized = name
    .replace(/[đĐ]/g, value => (value === 'Đ' ? 'D' : 'd'))
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()

  const words = normalized.match(/[a-z0-9]+/g) || []
  if (words.length === 0) return ''

  const [firstWord = ''] = words
  const startsWithNumber = /^\d/.test(firstWord)
  const code = startsWithNumber
    ? `${firstWord}${words.slice(1).map(word => word.charAt(0)).join('')}`
    : words.map(word => word.charAt(0)).join('')

  return code.slice(0, 12)
}
