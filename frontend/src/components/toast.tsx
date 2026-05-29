import { useToast } from "../hooks/toast"

export function ToastContainer() {
  const { toasts, dismiss } = useToast()

  if (toasts.length === 0) return null

  return (
    <div class="toast-container">
      {toasts.map((t) => (
        <div key={t.id} class={`toast toast-${t.type}`} onClick={() => dismiss(t.id)}>
          <span class="toast-msg">{t.message}</span>
          <button class="toast-close">&times;</button>
        </div>
      ))}
    </div>
  )
}
