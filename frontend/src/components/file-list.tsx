import { type File as OwlFile, type FileListResponse } from "../api"
import { useExtractFile, useFileExtensions } from "../hooks/queries"
import { route } from "preact-router"

export interface FilterState {
  extension?: string
  processing_status?: string
  supported?: string
  sort: string
  order: string
  page: number
  limit: number
}

const PROCESSING_STATUSES = ["unprocessed", "queued", "processing", "processed", "stale", "failed"] as const

interface Props {
  data?: FileListResponse
  loading: boolean
  dirName: string | null
  filters: FilterState
  onFilterChange: (filters: FilterState) => void
  onRefresh: () => void
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i]
}

function formatTime(iso: string | null): string {
  if (!iso) return "-"
  const d = new Date(iso)
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

export function FileList({ data, loading, dirName, filters, onFilterChange, onRefresh }: Props) {
  const extractMutation = useExtractFile()
  const extQuery = useFileExtensions()

  const title = dirName ? `Files in ${dirName}` : "All Files"
  const files = data?.files ?? []
  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / filters.limit)
  const start = (filters.page - 1) * filters.limit + 1
  const end = Math.min(filters.page * filters.limit, total)

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

  const handleSort = (column: string) => {
    if (filters.sort === column) {
      onFilterChange({ ...filters, order: filters.order === "asc" ? "desc" : "asc" })
    } else {
      onFilterChange({ ...filters, sort: column, order: "desc", page: 1 })
    }
  }

  const handlePage = (delta: number) => {
    const newPage = filters.page + delta
    if (newPage >= 1 && newPage <= totalPages) {
      onFilterChange({ ...filters, page: newPage })
    }
  }

  const handleExtract = async (fileId: number) => {
    try {
      await extractMutation.mutateAsync(fileId)
      onRefresh()
    } catch (error) {
      console.error("Failed to extract file:", error)
    }
  }

  return (
    <div class="file-list">
      <div class="file-list-header">
        <h2>{title}</h2>
        <button class="btn" onClick={onRefresh}>Refresh</button>
      </div>

      <div class="file-filter-bar">
        <select
          class="filter-select"
          value={filters.extension ?? ""}
          onChange={(e) => onFilterChange({ ...filters, extension: (e.target as HTMLSelectElement).value || undefined, page: 1 })}
        >
          <option value="">All extensions</option>
          {extQuery.data?.map(ext => <option key={ext} value={ext}>{ext}</option>)}
        </select>

        <select
          class="filter-select"
          value={filters.processing_status ?? ""}
          onChange={(e) => onFilterChange({ ...filters, processing_status: (e.target as HTMLSelectElement).value || undefined, page: 1 })}
        >
          <option value="">All statuses</option>
          {PROCESSING_STATUSES.map(s => <option key={s} value={s}>{s}</option>)}
        </select>

        <select
          class="filter-select"
          value={filters.supported ?? ""}
          onChange={(e) => onFilterChange({ ...filters, supported: (e.target as HTMLSelectElement).value || undefined, page: 1 })}
        >
          <option value="">All files</option>
          <option value="true">Supported only</option>
          <option value="false">Unsupported only</option>
        </select>

        <button
          class="btn btn-sm"
          onClick={() => onFilterChange({ extension: undefined, processing_status: undefined, supported: undefined, sort: "indexed_at", order: "desc", page: 1, limit: filters.limit })}
        >
          Clear
        </button>
      </div>

      {loading ? (
        <div class="empty">Loading...</div>
      ) : files.length === 0 ? (
        <div class="empty">No files found</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th class="sortable" onClick={() => handleSort("name")}>
                Name {filters.sort === "name" && (filters.order === "asc" ? "▲" : "▼")}
              </th>
              <th class="sortable" onClick={() => handleSort("extension")}>
                Ext {filters.sort === "extension" && (filters.order === "asc" ? "▲" : "▼")}
              </th>
              <th class="sortable" onClick={() => handleSort("size")}>
                Size {filters.sort === "size" && (filters.order === "asc" ? "▲" : "▼")}
              </th>
              <th>Type</th>
              <th>File Status</th>
              <th>Processing</th>
              <th class="sortable" onClick={() => handleSort("indexed_at")}>
                Indexed {filters.sort === "indexed_at" && (filters.order === "asc" ? "▲" : "▼")}
              </th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {files.map((f: OwlFile) => (
              <tr key={f.id}>
                <td class="file-name" title={f.path}>
                  <a href={`/files/${f.id}`} class="file-link" onClick={(e) => { e.preventDefault(); route(`/files/${f.id}`) }}>{f.name}</a>
                </td>
                <td>{f.extension || "-"}</td>
                <td>{formatBytes(f.size)}</td>
                <td class="file-mime">{f.mime_type}</td>
                <td><span class={`status-badge status-${f.status}`}>{f.status}</span></td>
                <td>
                  <span class="status-badge-wrap">
                    <span class={`status-badge ${getStatusClass(f.processing_status)}`}>
                      {getStatusText(f)}
                    </span>
                    {f.processing_error && <span class="status-tooltip">{f.processing_error}</span>}
                  </span>
                </td>
                <td>{formatTime(f.indexed_at)}</td>
                <td>
                  <button
                    class="btn btn-sm"
                    disabled={
                      f.processing_status === "processing" ||
                      f.processing_status === "processed" ||
                      f.processing_status === "queued" ||
                      (f.processing_status === "unprocessed" && f.processable === false)
                    }
                    onClick={() => handleExtract(f.id)}
                  >
                    Extract
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div class="file-footer">
        <div class="file-count">{total > 0 ? `Showing ${start}-${end} of ${total}` : "No files"}</div>
        <div class="file-pagination">
          <button class="btn btn-sm" disabled={filters.page <= 1} onClick={() => handlePage(-1)}>Prev</button>
          <span class="pagination-info">{filters.page} / {totalPages || 1}</span>
          <button class="btn btn-sm" disabled={filters.page >= totalPages} onClick={() => handlePage(1)}>Next</button>
        </div>
      </div>
    </div>
  )
}