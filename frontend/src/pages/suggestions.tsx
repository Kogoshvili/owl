import { useState, useEffect, useRef } from "preact/hooks"
import { useFolderSuggestions, useGenerateSuggestions, useDismissSuggestion, useRefineSuggestion, useRefineAllSuggestions, useAcceptSuggestion, useLlmStatus } from "../hooks/queries"
import { useToast } from "../hooks/toast"
import { route } from "preact-router"
import type { FolderSuggestionDisplay, MaterializeResult } from "../api"
import { getGenerationStatus } from "../api"

function AcceptModal({ suggestion, onAccept, onClose }: {
  suggestion: FolderSuggestionDisplay
  onAccept: (destination: string) => void
  onClose: () => void
}) {
  const [dest, setDest] = useState("")

  const handleSubmit = (e: Event) => {
    e.preventDefault()
    onAccept(dest || "")
  }

  return (
    <div class="modal-overlay" onClick={onClose}>
      <div class="modal" onClick={(e) => e.stopPropagation()}>
        <h3>Accept: {suggestion.name}</h3>
        <p>This will move {suggestion.file_count} files into a new folder.</p>
        <form onSubmit={handleSubmit}>
          <label>
            Destination base path
            <input
              type="text"
              value={dest}
              onInput={(e) => setDest((e.target as HTMLInputElement).value)}
              placeholder="~/Owl-organized (default)"
            />
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

function MaterializeStatus({ result }: { result: MaterializeResult }) {
  return (
    <div class="materialize-status">
      <p>Moved {result.moved} files to <code>{result.folder_path}</code></p>
      {result.failed && result.failed.length > 0 && (
        <p class="error-msg">Failed to move: {result.failed.join(", ")}</p>
      )}
    </div>
  )
}

function GenerationProgress() {
  const [status, setStatus] = useState<{ stage?: string; message?: string; progress?: number; total?: number } | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval>>()

  useEffect(() => {
    intervalRef.current = setInterval(async () => {
      try {
        const s = await getGenerationStatus()
        if (s.active) {
          setStatus(s)
        } else {
          setStatus(null)
          if (intervalRef.current) clearInterval(intervalRef.current)
        }
      } catch {
        setStatus(null)
      }
    }, 2000)
    return () => { if (intervalRef.current) clearInterval(intervalRef.current) }
  }, [])

  if (!status) return null

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

export function SuggestionsPage() {
  const toast = useToast()
  const suggestionsQuery = useFolderSuggestions()
  const generateMutation = useGenerateSuggestions()
  const dismissMutation = useDismissSuggestion()
  const refineMutation = useRefineSuggestion()
  const refineAllMutation = useRefineAllSuggestions()
  const acceptMutation = useAcceptSuggestion()
  const llmQuery = useLlmStatus()
  const [generating, setGenerating] = useState(false)
  const [refiningId, setRefiningId] = useState<number | null>(null)
  const [acceptingId, setAcceptingId] = useState<number | null>(null)
  const [acceptResult, setAcceptResult] = useState<MaterializeResult | null>(null)
  const [strategy, setStrategy] = useState("content_tfidf")

  const suggestions = suggestionsQuery.data ? Object.values(suggestionsQuery.data) : []
  const llmAvailable = llmQuery.data?.llm_available ?? false

  // Stop polling generation status on unmount or when done
  const [genActive, setGenActive] = useState(false)

  const handleGenerate = async () => {
    setGenerating(true)
    setGenActive(true)
    try {
      await generateMutation.mutateAsync({ strategy })
      toast.show({ type: "info", message: "Generating suggestions…" })
    } catch (e: any) {
      toast.show({ type: "error", message: e.message })
    }
  }

  // Poll generation status and stop when complete
  useEffect(() => {
    if (!genActive) return
    const interval = setInterval(async () => {
      try {
        const s = await getGenerationStatus()
        if (!s.active) {
          clearInterval(interval)
          setGenerating(false)
          setGenActive(false)
          suggestionsQuery.refetch()
          if (s.completed_at) {
            toast.show({ type: "success", message: s.message || "Suggestions generated" })
          }
        }
      } catch {
        clearInterval(interval)
        setGenerating(false)
        setGenActive(false)
      }
    }, 2000)
    return () => clearInterval(interval)
  }, [genActive, suggestionsQuery])

  const handleDismiss = async (id: number) => {
    try {
      await dismissMutation.mutateAsync(id)
      toast.show({ type: "success", message: "Suggestion dismissed" })
    } catch (e: any) {
      toast.show({ type: "error", message: e.message })
    }
  }

  const handleRefine = async (id: number) => {
    if (!llmAvailable) {
      toast.show({ type: "error", message: "LLM not available for refinement" })
      return
    }
    setRefiningId(id)
    try {
      await refineMutation.mutateAsync(id)
      toast.show({ type: "success", message: "Suggestion refined" })
    } catch (e: any) {
      toast.show({ type: "error", message: e.message })
    } finally {
      setRefiningId(null)
    }
  }

  const handleAccept = async (id: number, destination: string) => {
    setAcceptResult(null)
    try {
      const result = await acceptMutation.mutateAsync({ id, destination })
      setAcceptResult(result)
      toast.show({ type: "success", message: `Moved ${result.moved} files to ${result.folder_path}` })
    } catch (e: any) {
      toast.show({ type: "error", message: e.message })
    } finally {
      setAcceptingId(null)
    }
  }

  const handleDismissAll = async () => {
    for (const s of suggestions) {
      try {
        await dismissMutation.mutateAsync(s.id)
      } catch (e: any) {
        toast.show({ type: "error", message: `Failed to dismiss #${s.id}: ${e.message}` })
      }
    }
    toast.show({ type: "success", message: "All suggestions dismissed" })
  }

  const handleRefineAll = async () => {
    if (!llmAvailable) {
      toast.show({ type: "error", message: "LLM not available for refinement" })
      return
    }
    try {
      await refineAllMutation.mutateAsync()
      toast.show({ type: "info", message: "Refining all suggestions…" })
    } catch (e: any) {
      toast.show({ type: "error", message: e.message })
    }
  }

  return (
    <div class="page suggestions-page">
      {!llmAvailable && (
        <div class="llm-banner">
          LLM not available. Refine and strategy selection disabled.
          {llmQuery.isLoading && " Checking…"}
        </div>
      )}

      <div class="suggestions-header">
        <h2>Suggestions</h2>
        <div class="suggestions-actions">
          <select class="strategy-select" value={strategy} onChange={(e) => setStrategy((e.target as HTMLSelectElement).value)}>
            <option value="content_tfidf">content_tfidf</option>
            <option value="embeddings" disabled={!llmAvailable}>
              embeddings{!llmAvailable ? " (requires LLM)" : ""}
            </option>
          </select>
          <button
            class="btn btn-sm btn-primary"
            onClick={handleGenerate}
            disabled={generating}
          >
            {generating ? "Generating…" : "Generate"}
          </button>
          {suggestions.length > 0 && (
            <>
              <button class="btn btn-sm" onClick={handleRefineAll} disabled={refineAllMutation.isPending || !llmAvailable}>
                Refine All
              </button>
              <button class="btn btn-sm btn-danger" onClick={handleDismissAll} disabled={dismissMutation.isPending}>
                Dismiss All
              </button>
            </>
          )}
        </div>
      </div>

      {(generating || genActive) && <GenerationProgress />}

      {acceptResult && <MaterializeStatus result={acceptResult} />}

      {suggestionsQuery.isLoading && <div class="empty">Loading…</div>}
      {suggestionsQuery.isError && <div class="error-msg">Failed to load suggestions</div>}
      {!suggestionsQuery.isLoading && !suggestionsQuery.isError && suggestions.length === 0 && (
        <div class="empty">No suggestions. Click Generate to create some.</div>
      )}

      <div class="suggestion-list">
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

      {acceptingId !== null && (
        <AcceptModal
          suggestion={suggestions.find(s => s.id === acceptingId)!}
          onAccept={(dest) => handleAccept(acceptingId, dest)}
          onClose={() => setAcceptingId(null)}
        />
      )}
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
    <div class={`suggestion-card${suggestion.materialized_at ? " materialized" : ""}`} onClick={() => route(`/suggestions/${suggestion.id}`)}>
      <div class="suggestion-card-header">
        <span class="suggestion-name">{suggestion.name}</span>
        {suggestion.materialized_at && (
          <span class="badge badge-materialized">Materialized</span>
        )}
        {suggestion.confidence > 0 && !suggestion.materialized_at && (
          <span class="badge badge-confidence">{Math.round(suggestion.confidence * 100)}%</span>
        )}
      </div>
      {suggestion.description && (
        <div class="suggestion-desc">{suggestion.description}</div>
      )}
      <div class="suggestion-meta">
        <span>{suggestion.file_count} files</span>
        {suggestion.materialized_path && (
          <span class="suggestion-path">→ {suggestion.materialized_path}</span>
        )}
      </div>
      {suggestion.preview && suggestion.preview.length > 0 && (
        <div class="suggestion-preview">
          {suggestion.preview.slice(0, 5).map((name, i) => (
            <span class="suggestion-preview-file" key={i}>{name}</span>
          ))}
          {suggestion.preview.length > 5 && (
            <span class="suggestion-preview-more">+{suggestion.preview.length - 5} more</span>
          )}
        </div>
      )}
      <div class="suggestion-actions" onClick={(e) => e.stopPropagation()}>
        {!suggestion.materialized_at && (
          <button class="btn btn-sm btn-primary" onClick={onAccept} disabled={isMaterializing}>
            {isMaterializing ? "Moving…" : "Accept"}
          </button>
        )}
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
