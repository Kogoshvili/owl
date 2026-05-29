import { useWatchedDirs, useAddWatchedDir, useScanDir, useDeleteDir, useExtractDir } from "../hooks/queries"
import { WatchedDirs } from "../components/watched-dirs"

export function IngestPage() {
  const dirsQuery = useWatchedDirs()
  const addMutation = useAddWatchedDir()
  const scanMutation = useScanDir()
  const deleteMutation = useDeleteDir()
  const extractMutation = useExtractDir()

  const dirs = dirsQuery.data ?? []

  return (
    <div class="page">
      <WatchedDirs
        dirs={dirs}
        selectedDirId={null}
        loading={dirsQuery.isLoading}
        addMutation={addMutation}
        scanMutation={scanMutation}
        deleteMutation={deleteMutation}
        extractMutation={extractMutation}
        onSelect={() => {}}
      />
    </div>
  )
}
