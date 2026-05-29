import { useState } from "preact/hooks"
import { useFolderSuggestions, useGenerateSuggestions, useDismissSuggestion, useRefineSuggestion, useRefineAllSuggestions } from "../hooks/queries"
import { route } from "preact-router"
import type { FolderSuggestionDisplay } from "../api"

export function SuggestionsPage() {
  const suggestionsQuery = useFolderSuggestions()
  const generateMutation = useGenerateSuggestions()
  const dismissMutation = useDismissSuggestion()
  const refineMutation = useRefineSuggestion()
  const refineAllMutation = useRefineAllSuggestions()
  const [generating, setGenerating] = useState(false)
  const [refiningId, setRefiningId] = useState<number | null>(null)

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
            dismissing={dismissMutation.isPending}
            refining={refiningId === s.id}
          />
        ))}
      </div>
    </div>
  )
}

function SuggestionCard({ suggestion, onDismiss, onRefine, dismissing, refining }: {
  suggestion: FolderSuggestionDisplay
  onDismiss: () => void
  onRefine: () => void
  dismissing: boolean
  refining: boolean
}) {
  return (
    <div class="suggestion-card" onClick={() => route(`/suggestions/${suggestion.id}`)}>
      <div class="suggestion-card-header">
        <span class="suggestion-name">{suggestion.name}</span>
        {suggestion.confidence > 0 && (
          <span class="badge badge-confidence">{Math.round(suggestion.confidence * 100)}%</span>
        )}
      </div>
      {suggestion.description && (
        <div class="suggestion-desc">{suggestion.description}</div>
      )}
      <div class="suggestion-meta">
        <span>{suggestion.file_count} files</span>
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
        <button class="btn btn-sm btn-primary" onClick={onRefine} disabled={refining}>
          {refining ? "Refining..." : "Refine"}
        </button>
        <button class="btn btn-sm btn-danger" onClick={onDismiss} disabled={dismissing}>
          Dismiss
        </button>
      </div>
    </div>
  )
}
