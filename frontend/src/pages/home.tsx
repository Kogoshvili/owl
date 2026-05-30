import { useState, useEffect, useRef } from "preact/hooks"
import { desktopDir } from "@tauri-apps/api/path"
import { useWatchedDirs, useAddWatchedDir, useScanDir, useDeleteDir, useRunGuard, useExtractOrphans, useProcessingStats, useGuardStatus, useGenStatus, useScanStatus, useExtractStatus, useLlmStatus, useFolderSuggestions, useGenerateSuggestions, useDismissSuggestion, useRefineSuggestion, useRefineAllSuggestions, useAcceptSuggestion } from "../hooks/queries"
import { useToast } from "../hooks/toast"
import { FileTree } from "../components/file-tree"
import { ProgressBar } from "../components/progress-bar"
import { route } from "preact-router"
import type { FolderSuggestionDisplay, RunningStatus } from "../api"

function AcceptModal({ suggestion, onAccept, onClose }: {
  suggestion: FolderSuggestionDisplay
  onAccept: (destination: string, name: string) => void
  onClose: () => void
}) {
  const [dest, setDest] = useState("")
  const [folderName, setFolderName] = useState(suggestion.name)
  useEffect(() => { desktopDir().then(setDest).catch(() => {}) }, [])
  return (
    <div class="modal-overlay" onClick={onClose}>
      <div class="modal" onClick={(e) => e.stopPropagation()}>
        <h3>Materialize suggestion</h3>
        <p>This will move {suggestion.file_count} files into a new folder.</p>
        <form onSubmit={(e) => { e.preventDefault(); onAccept(dest || "", folderName) }}>
          <label>
            Folder name
            <input type="text" value={folderName} onInput={(e) => setFolderName((e.target as HTMLInputElement).value)} />
          </label>
          <label>
            Save to
            <input type="text" value={dest} onInput={(e) => setDest((e.target as HTMLInputElement).value)} placeholder="~/Owl-organized (default)" />
          </label>
          <div class="modal-actions">
            <button type="submit" class="btn btn-primary">Accept & Materialize</button>
            <button type="button" class="btn" onClick={onClose}>Cancel</button>
          </div>
        </form>
      </div>
    </div>
  )
}

function SuggestionCard({ suggestion, onDismiss, onRefine, onAccept, dismissing, refining, isMaterializing, llmAvailable }: {
  suggestion: FolderSuggestionDisplay
  onDismiss: () => void
  onRefine: () => void
  onAccept: () => void
  dismissing: boolean
  refining: boolean
  isMaterializing: boolean
  llmAvailable: boolean
}) {
  return (
    <div class="suggestion-card" onClick={() => route(`/suggestions/${suggestion.id}`)}>
      <div class="suggestion-card-header">
        <span class="suggestion-name">{suggestion.name}</span>
        {suggestion.confidence > 0 && (
          <span class="badge badge-confidence">{Math.round(suggestion.confidence * 100)}%</span>
        )}
      </div>
      {suggestion.description && <div class="suggestion-desc">{suggestion.description}</div>}
      <div class="suggestion-meta">
        <span>{suggestion.file_count} files</span>
      </div>
      {suggestion.preview && suggestion.preview.length > 0 && (
        <div class="suggestion-preview">
          {suggestion.preview.slice(0, 5).map((name, i) => (
            <span class="suggestion-preview-file" key={i}>{name}</span>
          ))}
          {suggestion.preview.length > 5 && <span class="suggestion-preview-more">+{suggestion.preview.length - 5} more</span>}
        </div>
      )}
      <div class="suggestion-actions" onClick={(e) => e.stopPropagation()}>
        <button class="btn btn-sm btn-primary" onClick={onAccept} disabled={isMaterializing}>
          {isMaterializing ? "Moving…" : "Accept"}
        </button>
        <button class="btn btn-sm" onClick={onRefine} disabled={refining || !llmAvailable} title={!llmAvailable ? "Requires LLM" : ""}>
          {refining ? "Refining…" : "Refine"}
        </button>
        <button class="btn btn-sm btn-danger" onClick={onDismiss} disabled={dismissing}>
          Dismiss
        </button>
      </div>
    </div>
  )
}

