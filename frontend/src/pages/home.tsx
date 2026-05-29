import { useWatchedDirs, useAddWatchedDir, useScanDir, useDeleteDir, useExtractDir, useRunGuard, useExtractOrphans, useProcessingStats } from "../hooks/queries"
import { WatchedDirs } from "../components/watched-dirs"
import { FileTree } from "../components/file-tree"

export function HomePage() {
  const dirsQuery = useWatchedDirs()
  const addMutation = useAddWatchedDir()
  const scanMutation = useScanDir()
  const deleteMutation = useDeleteDir()
  const extractMutation = useExtractDir()
  const guardMutation = useRunGuard()
  const orphansMutation = useExtractOrphans()
  const statsQuery = useProcessingStats()

  const dirs = dirsQuery.data ?? []
  const stats = statsQuery.data

  return (
    <div class="page">
      <details class="dirs-section" open>
        <summary class="dirs-summary">
          <span class="dirs-summary-icon">▾</span>
          Watched Directories
          <span class="dirs-summary-count">{dirs.length}{dirsQuery.isLoading ? " …" : ""}</span>
        </summary>
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
      </details>

      <div class="files-pipeline-bar">
        <div class="pipeline-actions">
          <button
            class="btn btn-sm"
            onClick={() => guardMutation.mutate()}
            disabled={guardMutation.isPending}
          >
            {guardMutation.isPending ? "Guarding…" : "Guard Folders"}
          </button>
          <button
            class="btn btn-sm"
            onClick={() => orphansMutation.mutate()}
            disabled={orphansMutation.isPending}
          >
            {orphansMutation.isPending ? "Extracting…" : "Extract Orphans"}
          </button>
        </div>
        <div class="pipeline-status">
          {stats && (
            <>
              <span>{stats.guarded} guarded</span>
              <span class="text-muted">·</span>
              <span>{stats.open} open</span>
              <span class="text-muted">·</span>
              <span>{stats.extractable} extractable</span>
              <span class="text-muted">·</span>
              <span>{stats.queued} queued</span>
              <span class="text-muted">·</span>
              <span>{stats.processing} processing</span>
              <span class="text-muted">·</span>
              <span>{stats.processed} extracted</span>
              <span class="text-muted">·</span>
              <span>{stats.failed} failed</span>
            </>
          )}
        </div>
      </div>

      <FileTree />
    </div>
  )
}
