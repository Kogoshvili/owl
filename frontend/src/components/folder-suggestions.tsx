import { useState } from "preact/hooks"
import { useFolderSuggestions, useGenerateFolderSuggestions, useAcceptFolderSuggestion, useDismissFolderSuggestion } from "../hooks/queries"
import type { FolderSuggestion } from "../api"

export function FolderSuggestions() {
  const suggestionsQuery = useFolderSuggestions()
  const generateMutation = useGenerateFolderSuggestions()
  const acceptMutation = useAcceptFolderSuggestion()
  const dismissMutation = useDismissFolderSuggestion()
  const [generating, setGenerating] = useState(false)

  const suggestions = suggestionsQuery.data ? Object.values(suggestionsQuery.data) : []

  const handleGenerate = async () => {
    setGenerating(true)
    try {
      await generateMutation.mutateAsync({})
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

  return (
    <div class="folder-suggestions">
      <div class="folder-suggestions-header">
        <h2>Suggestions</h2>
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
            accepting={acceptMutation.isPending}
            dismissing={dismissMutation.isPending}
          />
        ))}
      </div>
    </div>
  )
}

function SuggestionCard({ suggestion, onAccept, onDismiss, accepting, dismissing }: {
  suggestion: FolderSuggestion
  onAccept: () => void
  onDismiss: () => void
  accepting: boolean
  dismissing: boolean
}) {
  return (
    <div class="suggestion-card">
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
      <div class="suggestion-actions">
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
