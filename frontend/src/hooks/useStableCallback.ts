import { useCallback, useLayoutEffect, useRef } from 'react'

// Callback giữ nguyên tham chiếu giữa các lần render nhưng luôn gọi bản mới nhất (dùng cho props của dòng memo).
export function useStableCallback<A extends unknown[], R>(fn: (...args: A) => R) {
  const ref = useRef(fn)
  useLayoutEffect(() => {
    ref.current = fn
  })
  return useCallback((...args: A) => ref.current(...args), [])
}
