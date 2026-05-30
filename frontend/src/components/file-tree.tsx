import { useState } from "preact/hooks"
import { type PhysicalFolder, type WatchedDir } from "../api"
import type { UseMutationResult } from "@tanstack/preact-query"
import { usePhysicalFolders, useFolderGuards, useSetFolderGuard } from "../hooks/queries"

export function FileTree({ dirs, addMutation, scanMutation, deleteMutation, anyRunning }: {
  dirs: WatchedDir[]
  addMutation: UseMutationResult<WatchedDir, Error, string>
  scanMutation: UseMutationResult<WatchedDir, Error, number>
  deleteMutation: UseMutationResult<void, Error, number>
  anyRunning?: boolean
}) {
  const physicalFoldersQuery = usePhysicalFolders()
  const folderGuardsQuery = useFolderGuards()
  const setFolderGuardMutation = useSetFolderGuard()

  const [addPath, setAddPath] = useState("")
  const [addError, setAddError] = useState("")

  const guardMap = folderGuardsQuery.data ? folderGuardsQuery.data.reduce((acc, g) => {
    acc[g.path] = g.guarded
    return acc
  }, {} as Record<string, boolean>) : {}

  const handleToggleGuard = (path: string, guarded: boolean) => {
    setFolderGuardMutation.mutate({ path, guarded })
  }

  const handleAdd = async () => {
    const p = addPath.trim()
    if (!p) return
    setAddError("")
    try {
      await addMutation.mutateAsync(p)
      setAddPath("")
    } catch (e: any) {
      setAddError(e.message)
    }
  }

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Enter") handleAdd()
  }

  const findWatchedDir = (folderPath: string): WatchedDir | undefined => {
    const normalized = folderPath.replace(/[\\/]+$/, "").replace(/\\/g, "/")
    return dirs.find(d => d.path.replace(/[\\/]+$/, "").replace(/\\/g, "/") === normalized)
  }

  return (
    <div class="file-tree">
      <div class="add-dir-form">
        <input
          type="text"
          placeholder="/path/to/directory"
          value={addPath}
          onInput={(e) => setAddPath((e.target as HTMLInputElement).value)}
          onKeyDown={handleKeyDown}
          disabled={addMutation.isPending}
        />
        <button class="btn btn-primary" onClick={handleAdd} disabled={addMutation.isPending || !addPath.trim()}>
          {addMutation.isPending ? "..." : "Add"}
        </button>
        <button class="btn" disabled title="Requires Tauri desktop app">
          Browse
        </button>
      </div>

      {addError && <div class="error-msg">{addError}</div>}

      <div class="folder-tree-scroll">
        {physicalFoldersQuery.isLoading ? (
          <div class="empty">Loading...</div>
        ) : physicalFoldersQuery.data && physicalFoldersQuery.data.length > 0 ? (
          <div class="folder-tree">
            {physicalFoldersQuery.data.map((root) => (
              <FileTreeFolderNode
                key={root.path}
                folder={root}
                depth={0}
                guardMap={guardMap}
                onToggleGuard={handleToggleGuard}
                scanMutation={scanMutation}
                deleteMutation={deleteMutation}
                findWatchedDir={findWatchedDir}
                anyRunning={anyRunning}
              />
            ))}
          </div>
        ) : (
          <div class="empty">No folders found</div>
        )}
      </div>
    </div>
  )
}

function FileTreeFolderNode({ folder, depth, guardMap, onToggleGuard, scanMutation, deleteMutation, findWatchedDir, anyRunning }: {
  folder: PhysicalFolder
  depth: number
  guardMap: Record<string, boolean>
  onToggleGuard: (path: string, guarded: boolean) => void
  scanMutation: UseMutationResult<WatchedDir, Error, number>
  deleteMutation: UseMutationResult<void, Error, number>
  findWatchedDir: (folderPath: string) => WatchedDir | undefined
  anyRunning?: boolean
}) {
  const [expanded, setExpanded] = useState(false)
  const hasChildren = folder.children && folder.children.length > 0
  const isGuarded = guardMap?.[folder.path]

  const toggleExpanded = () => {
    if (hasChildren) setExpanded(!expanded)
  }

  const toggleGuard = (e: MouseEvent) => {
    e.stopPropagation()
    onToggleGuard(folder.path, !isGuarded)
  }

  const watched = depth === 0 ? findWatchedDir(folder.path) : undefined

  const handleScan = (e: MouseEvent) => {
    e.stopPropagation()
    if (watched) scanMutation.mutate(watched.id)
  }

  const handleDelete = (e: MouseEvent) => {
    e.stopPropagation()
    if (watched) deleteMutation.mutate(watched.id)
  }

  return (
    <div class="folder-tree-node">
      <div
        class="folder-tree-row"
        style={{ "--depth": String(depth) } as any}
        onClick={toggleExpanded}
      >
        <span class={`folder-tree-toggle ${hasChildren ? "" : "invisible"}`}>
          {expanded ? "▾" : "▸"}
        </span>
        <span class="folder-tree-icon">{expanded ? "📂" : "📁"}</span>
        <span class="folder-tree-name">{folder.name}</span>
        {folder.file_count > 0 && (
          <span class="folder-tree-count">({folder.file_count})</span>
        )}
        <span
          class={`folder-guard-badge ${isGuarded ? "guarded" : "open"}`}
          onClick={toggleGuard}
          title={isGuarded ? "Guarded (click to unguard)" : "Open (click to guard)"}
        >
          {isGuarded ? "🔒" : "🔓"}
        </span>
        {watched && (
          <div class="dir-card-actions">
            <button class="btn btn-sm" disabled={anyRunning || scanMutation.isPending} onClick={handleScan}>
              Rescan
            </button>
            <button class="btn btn-sm btn-danger" disabled={deleteMutation.isPending} onClick={handleDelete}>
              Delete
            </button>
          </div>
        )}
      </div>

      {expanded && hasChildren && (
        <div class="folder-tree-children">
          {folder.children!.map((child) => (
            <FileTreeFolderNode
              key={child.path}
              folder={child}
              depth={depth + 1}
              guardMap={guardMap}
              onToggleGuard={onToggleGuard}
              scanMutation={scanMutation}
              deleteMutation={deleteMutation}
              findWatchedDir={findWatchedDir}
              anyRunning={anyRunning}
            />
          ))}
        </div>
      )}
    </div>
  )
}
