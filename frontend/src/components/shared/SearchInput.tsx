import { useEffect, useState } from 'react'
import { Search, X } from '@/components/icons'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

const SEARCH_DEBOUNCE_MS = 500

interface SearchInputProps {
  value: string
  onSearch: (value: string) => void
  placeholder?: string
  label: string
  className?: string
}

// Ô tìm kiếm debounce 500ms: gõ tới đâu hiện tới đó, ngừng gõ mới đẩy vào bộ lọc thật.
export function SearchInput({ value, onSearch, placeholder, label, className }: SearchInputProps) {
  const [draft, setDraft] = useState(value)
  const [syncedValue, setSyncedValue] = useState(value)

  if (value !== syncedValue) {
    setSyncedValue(value)
    setDraft(value)
  }

  useEffect(() => {
    if (draft.trim() === value.trim()) return
    const id = setTimeout(() => onSearch(draft.trim()), SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(id)
  }, [draft, value, onSearch])

  return (
    <div className={cn('relative', className)}>
      <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
      <Input
        type="search"
        enterKeyHint="search"
        aria-label={label}
        value={draft}
        placeholder={placeholder}
        onChange={e => setDraft(e.target.value)}
        className="border-input bg-background pl-9 pr-10 [&::-webkit-search-cancel-button]:hidden"
      />
      {draft && (
        <button
          type="button"
          onClick={() => { setDraft(''); onSearch('') }}
          className="absolute right-0 top-0 flex h-full w-10 items-center justify-center text-muted-foreground hover:text-foreground"
          aria-label="Xóa từ khóa"
        >
          <X className="size-4" />
        </button>
      )}
    </div>
  )
}
