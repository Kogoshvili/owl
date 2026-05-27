import { useState } from "preact/hooks"
import { useAllFiles } from "../hooks/queries"
import { FileList } from "../components/file-list"
import type { FilterState } from "../components/file-list"

export function FilesPage() {
  const [filters, setFilters] = useState<FilterState>({ extension: undefined, processing_status: undefined, sort: "indexed_at", order: "desc", page: 1, limit: 50 })
  const filesQuery = useAllFiles(filters)

  return (
    <div class="page">
      <FileList
        data={filesQuery.data}
        loading={filesQuery.isLoading}
        dirName={null}
        filters={filters}
        onFilterChange={setFilters}
        onRefresh={() => filesQuery.refetch()}
      />
    </div>
  )
}
