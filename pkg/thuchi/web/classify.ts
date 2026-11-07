import type { Category, Direction, Rule } from './types'
import type { KeywordDict } from './presets'

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

export function suggestCategory(input: string, direction: Direction, categories: Category[], rules: Rule[], dict: KeywordDict): string | null {
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
  let best: string | null = null
  let bestLen = 0
  for (const [words, key] of dict) {
    for (const w of words) {
      if (w.length <= bestLen || !hasWord(text, w)) continue
      const cat = categories.find((c) => c.key === key && c.kind === kind)
      if (cat) {
        best = cat.id
        bestLen = w.length
      }
    }
  }
  return best
}
