import { useState } from "preact/hooks"
import { useWatchedDirs, useAllFiles, useFilesByDir, useAddWatchedDir, useScanDir, useDeleteDir } from "../hooks/queries"
import { WatchedDirs } from "../components/watched-dirs"
import { FileList } from "../components/file-list"

export function DashboardPage() {
  const [selectedDirId, setSelectedDirId] = useState<number | null>(null)
  const [selectedDirName, setSelectedDirName] = useState<string | null>(null)

  const dirsQuery = useWatchedDirs()
  const allFilesQuery = useAllFiles()
  const dirFilesQuery = useFilesByDir(selectedDirId)

  const addMutation = useAddWatchedDir()
  const scanMutation = useScanDir()
  const deleteMutation = useDeleteDir()

  const dirs = dirsQuery.data ?? []
  const files = selectedDirId !== null
    ? (dirFilesQuery.data ?? [])
    : (allFilesQuery.data ?? [])
  const loading = selectedDirId !== null ? dirFilesQuery.isLoading : allFilesQuery.isLoading

  const handleSelect = (dirId: number | null) => {
    setSelectedDirId(dirId)
    if (dirId !== null) {
      const dir = dirs.find((d) => d.id === dirId)
      setSelectedDirName(dir?.path ?? null)
    } else {
      setSelectedDirName(null)
    }
  }

  return (
    <div class="page dashboard-page">
      <aside class="sidebar">
        <WatchedDirs
          dirs={dirs}
          selectedDirId={selectedDirId}
          loading={dirsQuery.isLoading}
          addMutation={addMutation}
          scanMutation={scanMutation}
          deleteMutation={deleteMutation}
          onSelect={handleSelect}
        />
      </aside>
      <section class="content">
        <FileList
          files={files}
          loading={loading}
          dirName={selectedDirName}
          onRefresh={() => {
            allFilesQuery.refetch()
            dirFilesQuery.refetch()
          }}
        />
      </section>
    </div>
  )
}
