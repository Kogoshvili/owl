import { createContext, type ComponentChildren } from "preact"
import { useContext, useState, useCallback, useRef } from "preact/hooks"

export interface Toast {
  id: number
  type: "success" | "error" | "info"
  message: string
}

interface ToastContextValue {
  toasts: Toast[]
  show: (t: { type: Toast["type"]; message: string; duration?: number }) => void
  dismiss: (id: number) => void
}

const ToastContext = createContext<ToastContextValue>(null!)

export function useToast() {
  return useContext(ToastContext)
}

let nextId = 1

export function ToastProvider({ children }: { children: ComponentChildren }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const timers = useRef<Map<number, ReturnType<typeof setTimeout>>>(new Map())

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
    const timer = timers.current.get(id)
    if (timer) {
      clearTimeout(timer)
      timers.current.delete(id)
    }
  }, [])

  const show = useCallback(({ type, message, duration = 4000 }: { type: Toast["type"]; message: string; duration?: number }) => {
    const id = nextId++
    setToasts((prev) => [...prev, { id, type, message }])
    const timer = setTimeout(() => dismiss(id), duration)
    timers.current.set(id, timer)
  }, [dismiss])

  return (
    <ToastContext.Provider value={{ toasts, show, dismiss }}>
      {children}
    </ToastContext.Provider>
  )
}
