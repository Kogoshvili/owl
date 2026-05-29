import { useState } from "preact/hooks"
import { useWatchedDirs, useAddWatchedDir, useScanDir, useDeleteDir, useExtractDir, useRunGuard, useExtractOrphans, useProcessingStats, useGuardStatus, useLlmStatus } from "../hooks/queries"
import { useToast } from "../hooks/toast"
import { WatchedDirs } from "../components/watched-dirs"
import { FileTree } from "../components/file-tree"

function GuardProgress({ status }: { status: { stage?: string; progress?: number; total?: number; message?: string } }) {
  const total = status.total ?? 1
  const pct = total > 0 ? Math.min(100, Math.round((status.progress ?? 0) / total * 100)) : 0
  return (
    <div class="operation-progress">
      <div class="progress-bar">
        <div class="progress-fill" style={{ width: `${pct}%` }} />
      </div>
      <span class="progress-text">{status.message || status.stage}</span>
    </div>
  )
}

export function HomePage() {
  const toast = useToast()
  const dirsQuery = useWatchedDirs()
  const addMutation = useAddWatchedDir()
  const scanMutation = useScanDir()
  const deleteMutation = useDeleteDir()
  const extractMutation = useExtractDir()
  const guardMutation = useRunGuard()
  const orphansMutation = useExtractOrphans()
  const statsQuery = useProcessingStats()
  const llmQuery = useLlmStatus()

  const [guardJustStarted, setGuardJustStarted] = useState(false)
  const guardStatusQuery = useGuardStatus(guardJustStarted)
  const guardStatus = guardStatusQuery.data

  // Stop polling when guard completes or errors
  if (guardStatus && !guardStatus.running && guardJustStarted) {
    setGuardJustStarted(false)
    if (guardStatus.error) {
      toast.show({ type: "error", message: "Guard failed: " + guardStatus.error })
    } else {
      toast.show({ type: "success", message: guardStatus.message || "Guard complete" })
    }
  }

  const dirs = dirsQuery.data ?? []
  const stats = statsQuery.data
  const llmAvailable = llmQuery.data?.llm_available ?? false

  const handleGuard = () => {
    setGuardJustStarted(true)
    guardMutation.mutate(undefined, {
      onError: (err: Error) => {
        setGuardJustStarted(false)
        toast.show({ type: "error", message: err.message })
      },
    })
  }

  const handleExtract = () => {
    orphansMutation.mutate(undefined, {
      onError: (err: Error) => {
        toast.show({ type: "error", message: err.message })
      },
    })
    toast.show({ type: "info", message: "Extraction started" })
  }

  return (
    <div class="page">
      {!llmAvailable && (
        <div class="llm-banner">
          LLM not available. Folder guard and refinement are disabled.
          {llmQuery.isLoading && " Checking..."}
        </div>
      )}

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
            onClick={handleGuard}
            disabled={guardMutation.isPending || guardJustStarted || !llmAvailable}
            title={!llmAvailable ? "Requires LLM" : ""}
          >
            {guardMutation.isPending || guardJustStarted ? "Guarding…" : "Guard Folders"}
          </button>
          <button
            class="btn btn-sm"
            onClick={handleExtract}
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
          {statsQuery.isLoading && <span>Loading stats…</span>}
          {statsQuery.isError && <span class="error-msg">Failed to load stats</span>}
        </div>
      </div>

      {guardStatus?.running && <GuardProgress status={guardStatus} />}

      <FileTree />
    </div>
  )
}
