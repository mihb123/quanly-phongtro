export const formatNumber = (val: string | number): string => {
  if (!val && val !== 0) return ''
  let num = typeof val === 'string' ? val.replace(/\D/g, '') : val.toString()
  if (!num) return ''
  // Remove leading zeros
  num = num.replace(/^0+/, '')
  if (!num) return '0'
  return num.replace(/\B(?=(\d{3})+(?!\d))/g, '.')
}

export const parseNumber = (val: string): number => {
  if (!val) return 0
  const cleanStr = val.replace(/\./g, '')
  return parseInt(cleanStr) || 0
}
