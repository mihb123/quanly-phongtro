import { useEffect, useRef } from 'react'

const STATE_KEY = '__overlay'

const stack: string[] = []
let ignoredPops = 0
let listening = false
const handlers = new Map<string, () => void>()
let pendingRelease: { token: string; timer: ReturnType<typeof setTimeout> } | null = null

function readState(): Record<string, unknown> {
  const state = window.history.state
  return state && typeof state === 'object' ? (state as Record<string, unknown>) : {}
}

function onPopState() {
  if (ignoredPops > 0) {
    ignoredPops -= 1
    return
  }
  const top = stack[stack.length - 1]
  if (top) handlers.get(top)?.()
}

function ensureListener() {
  if (listening) return
  window.addEventListener('popstate', onPopState)
  listening = true
}

// Nút Back hệ thống đóng lớp phủ trên cùng (modal/lightbox) thay vì rời trang.
export function useBackToClose(open: boolean, onClose: () => void, dismissible = true) {
  const onCloseRef = useRef(onClose)
  const dismissibleRef = useRef(dismissible)

  useEffect(() => {
    onCloseRef.current = onClose
    dismissibleRef.current = dismissible
  })

  useEffect(() => {
    if (!open || typeof window === 'undefined') return
    ensureListener()

    let token: string
    if (pendingRelease && readState()[STATE_KEY] === pendingRelease.token) {
      // Mount lại ngay sau unmount (StrictMode/re-render): dùng lại entry lịch sử cũ, không push/back thêm.
      clearTimeout(pendingRelease.timer)
      token = pendingRelease.token
      pendingRelease = null
    } else {
      token = `${Date.now()}-${Math.random().toString(36).slice(2)}`
      try {
        window.history.pushState({ ...readState(), [STATE_KEY]: token }, '')
      } catch {
        return
      }
    }
    let closedByBack = false

    stack.push(token)

    handlers.set(token, () => {
      if (!dismissibleRef.current) {
        try {
          window.history.pushState({ ...readState(), [STATE_KEY]: token }, '')
        } catch {
          // Bỏ qua khi trình duyệt chặn pushState.
        }
        return
      }
      closedByBack = true
      onCloseRef.current()
    })

    return () => {
      handlers.delete(token)
      const index = stack.lastIndexOf(token)
      if (index !== -1) stack.splice(index, 1)
      if (closedByBack) return
      if (readState()[STATE_KEY] !== token) return
      const timer = setTimeout(() => {
        if (pendingRelease?.token !== token) return
        pendingRelease = null
        if (readState()[STATE_KEY] === token) {
          ignoredPops += 1
          window.history.back()
        }
      }, 0)
      pendingRelease = { token, timer }
    }
  }, [open])
}
