import { useState, useEffect } from "preact/hooks"
import { type File as OwlFile, type PhysicalFolder, type WatchedDir } from "../api"
import type { UseMutationResult } from "@tanstack/preact-query"
import { useFileExtensions, useExtractFile, usePhysicalFolders, useFolderGuards, useSetFolderGuard } from "../hooks/queries"
import { useToast } from "../hooks/toast"
import { listPhysicalFolderFiles } from "../api"
import { route } from "preact-router"

export interface FileTreeFilterState {
  extension?: string
  processing_status?: string
  supported?: string
}

const PROCESSING_STATUSES = ["unprocessed", "queued", "processing", "processed", "stale", "failed"] as const

export function FileTree({ dirs, addMutation, scanMutation, deleteMutation, anyRunning }: {
  dirs: WatchedDir[]
  addMutation: UseMutationResult<WatchedDir, Error, string>
  scanMutation: UseMutationResult<WatchedDir, Error, number>
  deleteMutation: UseMutationResult<void, Error, number>
  anyRunning?: boolean
}) {
  const toast = useToast()
  const physicalFoldersQuery = usePhysicalFolders()
  const folderGuardsQuery = useFolderGuards()
  const setFolderGuardMutation = useSetFolderGuard()
  const extQuery = useFileExtensions()
  const extractMutation = useExtractFile()

  const [filters, setFilters] = useState<FileTreeFilterState>({})
  const [showFiles, setShowFiles] = useState(false)
  const [refreshKey, setRefreshKey] = useState(0)
  const [addPath, setAddPath] = useState("")
  const [addError, setAddError] = useState("")

  const guardMap = folderGuardsQuery.data ? folderGuardsQuery.data.reduce((acc, g) => {
    acc[g.path] = g.guarded
    return acc
  }, {} as Record<string, boolean>) : {}

  const handleToggleGuard = (path: string, guarded: boolean) => {
    setFolderGuardMutation.mutate({ path, guarded })
  }

  const handleExtract = async (fileId: number) => {
    try {
      await extractMutation.mutateAsync(fileId)
      setRefreshKey(n => n + 1)
    } catch (error: any) {
      toast.show({ type: "error", message: "Extract failed: " + error.message })
    }
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

      <div class="file-filter-bar">
        <select
          class="filter-select"
          value={filters.extension ?? ""}
          onChange={(e) => setFilters({ ...filters, extension: (e.target as HTMLSelectElement).value || undefined })}
        >
          <option value="">All extensions</option>
          {extQuery.data?.map(ext => <option key={ext} value={ext}>{ext}</option>)}
        </select>

        <select
          class="filter-select"
          value={filters.processing_status ?? ""}
          onChange={(e) => setFilters({ ...filters, processing_status: (e.target as HTMLSelectElement).value || undefined })}
        >
          <option value="">All statuses</option>
          {PROCESSING_STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
        </select>

        <select
          class="filter-select"
          value={filters.supported ?? ""}
          onChange={(e) => setFilters({ ...filters, supported: (e.target as HTMLSelectElement).value || undefined })}
        >
          <option value="">All files</option>
          <option value="true">Supported only</option>
          <option value="false">Unsupported only</option>
        </select>

        <button class="btn btn-sm" onClick={() => setFilters({})}>
          Clear
        </button>
        <button
          class={`btn btn-sm${showFiles ? " btn-primary" : ""}`}
          onClick={() => setShowFiles(!showFiles)}
          style="margin-left:auto"
        >
          {showFiles ? "Hide files" : "Show files"}
        </button>
        <button class="btn btn-sm" onClick={() => { physicalFoldersQuery.refetch(); folderGuardsQuery.refetch(); setRefreshKey(n => n + 1) }}>
          Refresh
        </button>
      </div>

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
                filters={filters}
                guardMap={guardMap}
                onToggleGuard={handleToggleGuard}
                onExtract={handleExtract}
                refreshKey={refreshKey}
                showFiles={showFiles}
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

function FileTreeFolderNode({ folder, depth, filters, guardMap, onToggleGuard, onExtract, refreshKey, showFiles, scanMutation, deleteMutation, findWatchedDir, anyRunning }: {
  folder: PhysicalFolder
  depth: number
  filters: FileTreeFilterState
  guardMap: Record<string, boolean>
  onToggleGuard: (path: string, guarded: boolean) => void
  onExtract: (fileId: number) => void
  refreshKey: number
  showFiles: boolean
  scanMutation: UseMutationResult<WatchedDir, Error, number>
  deleteMutation: UseMutationResult<void, Error, number>
  findWatchedDir: (folderPath: string) => WatchedDir | undefined
  anyRunning?: boolean
}) {
  const [expanded, setExpanded] = useState(false)
  const [files, setFiles] = useState<OwlFile[] | null>(null)
  const [loadingFiles, setLoadingFiles] = useState(false)
  const hasChildren = folder.children && folder.children.length > 0
  const canExpand = hasChildren || (showFiles && folder.file_count > 0)
  const isGuarded = guardMap?.[folder.path]

  const filteredFiles = files ? applyFileFilters(files, filters) : null

  useEffect(() => {
    if (expanded && showFiles && (files === null || refreshKey > 0) && folder.file_count > 0 && !loadingFiles) {
      setLoadingFiles(true)
      listPhysicalFolderFiles(folder.path).then((res) => {
        setFiles(res.files)
      }).catch(() => {
        setFiles([])
      }).finally(() => {
        setLoadingFiles(false)
      })
    }
  }, [expanded, showFiles, folder.path, folder.file_count, refreshKey])

  const toggleExpanded = () => {
    if (canExpand) setExpanded(!expanded)
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
        <span class={`folder-tree-toggle ${canExpand ? "" : "invisible"}`}>
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

      {expanded && (
        <div class="folder-tree-children">
          {hasChildren && folder.children!.map((child) => (
            <FileTreeFolderNode
              key={child.path}
              folder={child}
              depth={depth + 1}
              filters={filters}
              guardMap={guardMap}
              onToggleGuard={onToggleGuard}
              onExtract={onExtract}
              refreshKey={refreshKey}
              showFiles={showFiles}
                scanMutation={scanMutation}
                deleteMutation={deleteMutation}
                findWatchedDir={findWatchedDir}
                anyRunning={anyRunning}
              />
            ))}
          {showFiles && filteredFiles?.map((f) => (
            <FileTreeFileRow key={f.id} file={f} depth={depth + 1} onExtract={onExtract} />
          ))}
          {loadingFiles && (
            <div class="folder-tree-row" style={{ "--depth": String(depth + 1) } as any}>
              <span class="folder-tree-name">Loading files...</span>
            </div>
          )}
          {!loadingFiles && filteredFiles !== null && filteredFiles.length === 0 && !hasChildren && (
            <div class="folder-tree-row" style={{ "--depth": String(depth + 1) } as any}>
              <span class="folder-tree-name text-muted">(empty)</span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function FileTreeFileRow({ file, depth, onExtract }: { file: OwlFile; depth: number; onExtract: (fileId: number) => void }) {
  const getStatusClass = (status: string) => {
    switch (status) {
      case "unprocessed": return "status-unprocessed"
      case "queued": return "status-queued"
      case "processing": return "status-processing"
      case "processed": return "status-processed"
      case "stale": return "status-stale"
      case "failed": return "status-failed"
      default: return "status-other"
    }
  }

  const getStatusText = (file: OwlFile) => {
    if (file.processing_status === "unprocessed" && file.processable === false) return "unsupported"
    return file.processing_status
  }

  const canExtract = file.processing_status === "unprocessed" && file.processable !== false

  return (
    <div class="folder-tree-node">
      <div class="folder-tree-row file-row" style={{ "--depth": String(depth) } as any}>
        <span class="folder-tree-toggle invisible">▸</span>
        <span class="folder-tree-icon">📄</span>
        <span class="folder-tree-name">
          <a href={`/files/${file.id}`} class="file-link" onClick={(e) => { e.preventDefault(); route(`/files/${file.id}`) }}>{file.name}</a>
        </span>
        <span class="file-ext">{file.extension || "-"}</span>
        <span class="file-size">{formatBytes(file.size)}</span>
        <span class={`status-badge ${getStatusClass(file.processing_status)}`}>
          {getStatusText(file)}
        </span>
        {file.processing_error && <span class="file-error" title={file.processing_error}>⚠️</span>}
        <button
          class="btn btn-sm"
          disabled={!canExtract || file.processing_status === "processing" || file.processing_status === "queued"}
          onClick={(e) => { e.stopPropagation(); onExtract(file.id) }}
        >
          Extract
        </button>
      </div>
    </div>
  )
}

function applyFileFilters(files: OwlFile[], filters: FileTreeFilterState): OwlFile[] {
  return files.filter((f) => {
    if (filters.extension && f.extension !== filters.extension) return false
    if (filters.processing_status) {
      const status = f.processing_status === "unprocessed" && f.processable === false ? "unsupported" : f.processing_status
      if (status !== filters.processing_status) return false
    }
    if (filters.supported === "true") {
      if (f.processing_status === "unprocessed" && f.processable === false) return false
    }
    if (filters.supported === "false") {
      if (f.processing_status !== "unprocessed" || f.processable !== false) return false
    }
    return true
  })
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i]
}
