import { FileTree } from "../components/file-tree"
import { useRunGuard, useExtractOrphans, useProcessingStats } from "../hooks/queries"

export function FilesPage() {
  const guardMutation = useRunGuard()
  const extractMutation = useExtractOrphans()
  const statsQuery = useProcessingStats()

  const stats = statsQuery.data

  return (
    <div class="page">
      <div class="files-pipeline-bar">
        <div class="pipeline-actions">
          <button
            class="btn btn-sm"
            onClick={() => guardMutation.mutate()}
            disabled={guardMutation.isPending}
          >
            {guardMutation.isPending ? "Guarding..." : "Guard Folders"}
          </button>
          <button
            class="btn btn-sm"
            onClick={() => extractMutation.mutate()}
            disabled={extractMutation.isPending}
          >
            {extractMutation.isPending ? "Extracting..." : "Extract Orphans"}
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
