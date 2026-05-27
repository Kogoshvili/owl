import { useState } from "preact/hooks"
import { useTags, useTagFilesList } from "../hooks/queries"
import { route } from "preact-router"
import type { FilterState } from "../components/file-list"
import type { File } from "../api"

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i]
}

export function TagDetailPage({ id }: { id?: string }) {
  const tagId = parseInt(id || "0", 10)
  const tagsQuery = useTags()
  const tag = tagsQuery.data?.find((t) => t.id === tagId)

  const [filters, setFilters] = useState<FilterState>({
    sort: "name",
    order: "asc",
    page: 1,
    limit: 50,
  })

  const filesQuery = useTagFilesList(tagId, filters)

  if (tagsQuery.isLoading) return <div class="page"><div class="empty">Loading...</div></div>
  if (!tag) return <div class="page"><div class="empty">Tag not found</div></div>

  const files = filesQuery.data?.files ?? []
  const total = filesQuery.data?.total ?? 0

  return (
    <div class="page tag-detail-page">
      <button class="btn btn-back" onClick={() => route("/tags")}>&larr; Back</button>

      <div class="tag-detail-header">
        <div class="tag-detail-title-row">
          <h2>{tag.name}</h2>
          <span class={`badge ${tag.source === "auto" ? "badge-auto" : "badge-manual"}`}>
            {tag.source}
          </span>
        </div>
        <div class="tag-detail-meta">{tag.file_count} file{tag.file_count !== 1 ? "s" : ""}</div>
      </div>

      <div class="tag-files-section">
        <h3>Files ({total})</h3>

        {filesQuery.isLoading && <div class="empty">Loading...</div>}
        {!filesQuery.isLoading && files.length === 0 && (
          <div class="empty">No files with this tag</div>
        )}

        {files.length > 0 && (
          <table class="folder-files-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Extension</th>
                <th>Size</th>
              </tr>
            </thead>
            <tbody>
              {files.map((f: File) => (
                <tr key={f.id}>
                  <td>
                    <a href={`/files/${f.id}`} onClick={(e) => { e.preventDefault(); route(`/files/${f.id}`) }}>
                      {f.name}
                    </a>
                  </td>
                  <td>{f.extension || "-"}</td>
                  <td>{formatBytes(f.size)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {total > filters.limit && (
          <div class="file-footer">
            <div class="file-pagination">
              <button
                class="btn btn-sm"
                disabled={filters.page <= 1}
                onClick={() => setFilters({ ...filters, page: filters.page - 1 })}
              >
                Prev
              </button>
              <span class="pagination-info">
                Page {filters.page} of {Math.ceil(total / filters.limit)}
              </span>
              <button
                class="btn btn-sm"
                disabled={filters.page >= Math.ceil(total / filters.limit)}
                onClick={() => setFilters({ ...filters, page: filters.page + 1 })}
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
