import { useState } from "preact/hooks"
import { useAllFiles, useFileExtensions } from "../hooks/queries"
import type { FilterState } from "./file-list"

interface Props {
  existingFileIds: Set<number>
  onAdd: (fileIds: number[]) => void
  onClose: () => void
  adding?: boolean
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i]
}

export function FilePickerDialog({ existingFileIds, onAdd, onClose, adding }: Props) {
  const [filters, setFilters] = useState<FilterState>({ sort: "name", order: "asc", page: 1, limit: 50 })
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [searchName, setSearchName] = useState("")
  const filesQuery = useAllFiles(filters)
  const extQuery = useFileExtensions()

  const files = filesQuery.data?.files ?? []
  const total = filesQuery.data?.total ?? 0
  const totalPages = Math.ceil(total / filters.limit)

  const filteredFiles = searchName
    ? files.filter((f) => f.name.toLowerCase().includes(searchName.toLowerCase()))
    : files

  const toggleSelect = (id: number) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const toggleAll = () => {
    const selectableIds = filteredFiles.filter((f) => !existingFileIds.has(f.id)).map((f) => f.id)
    const allSelected = selectableIds.every((id) => selected.has(id))
    if (allSelected) {
      setSelected(new Set())
    } else {
      setSelected(new Set(selectableIds))
    }
  }

  const handleAdd = () => {
    if (selected.size === 0) return
    onAdd(Array.from(selected))
  }

  const handlePage = (delta: number) => {
    setFilters({ ...filters, page: filters.page + delta })
  }

  return (
    <div class="dialog-overlay" onClick={onClose}>
      <div class="dialog" onClick={(e) => e.stopPropagation()}>
        <div class="dialog-header">
          <h3>Add Files to Folder</h3>
          <button class="btn btn-sm" onClick={onClose}>Close</button>
        </div>

        <div class="dialog-filters">
          <input
            type="text"
            placeholder="Search by name..."
            value={searchName}
            onInput={(e) => setSearchName((e.target as HTMLInputElement).value)}
            class="picker-search"
          />
          <select
            class="filter-select"
            value={filters.extension || ""}
            onChange={(e) => setFilters({ ...filters, extension: (e.target as HTMLSelectElement).value || undefined, page: 1 })}
          >
            <option value="">All extensions</option>
            {(extQuery.data ?? []).map((ext) => (
              <option key={ext} value={ext}>.{ext}</option>
            ))}
          </select>
        </div>

        <div class="dialog-toolbar">
          <button class="btn btn-sm" onClick={toggleAll}>
            {filteredFiles.filter((f) => !existingFileIds.has(f.id)).every((f) => selected.has(f.id)) && filteredFiles.length > 0 ? "Deselect All" : "Select All"}
          </button>
          <span class="picker-count">{selected.size} selected</span>
        </div>

        <div class="dialog-body">
          {filesQuery.isLoading && <div class="empty">Loading...</div>}
          {!filesQuery.isLoading && filteredFiles.length === 0 && <div class="empty">No files found</div>}
          <table class="picker-table">
            <thead>
              <tr>
                <th class="picker-check"></th>
                <th>Name</th>
                <th>Ext</th>
                <th>Size</th>
              </tr>
            </thead>
            <tbody>
              {filteredFiles.map((f) => {
                const isExisting = existingFileIds.has(f.id)
                return (
                  <tr key={f.id} class={isExisting ? "row-existing" : selected.has(f.id) ? "row-selected" : ""}>
                    <td class="picker-check">
                      <input
                        type="checkbox"
                        checked={isExisting || selected.has(f.id)}
                        disabled={isExisting}
                        onChange={() => toggleSelect(f.id)}
                      />
                    </td>
                    <td>
                      {f.name}
                      {isExisting && <span class="badge badge-existing">in folder</span>}
                    </td>
                    <td>{f.extension || "-"}</td>
                    <td>{formatBytes(f.size)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>

        {totalPages > 1 && (
          <div class="dialog-pagination">
            <button class="btn btn-sm" disabled={filters.page <= 1} onClick={() => handlePage(-1)}>Prev</button>
            <span>Page {filters.page} of {totalPages}</span>
            <button class="btn btn-sm" disabled={filters.page >= totalPages} onClick={() => handlePage(1)}>Next</button>
          </div>
        )}

        <div class="dialog-footer">
          <button class="btn" onClick={onClose}>Cancel</button>
          <button class="btn btn-primary" onClick={handleAdd} disabled={adding || selected.size === 0}>
            {adding ? "Adding..." : `Add ${selected.size} File${selected.size !== 1 ? "s" : ""}`}
          </button>
        </div>
      </div>
    </div>
  )
}