export function HomePage() {
  const toast = useToast()
  const dirsQuery = useWatchedDirs()
  const addMutation = useAddWatchedDir()
  const scanMutation = useScanDir()
  const deleteMutation = useDeleteDir()
  const guardMutation = useRunGuard()
  const orphansMutation = useExtractOrphans()
  const statsQuery = useProcessingStats()
  const llmQuery = useLlmStatus()
  const suggestionsQuery = useFolderSuggestions()
  const generateMutation = useGenerateSuggestions()
  const dismissMutation = useDismissSuggestion()
  const refineMutation = useRefineSuggestion()
  const refineAllMutation = useRefineAllSuggestions()
  const acceptMutation = useAcceptSuggestion()

  const scanStatusQ = useScanStatus()
  const extractStatusQ = useExtractStatus()
  const guardStatusQ = useGuardStatus()
  const genStatusQ = useGenStatus()

  const [refiningId, setRefiningId] = useState<number | null>(null)
  const [acceptingId, setAcceptingId] = useState<number | null>(null)
  const [strategy, setStrategy] = useState("content_tfidf")

  const dirs = dirsQuery.data ?? []
  const stats = statsQuery.data
  const llmAvailable = llmQuery.data?.llm_available ?? false
  const suggestions = suggestionsQuery.data ? Object.values(suggestionsQuery.data) : []

  const anyRunning = !!(scanStatusQ.data?.running || extractStatusQ.data?.running || guardStatusQ.data?.running || genStatusQ.data?.running)

  function useOpToast(status: RunningStatus | undefined, label: string) {
    const prev = useRef({ running: false, completed: "" })
    useEffect(() => {
      if (!status) { prev.current.running = false; return }
      if (prev.current.running && !status.running && status.completed_at && status.completed_at !== prev.current.completed) {
        prev.current.completed = status.completed_at
        if (status.error) {
          toast.show({ type: "error", message: `${label} failed: ${status.error}` })
        } else {
          toast.show({ type: "success", message: status.message || `${label} complete` })
        }
      }
      prev.current.running = status.running
    }, [status])
  }

  useOpToast(scanStatusQ.data, "Scan")
  useOpToast(extractStatusQ.data, "Extraction")
  useOpToast(guardStatusQ.data, "Guard")
  useOpToast(genStatusQ.data, "Generation")

  const handleGuard = () => {
    guardMutation.mutate(undefined, {
      onError: (err: Error) => { toast.show({ type: "error", message: err.message }) },
    })
  }

  const handleExtract = () => {
    orphansMutation.mutate(undefined, {
      onError: (err: Error) => { toast.show({ type: "error", message: err.message }) },
    })
  }

  const handleGenerate = async () => {
    try {
      await generateMutation.mutateAsync({ strategy })
    } catch (e: any) {
      toast.show({ type: "error", message: e.message })
    }
  }

  const handleDismiss = async (id: number) => {
    try {
      await dismissMutation.mutateAsync(id)
      toast.show({ type: "success", message: "Suggestion dismissed" })
    } catch (e: any) { toast.show({ type: "error", message: e.message }) }
  }

  const handleRefine = async (id: number) => {
    if (!llmAvailable) { toast.show({ type: "error", message: "LLM not available for refinement" }); return }
    setRefiningId(id)
    try {
      await refineMutation.mutateAsync(id)
      toast.show({ type: "success", message: "Suggestion refined" })
    } catch (e: any) { toast.show({ type: "error", message: e.message }) }
    finally { setRefiningId(null) }
  }

  const handleAccept = async (id: number, destination: string, name: string) => {
    setAcceptingId(id)
    try {
      const result = await acceptMutation.mutateAsync({ id, destination, name })
      toast.show({ type: "success", message: `Moved ${result.moved} files to ${result.folder_path}` })
    } catch (e: any) { toast.show({ type: "error", message: e.message }) }
    finally { setAcceptingId(null) }
  }

  const handleDismissAll = async () => {
    for (const s of suggestions) {
      try { await dismissMutation.mutateAsync(s.id) }
      catch (e: any) { toast.show({ type: "error", message: `Failed to dismiss #${s.id}: ${e.message}` }) }
    }
    toast.show({ type: "success", message: "All suggestions dismissed" })
  }

  const handleRefineAll = async () => {
    if (!llmAvailable) { toast.show({ type: "error", message: "LLM not available for refinement" }); return }
    try {
      await refineAllMutation.mutateAsync()
      toast.show({ type: "info", message: "Refining all suggestions…" })
    } catch (e: any) { toast.show({ type: "error", message: e.message }) }
  }

  return (
    <div class="page">
      {!llmAvailable && (
        <div class="llm-banner">
          LLM not available. Folder guard, refinement, and embeddings strategy are disabled.
          {llmQuery.isLoading && " Checking…"}
        </div>
      )}

      <div class="files-pipeline-bar">
        <div class="pipeline-actions">
          <button class="btn btn-sm" onClick={handleGuard} disabled={anyRunning || !llmAvailable} title={!llmAvailable ? "Requires LLM" : ""}>
            {guardStatusQ.data?.running ? "Guarding…" : "Guard Folders"}
          </button>
          <button class="btn btn-sm" onClick={handleExtract} disabled={anyRunning}>
            {extractStatusQ.data?.running ? "Extracting…" : "Extract Orphans"}
          </button>
          <select class="strategy-select" value={strategy} onChange={(e) => setStrategy((e.target as HTMLSelectElement).value)}>
            <option value="content_tfidf">content_tfidf</option>
            <option value="embeddings" disabled={!llmAvailable}>embeddings{!llmAvailable ? " (requires LLM)" : ""}</option>
          </select>
          <button class="btn btn-sm btn-primary" onClick={handleGenerate} disabled={anyRunning}>
            {genStatusQ.data?.running ? "Generating…" : "Generate"}
          </button>
        </div>
        <div class="pipeline-status">
          {stats && (
            <>
              <span>{stats.guarded} guarded</span><span class="text-muted">·</span>
              <span>{stats.open} open</span><span class="text-muted">·</span>
              <span>{stats.extractable} extractable</span><span class="text-muted">·</span>
              <span>{stats.queued} queued</span><span class="text-muted">·</span>
              <span>{stats.processing} processing</span><span class="text-muted">·</span>
              <span>{stats.processed} extracted</span><span class="text-muted">·</span>
              <span>{stats.failed} failed</span>
            </>
          )}
          {statsQuery.isLoading && <span>Loading stats…</span>}
          {statsQuery.isError && <span class="error-msg">Failed to load stats</span>}
        </div>
      </div>

      <ProgressBar running={scanStatusQ.data?.running ?? false} progress={scanStatusQ.data?.progress} total={scanStatusQ.data?.total} message={scanStatusQ.data?.message} />
      <ProgressBar running={extractStatusQ.data?.running ?? false} progress={extractStatusQ.data?.progress} total={extractStatusQ.data?.total} message={extractStatusQ.data?.message} />
      <ProgressBar running={guardStatusQ.data?.running ?? false} progress={guardStatusQ.data?.progress} total={guardStatusQ.data?.total} message={guardStatusQ.data?.message} />
      <ProgressBar running={genStatusQ.data?.running ?? false} progress={genStatusQ.data?.progress} total={genStatusQ.data?.total} message={genStatusQ.data?.message} />

      <div class="section">
        <FileTree
          dirs={dirs}
          addMutation={addMutation}
          scanMutation={scanMutation}
          deleteMutation={deleteMutation}
          anyRunning={anyRunning}
        />
      </div>

      <div class="section">
        <h2 class="section-title">Suggestions</h2>

        {suggestions.length > 0 && (
          <div class="suggestions-actions" style="margin-bottom:8px">
            <button class="btn btn-sm" onClick={handleRefineAll} disabled={refineAllMutation.isPending || !llmAvailable}>Refine All</button>
            <button class="btn btn-sm btn-danger" onClick={handleDismissAll} disabled={dismissMutation.isPending}>Dismiss All</button>
          </div>
        )}

        {suggestionsQuery.isLoading && <div class="empty">Loading…</div>}
        {suggestionsQuery.isError && <div class="error-msg">Failed to load suggestions</div>}
        {!suggestionsQuery.isLoading && !suggestionsQuery.isError && suggestions.length === 0 && (
          <div class="empty">No suggestions. Click Generate above to create some.</div>
        )}

        {suggestions.length > 0 && (
          <div class="suggestion-grid">
            {suggestions.map((s) => (
              <SuggestionCard
                key={s.id}
                suggestion={s}
                onDismiss={() => handleDismiss(s.id)}
                onRefine={() => handleRefine(s.id)}
                onAccept={() => setAcceptingId(s.id)}
                dismissing={dismissMutation.isPending}
                refining={refiningId === s.id}
                isMaterializing={acceptMutation.isPending && acceptingId === s.id}
                llmAvailable={llmAvailable}
              />
            ))}
          </div>
        )}
      </div>

      {acceptingId !== null && (
        <AcceptModal
          suggestion={suggestions.find(s => s.id === acceptingId)!}
          onAccept={(dest, name) => handleAccept(acceptingId, dest, name)}
          onClose={() => setAcceptingId(null)}
        />
      )}
    </div>
  )
}
