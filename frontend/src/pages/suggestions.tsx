import { useState } from "preact/hooks"
import { useFolderSuggestions, useGenerateSuggestions, useDismissSuggestion, useRefineSuggestion, useRefineAllSuggestions, useAcceptSuggestion } from "../hooks/queries"
import { route } from "preact-router"
import type { FolderSuggestionDisplay, MaterializeResult } from "../api"

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

export function SuggestionsPage() {
  const suggestionsQuery = useFolderSuggestions()
  const generateMutation = useGenerateSuggestions()
  const dismissMutation = useDismissSuggestion()
  const refineMutation = useRefineSuggestion()
  const refineAllMutation = useRefineAllSuggestions()
  const acceptMutation = useAcceptSuggestion()
  const [generating, setGenerating] = useState(false)
  const [refiningId, setRefiningId] = useState<number | null>(null)
  const [acceptingId, setAcceptingId] = useState<number | null>(null)
  const [acceptResult, setAcceptResult] = useState<MaterializeResult | null>(null)

  const suggestions = suggestionsQuery.data ? Object.values(suggestionsQuery.data) : []

  const handleGenerate = async () => {
    setGenerating(true)
    try {
      await generateMutation.mutateAsync({})
    } catch (e: any) {
      console.error(e)
    } finally {
      setTimeout(() => {
        setGenerating(false)
      }, 10000)
    }
  }

  const handleDismiss = async (id: number) => {
    try {
      await dismissMutation.mutateAsync(id)
    } catch (e: any) {
      console.error(e)
    }
  }

  const handleRefine = async (id: number) => {
    setRefiningId(id)
    try {
      await refineMutation.mutateAsync(id)
    } catch (e: any) {
      console.error(e)
    } finally {
      setRefiningId(null)
    }
  }

  const handleAccept = async (id: number, destination: string) => {
    setAcceptResult(null)
    try {
      const result = await acceptMutation.mutateAsync({ id, destination })
      setAcceptResult(result)
    } catch (e: any) {
      console.error(e)
    } finally {
      setAcceptingId(null)
    }
  }

  const handleDismissAll = async () => {
    for (const s of suggestions) {
      try {
        await dismissMutation.mutateAsync(s.id)
      } catch (e: any) {
        console.error(e)
      }
    }
  }

  const handleRefineAll = async () => {
    try {
      await refineAllMutation.mutateAsync()
    } catch (e: any) {
      console.error(e)
    }
  }

  return (
    <div class="page suggestions-page">
      <div class="suggestions-header">
        <h2>Suggestions</h2>
        <div class="suggestions-actions">
          <button
            class="btn btn-sm btn-primary"
            onClick={handleGenerate}
            disabled={generating}
          >
            {generating ? "Generating..." : "Generate"}
          </button>
          {suggestions.length > 0 && (
            <>
              <button class="btn btn-sm" onClick={handleRefineAll} disabled={refineAllMutation.isPending}>
                Refine All
              </button>
              <button class="btn btn-sm btn-danger" onClick={handleDismissAll} disabled={dismissMutation.isPending}>
                Dismiss All
              </button>
            </>
          )}
        </div>
      </div>

      {acceptResult && <MaterializeStatus result={acceptResult} />}

      {suggestionsQuery.isLoading && <div class="empty">Loading...</div>}
      {!suggestionsQuery.isLoading && suggestions.length === 0 && (
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

function SuggestionCard({ suggestion, onDismiss, onRefine, onAccept, dismissing, refining, isMaterializing }: {
  suggestion: FolderSuggestionDisplay
  onDismiss: () => void
  onRefine: () => void
  onAccept: () => void
  dismissing: boolean
  refining: boolean
  isMaterializing: boolean
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
            {isMaterializing ? "Moving..." : "Accept"}
          </button>
        )}
        <button class="btn btn-sm" onClick={onRefine} disabled={refining}>
          {refining ? "Refining..." : "Refine"}
        </button>
        <button class="btn btn-sm btn-danger" onClick={onDismiss} disabled={dismissing}>
          Dismiss
        </button>
      </div>
    </div>
  )
}
