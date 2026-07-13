export const HOUSE_CODE_PATTERN = /^[a-zA-Z0-9_-]+$/

// isHouseCodeTaken kiểm tra nhanh (best-effort) house_code có trùng trong danh sách nhà đang có ở client chưa
// (so khớp không phân biệt hoa/thường); bỏ qua nhà excludeId khi sửa. Backend mới là nơi kiểm tra trùng toàn hệ thống.
export function isHouseCodeTaken(
  houses: { id: string; house_code: string }[],
  code: string,
  excludeId?: string,
): boolean {
  const normalized = code.trim().toLowerCase()
  return houses.some(
    house => house.id !== excludeId && house.house_code.trim().toLowerCase() === normalized,
  )
}

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
