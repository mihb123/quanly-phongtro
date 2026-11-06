import type { Category, Direction, Rule } from './types'

const DICT: [string[], string][] = [
  [['GRABFOOD', 'SHOPEEFOOD', 'BEFOOD', 'HIGHLANDS', 'STARBUCKS', 'PHUC LONG', 'KATINAT', 'COFFEE', 'CAFE', 'CA PHE', 'TRA SUA', 'KFC', 'LOTTERIA', 'JOLLIBEE', 'CIRCLE K', 'GS25', 'FAMILYMART', 'PHO', 'BUN', 'COM', 'BANH MI', 'AN SANG', 'AN TRUA', 'AN TOI'], 'an_uong'],
  [['NHA HANG', 'LAU', 'NUONG', 'BUFFET', 'PIZZA'], 'bua_lon'],
  [['GRAB', 'XANH SM', 'BE', 'TAXI', 'GUI XE', 'VE XE', 'VETC', 'EPASS'], 'di_lai'],
  [['XANG', 'PETROLIMEX', 'PVOIL', 'DO XANG'], 'xang_xe'],
  [['SHOPEE', 'LAZADA', 'TIKI', 'WINMART', 'BACH HOA', 'CO OP', 'AEON', 'SIEU THI', 'CHO', 'QUAN AO', 'GIAY'], 'mua_sam'],
  [['GUARDIAN', 'WATSONS', 'HASAKI', 'MY PHAM', 'SON', 'KEM'], 'my_pham'],
  [['CAT TOC', 'GOI DAU', 'NAIL', 'SPA', 'LAM TOC'], 'lam_dep'],
  [['THUOC', 'PHARMACITY', 'LONG CHAU', 'AN KHANG', 'BENH VIEN', 'KHAM'], 'thuoc_men'],
  [['SACH', 'HOC PHI', 'KHOA HOC', 'UDEMY', 'COURSERA'], 'hoc_tap'],
  [['NETFLIX', 'SPOTIFY', 'YOUTUBE', 'APPLE', 'GOOGLE', 'ICLOUD', 'CHATGPT', 'OPENAI', 'CLAUDE'], 'dich_vu_online'],
  [['CGV', 'PHIM', 'GAME', 'STEAM', 'KARAOKE', 'DU LICH'], 'giai_tri'],
  [['TIEN DIEN', 'TIEN NUOC', 'INTERNET', 'WIFI', 'VIETTEL', 'VNPT', 'FPT', 'NAP DIEN THOAI', 'EVN'], 'hoa_don'],
  [['TIEN NHA', 'THUE NHA', 'SUA NHA', 'DIEN MAY', 'NOI THAT'], 'nha_cua'],
  [['LUONG', 'SALARY'], 'luong'],
  [['TIEN PHONG'], 'tien_phong'],
]

export function normalize(s: string) {
  return s
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/đ/g, 'd')
    .replace(/Đ/g, 'D')
    .toUpperCase()
    .replace(/[^A-Z0-9]+/g, ' ')
    .trim()
}

function hasWord(text: string, word: string) {
  return ` ${text} `.includes(` ${word} `)
}

export function suggestCategory(input: string, direction: Direction, categories: Category[], rules: Rule[]): string | null {
  const text = normalize(input)
  if (!text) return null
  const kind = direction === 'out' ? 'expense' : 'income'
  const valid = new Set(categories.filter((c) => c.kind === kind).map((c) => c.id))
  for (const r of rules) {
    if (!r.category_id || !valid.has(r.category_id)) continue
    const p = normalize(r.pattern)
    if (!p) continue
    if ((r.match === 'merchant' && p === text) || (r.match === 'contains' && text.includes(p))) return r.category_id
  }
  for (const r of rules) {
    if (r.match === 'merchant' && r.category_id && valid.has(r.category_id) && text.includes(normalize(r.pattern))) return r.category_id
  }
  for (const [words, key] of DICT) {
    if (words.some((w) => hasWord(text, w))) {
      const cat = categories.find((c) => c.key === key && c.kind === kind)
      if (cat) return cat.id
    }
  }
  return null
}
