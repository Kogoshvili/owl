import { useState } from "preact/hooks"
import { useFolderSuggestions, useGenerateFolderSuggestions, useAcceptFolderSuggestion, useDismissFolderSuggestion, useRefineFolder, useStrategies } from "../hooks/queries"
import { route } from "preact-router"
import type { FolderSuggestion } from "../api"

export function FolderSuggestions() {
  const suggestionsQuery = useFolderSuggestions()
  const strategiesQuery = useStrategies()
  const generateMutation = useGenerateFolderSuggestions()
  const acceptMutation = useAcceptFolderSuggestion()
  const dismissMutation = useDismissFolderSuggestion()
  const refineMutation = useRefineFolder()
  const [generating, setGenerating] = useState(false)
  const [refiningId, setRefiningId] = useState<number | null>(null)
  const [strategy, setStrategy] = useState<string>("path_tfidf")

  const suggestions = suggestionsQuery.data ? Object.values(suggestionsQuery.data) : []
  const strategies = strategiesQuery.data ?? []

  const handleGenerate = async () => {
    setGenerating(true)
    try {
      await generateMutation.mutateAsync({ strategy })
    } catch (e: any) {
      console.error(e)
    } finally {
      setGenerating(false)
    }
  }

  const handleAccept = async (id: number) => {
    try {
      await acceptMutation.mutateAsync(id)
    } catch (e: any) {
      console.error(e)
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

  const handleAcceptAll = async () => {
    for (const s of suggestions) {
      try {
        await acceptMutation.mutateAsync(s.id)
      } catch (e: any) {
        console.error(e)
      }
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
    for (const s of suggestions) {
      try {
        await refineMutation.mutateAsync(s.id)
      } catch (e: any) {
        console.error(e)
      }
    }
  }

  return (
    <div class="folder-suggestions">
      <div class="folder-suggestions-header">
        <h2>Suggestions</h2>
        {strategies.length > 0 && (
          <select
            class="strategy-select"
            value={strategy}
            onChange={(e) => setStrategy((e.target as HTMLSelectElement).value)}
          >
            {strategies.map((s) => (
              <option key={s.id} value={s.id} disabled={!s.available}>
                {s.display_name} — {s.speed_hint}
              </option>
            ))}
          </select>
        )}
        <div class="folder-suggestions-actions">
          <button
            class="btn btn-sm btn-primary"
            onClick={handleGenerate}
            disabled={generating}
          >
            {generating ? "Generating..." : "Generate"}
          </button>
          {suggestions.length > 0 && (
            <>
              <button class="btn btn-sm" onClick={handleRefineAll} disabled={refineMutation.isPending}>
                Refine All
              </button>
              <button class="btn btn-sm" onClick={handleAcceptAll} disabled={acceptMutation.isPending}>
                Accept All
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
            onAccept={() => handleAccept(s.id)}
            onDismiss={() => handleDismiss(s.id)}
            onRefine={() => handleRefine(s.id)}
            accepting={acceptMutation.isPending}
            dismissing={dismissMutation.isPending}
            refining={refiningId === s.id}
          />
        ))}
      </div>
    </div>
  )
}

function SuggestionCard({ suggestion, onAccept, onDismiss, onRefine, accepting, dismissing, refining }: {
  suggestion: FolderSuggestion
  onAccept: () => void
  onDismiss: () => void
  onRefine: () => void
  accepting: boolean
  dismissing: boolean
  refining: boolean
}) {
  return (
    <div class="suggestion-card" onClick={() => route(`/virtual-folders/${suggestion.id}`)}>
      <div class="suggestion-card-header">
        <span class="suggestion-name">{suggestion.name}</span>
        <span class="badge badge-auto">auto</span>
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
        <button class="btn btn-sm btn-primary" onClick={onAccept} disabled={accepting}>
          Accept
        </button>
        <button class="btn btn-sm btn-danger" onClick={onDismiss} disabled={dismissing}>
          Dismiss
        </button>
      </div>
    </div>
  )
}