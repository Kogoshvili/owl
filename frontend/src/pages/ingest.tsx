import { useState } from "preact/hooks"
import { useWatchedDirs, useAllFiles, useFilesByDir, useAddWatchedDir, useScanDir, useDeleteDir, useExtractDir } from "../hooks/queries"
import { WatchedDirs } from "../components/watched-dirs"
import { FileList } from "../components/file-list"
import type { FilterState } from "../components/file-list"

export function IngestPage() {
  const [selectedDirId, setSelectedDirId] = useState<number | null>(null)
  const [selectedDirName, setSelectedDirName] = useState<string | null>(null)
  const [filters, setFilters] = useState<FilterState>({ extension: undefined, processing_status: undefined, sort: "indexed_at", order: "desc", page: 1, limit: 50 })

  const dirsQuery = useWatchedDirs()
  const allFilesQuery = useAllFiles(filters)
  const dirFilesQuery = useFilesByDir(selectedDirId, filters)

  const addMutation = useAddWatchedDir()
  const scanMutation = useScanDir()
  const deleteMutation = useDeleteDir()
  const extractMutation = useExtractDir()

  const dirs = dirsQuery.data ?? []
  const data = selectedDirId !== null ? dirFilesQuery.data : allFilesQuery.data
  const loading = selectedDirId !== null ? dirFilesQuery.isLoading : allFilesQuery.isLoading

  const handleSelect = (dirId: number | null) => {
    setSelectedDirId(dirId)
    setFilters({ ...filters, page: 1 })
    if (dirId !== null) {
      const dir = dirs.find((d) => d.id === dirId)
      setSelectedDirName(dir?.path ?? null)
    } else {
      setSelectedDirName(null)
    }
  }

  const handleRefresh = () => {
    allFilesQuery.refetch()
    dirFilesQuery.refetch()
  }

  return (
    <div class="page ingest-page">
      <aside class="sidebar">
        <WatchedDirs
          dirs={dirs}
          selectedDirId={selectedDirId}
          loading={dirsQuery.isLoading}
          addMutation={addMutation}
          scanMutation={scanMutation}
          deleteMutation={deleteMutation}
          extractMutation={extractMutation}
          onSelect={handleSelect}
        />
      </aside>
      <section class="content">
        <FileList
          data={data}
          loading={loading}
          dirName={selectedDirName}
          filters={filters}
          onFilterChange={setFilters}
          onRefresh={handleRefresh}
        />
      </section>
    </div>
  )
}
