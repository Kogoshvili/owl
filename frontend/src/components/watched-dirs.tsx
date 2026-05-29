import { useState } from "preact/hooks"
import type { UseMutationResult } from "@tanstack/preact-query"
import { type WatchedDir } from "../api"

interface Props {
  dirs: WatchedDir[]
  selectedDirId?: number | null
  loading?: boolean
  addMutation: UseMutationResult<WatchedDir, Error, string>
  scanMutation: UseMutationResult<WatchedDir, Error, number>
  deleteMutation: UseMutationResult<void, Error, number>
  extractMutation: UseMutationResult<{queued: number}, Error, number>
  onSelect?: (dirId: number | null) => void
}

export function WatchedDirs({ dirs, selectedDirId, loading, addMutation, scanMutation, deleteMutation, extractMutation, onSelect }: Props) {
  const [path, setPath] = useState("")
  const [error, setError] = useState("")

  const handleAdd = async () => {
    const p = path.trim()
    if (!p) return
    setError("")
    try {
      await addMutation.mutateAsync(p)
      setPath("")
    } catch (e: any) {
      setError(e.message)
    }
  }

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Enter") handleAdd()
  }

  const handleScan = async (id: number) => {
    try {
      await scanMutation.mutateAsync(id)
    } catch (e: any) {
      setError(e.message)
    }
  }

  const handleExtract = async (id: number) => {
    try {
      await extractMutation.mutateAsync(id)
    } catch (e: any) {
      setError(e.message)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      if (selectedDirId === id && onSelect) onSelect(null)
    } catch (e: any) {
      setError(e.message)
    }
  }

  const formatTime = (iso: string | null) => {
    if (!iso) return "Never scanned"
    const d = new Date(iso)
    const now = new Date()
    const diffMs = now.getTime() - d.getTime()
    const diffMin = Math.floor(diffMs / 60000)
    if (diffMin < 1) return "Just now"
    if (diffMin < 60) return `${diffMin}m ago`
    if (diffMin < 1440) return `${Math.floor(diffMin / 60)}h ago`
    return d.toLocaleDateString()
  }

  const adding = addMutation.isPending

  return (
    <div class="watched-dirs">
      <h2>Watched Directories</h2>

      <div class="add-dir-form">
        <input
          type="text"
          placeholder="/path/to/directory"
          value={path}
          onInput={(e) => setPath((e.target as HTMLInputElement).value)}
          onKeyDown={handleKeyDown}
          disabled={adding}
        />
        <button class="btn btn-primary" onClick={handleAdd} disabled={adding || !path.trim()}>
          {adding ? "..." : "Add"}
        </button>
        <button class="btn" disabled title="Requires Tauri desktop app">
          Browse
        </button>
      </div>

      {error && <div class="error-msg">{error}</div>}

      <div class="dir-list">
        {loading && <div class="empty">Loading...</div>}
        {!loading && dirs.length === 0 && <div class="empty">No watched directories yet</div>}
        {dirs.map((dir: WatchedDir) => (
          <div
            class={`dir-card${selectedDirId === dir.id ? " selected" : ""}`}
            onClick={() => onSelect?.(selectedDirId === dir.id ? null : dir.id)}
          >
            <div class="dir-card-header">
              <span class="dir-path" title={dir.path}>{dir.path}</span>
            </div>
            <div class="dir-card-meta">
              <span class="dir-scanned">{formatTime(dir.last_scanned_at)}</span>
            </div>
            <div class="dir-card-actions">
              <button
                class="btn btn-sm"
                disabled={scanMutation.isPending}
                onClick={(e) => { e.stopPropagation(); handleScan(dir.id) }}
              >
                Rescan
              </button>
              <button
                class="btn btn-sm"
                disabled={extractMutation.isPending}
                onClick={(e) => { e.stopPropagation(); handleExtract(dir.id) }}
              >
                Extract
              </button>
              <button
                class="btn btn-sm btn-danger"
                disabled={deleteMutation.isPending}
                onClick={(e) => { e.stopPropagation(); handleDelete(dir.id) }}
              >
                Delete
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export function WatchedDirsPanel({ dirs, loading, addMutation, scanMutation, deleteMutation, extractMutation }: Omit<Props, "selectedDirId" | "onSelect">) {
  return (
    <WatchedDirs
      dirs={dirs}
      loading={loading}
      addMutation={addMutation}
      scanMutation={scanMutation}
      deleteMutation={deleteMutation}
      extractMutation={extractMutation}
    />
  )
}
