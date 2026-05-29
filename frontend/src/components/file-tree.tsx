import { useState, useEffect } from "preact/hooks"
import { type File as OwlFile, type PhysicalFolder } from "../api"
import { useFileExtensions, useExtractFile, usePhysicalFolders, useFolderGuards, useSetFolderGuard } from "../hooks/queries"
import { listPhysicalFolderFiles } from "../api"
import { route } from "preact-router"

export interface FileTreeFilterState {
  extension?: string
  processing_status?: string
  supported?: string
}

const PROCESSING_STATUSES = ["unprocessed", "queued", "processing", "processed", "stale", "failed"] as const

export function FileTree() {
  const physicalFoldersQuery = usePhysicalFolders()
  const folderGuardsQuery = useFolderGuards()
  const setFolderGuardMutation = useSetFolderGuard()
  const extQuery = useFileExtensions()
  const extractMutation = useExtractFile()

  const [filters, setFilters] = useState<FileTreeFilterState>({})
  const [refreshKey, setRefreshKey] = useState(0)

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
    } catch (error) {
      console.error("Failed to extract file:", error)
    }
  }

  return (
    <div class="file-tree">
      <div class="file-list-header">
        <h2>Files</h2>
        <button class="btn" onClick={() => { physicalFoldersQuery.refetch(); folderGuardsQuery.refetch(); setRefreshKey(n => n + 1) }}>Refresh</button>
      </div>

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
      </div>

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
            />
          ))}
        </div>
      ) : (
        <div class="empty">No folders found</div>
      )}
    </div>
  )
}

function FileTreeFolderNode({ folder, depth, filters, guardMap, onToggleGuard, onExtract, refreshKey }: {
  folder: PhysicalFolder
  depth: number
  filters: FileTreeFilterState
  guardMap: Record<string, boolean>
  onToggleGuard: (path: string, guarded: boolean) => void
  onExtract: (fileId: number) => void
  refreshKey: number
}) {
  const [expanded, setExpanded] = useState(false)
  const [files, setFiles] = useState<OwlFile[] | null>(null)
  const [loadingFiles, setLoadingFiles] = useState(false)
  const hasChildren = folder.children && folder.children.length > 0
  const canExpand = hasChildren || folder.file_count > 0
  const isGuarded = guardMap?.[folder.path]

  const filteredFiles = files ? applyFileFilters(files, filters) : null

  useEffect(() => {
    if (expanded && (files === null || refreshKey > 0) && folder.file_count > 0 && !loadingFiles) {
      setLoadingFiles(true)
      listPhysicalFolderFiles(folder.path).then((res) => {
        setFiles(res.files)
      }).catch(() => {
        setFiles([])
      }).finally(() => {
        setLoadingFiles(false)
      })
    }
  }, [expanded, folder.path, folder.file_count, refreshKey])

  const toggleExpanded = () => {
    if (canExpand) setExpanded(!expanded)
  }

  const toggleGuard = (e: MouseEvent) => {
    e.stopPropagation()
    onToggleGuard(folder.path, !isGuarded)
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
        <span class="folder-tree-icon">{canExpand ? (expanded ? "📂" : "📁") : "📄"}</span>
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
            />
          ))}
          {filteredFiles?.map((f) => (
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
