interface ProgressBarProps {
  running: boolean
  progress?: number
  total?: number
  message?: string
}

export function ProgressBar({ running, progress, total, message }: ProgressBarProps) {
  if (!running) return null

  const determinate = total !== undefined && total > 0
  const pct = determinate ? Math.min(100, Math.round((progress ?? 0) / total * 100)) : 0

  return (
    <div class="operation-progress">
      <div class={`progress-bar${determinate ? '' : ' indeterminate'}`}>
        <div class="progress-fill" style={determinate ? { width: `${pct}%` } : undefined} />
      </div>
      <span class="progress-text">{message || (determinate ? `${pct}%` : '')}</span>
    </div>
  )
}
