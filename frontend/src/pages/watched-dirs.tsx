import { useWatchedDirs, useAddWatchedDir, useScanDir, useDeleteDir } from "../hooks/queries"
import { WatchedDirsPanel } from "../components/watched-dirs"

export function WatchedDirsPage() {
  const dirsQuery = useWatchedDirs()
  const addMutation = useAddWatchedDir()
  const scanMutation = useScanDir()
  const deleteMutation = useDeleteDir()

  return (
    <div class="page">
      <WatchedDirsPanel
        dirs={dirsQuery.data ?? []}
        loading={dirsQuery.isLoading}
        addMutation={addMutation}
        scanMutation={scanMutation}
        deleteMutation={deleteMutation}
      />
    </div>
  )
}
