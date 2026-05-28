import { useState } from "preact/hooks"
import type { UseMutationResult } from "@tanstack/preact-query"
import type { SmartSuggestion } from "../api"

interface Props {
  suggestions: SmartSuggestion[]
  loading: boolean
  acceptMutation: UseMutationResult<any, Error, {
    type: string
    file_ids: number[]
    name?: string
    target_id?: number
    source_paths?: string[]
  }>
  generateMutation: UseMutationResult<any, Error, number, any>
  watchedDirId: number | null
  unprocessedCount: number | undefined
  onDismiss: (suggestion: SmartSuggestion) => void
}

export function SmartSuggestions({ suggestions, loading, acceptMutation, generateMutation, watchedDirId, unprocessedCount, onDismiss }: Props) {
  const [acceptingId, setAcceptingId] = useState<string | null>(null)

  const handleGenerate = async () => {
    if (!watchedDirId) return
    try {
      await generateMutation.mutateAsync(watchedDirId)
    } catch (e: any) {
      console.error(e)
    }
  }

  const handleAccept = async (s: SmartSuggestion) => {
    const id = `${s.type}-${s.name || s.target_path || s.source_paths?.join("+")}`
    setAcceptingId(id)
    try {
      await acceptMutation.mutateAsync({
        type: s.type,
        file_ids: s.file_ids,
        name: s.name,
        target_id: s.target_id,
        source_paths: s.source_paths,
      })
    } finally {
      setAcceptingId(null)
    }
  }

  const typeLabel = (t: string) => {
    switch (t) {
      case "new_folder": return "New Folder"
      case "add_to_folder": return "Add to Folder"
      case "merge_folders": return "Merge Folders"
      default: return t
    }
  }

  const typeBadge = (t: string) => {
    switch (t) {
      case "new_folder": return "new"
      case "add_to_folder": return "add"
      case "merge_folders": return "merge"
      default: return ""
    }
  }

  const hasGenerated = suggestions.length > 0

  return (
    <div class="smart-suggestions">
      <div class="smart-suggestions-header">
        <h2>Smart Suggestions</h2>
        {watchedDirId && !loading && (
          <button
            class="btn btn-sm btn-primary"
            onClick={handleGenerate}
            disabled={generateMutation.isPending}
          >
            {generateMutation.isPending ? "..." : "Generate"}
          </button>
        )}
      </div>

      {loading && <div class="empty">Analyzing...</div>}
      {!loading && !hasGenerated && !watchedDirId && (
        <div class="empty">Select a watched directory above to generate suggestions.</div>
      )}
      {!loading && !hasGenerated && watchedDirId && (
        <div class="empty">Click 'Generate' to analyze your folders and find orphan files.</div>
      )}
      {!loading && hasGenerated && suggestions.length === 0 && (
        <div class="empty">
          {unprocessedCount !== undefined && unprocessedCount > 0 ? (
            <>
              No suggestions found. {unprocessedCount} files are unprocessed.
              {" "}Extract content via the{" "}
              <a href="/ingest" class="link">Ingest page</a> to enable smart suggestions.
            </>
          ) : (
            "No suggestions found. Your files are already well-organized."
          )}
        </div>
      )}

      <div class="smart-suggestion-list">
        {suggestions.map((s, i) => {
          const id = `${s.type}-${s.name || s.target_path || s.source_paths?.join("+")}-${i}`
          return (
            <div class="smart-suggestion-card" key={id}>
              <div class="smart-suggestion-card-header">
                <span class={`suggestion-type-badge badge-${typeBadge(s.type)}`}>{typeLabel(s.type)}</span>
                <span class="smart-suggestion-confidence">
                  {Math.round(s.confidence * 100)}%
                </span>
              </div>

              {s.type === "new_folder" && (
                <>
                  <div class="smart-suggestion-name">{s.name}</div>
                  {s.description && <div class="smart-suggestion-desc">{s.description}</div>}
                  <div class="smart-suggestion-meta">{s.file_count} files</div>
                  {s.preview && s.preview.length > 0 && (
                    <div class="smart-suggestion-preview">
                      {s.preview.slice(0, 5).map((name, i) => (
                        <span class="smart-suggestion-preview-file" key={i}>{name}</span>
                      ))}
                      {s.preview.length > 5 && (
                        <span class="smart-suggestion-preview-more">+{s.preview.length - 5} more</span>
                      )}
                    </div>
                  )}
                </>
              )}

              {s.type === "add_to_folder" && (
                <>
                  <div class="smart-suggestion-name">Add {s.file_count} files to</div>
                  <div class="smart-suggestion-target">📁 {s.target_path || s.name}</div>
                  {s.preview && s.preview.length > 0 && (
                    <div class="smart-suggestion-preview">
                      {s.preview.slice(0, 5).map((name, i) => (
                        <span class="smart-suggestion-preview-file" key={i}>{name}</span>
                      ))}
                      {s.preview.length > 5 && (
                        <span class="smart-suggestion-preview-more">+{s.preview.length - 5} more</span>
                      )}
                    </div>
                  )}
                </>
              )}

              {s.type === "merge_folders" && (
                <>
                  <div class="smart-suggestion-name">Merge {s.source_paths?.length ?? 0} folders</div>
                  <div class="smart-suggestion-target">
                    {s.source_paths?.map((p) => `📁 ${p.split("/").pop()}`).join(" + ")}
                  </div>
                  <div class="smart-suggestion-meta">{s.file_count} files total</div>
                </>
              )}

              <div class="smart-suggestion-actions">
                <button
                  class="btn btn-sm btn-primary"
                  onClick={() => handleAccept(s)}
                  disabled={acceptingId === id}
                >
                  {acceptingId === id ? "..." : "Accept"}
                </button>
                <button
                  class="btn btn-sm btn-danger"
                  onClick={() => onDismiss(s)}
                  disabled={acceptingId === id}
                >
                  Dismiss
                </button>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
