import { useState } from "preact/hooks"
import { useTags, useTagFiles, useAcceptTag, useDeleteTag, useRefineTag, useStrategies } from "../hooks/queries"
import { route } from "preact-router"
import type { TagWithCount } from "../api"

export function TagSuggestions() {
  const tagsQuery = useTags("auto")
  const strategiesQuery = useStrategies()
  const autoTagMutation = useTagFiles()
  const acceptMutation = useAcceptTag()
  const deleteMutation = useDeleteTag()
  const refineMutation = useRefineTag()
  const [autoTagResult, setAutoTagResult] = useState<{count: number; tagged: number; tag_count: number} | null>(null)
  const [refiningId, setRefiningId] = useState<number | null>(null)
  const [strategy, setStrategy] = useState<string>("path_tfidf")

  const autoTags = tagsQuery.data ?? []
  const strategies = strategiesQuery.data ?? []

  const handleAutoTag = async () => {
    setAutoTagResult(null)
    try {
      const result = await autoTagMutation.mutateAsync({ strategy })
      setAutoTagResult(result)
    } catch (e: any) {
      console.error(e)
    }
  }

  const handleAccept = async (id: number) => {
    try {
      await acceptMutation.mutateAsync(id)
    } catch (e: any) {
      console.error(e)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
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
    for (const t of autoTags) {
      try {
        await acceptMutation.mutateAsync(t.id)
      } catch (e: any) {
        console.error(e)
      }
    }
  }

  const handleDismissAll = async () => {
    for (const t of autoTags) {
      try {
        await deleteMutation.mutateAsync(t.id)
      } catch (e: any) {
        console.error(e)
      }
    }
  }

  const handleRefineAll = async () => {
    for (const t of autoTags) {
      try {
        await refineMutation.mutateAsync(t.id)
      } catch (e: any) {
        console.error(e)
      }
    }
  }

  const generating = autoTagMutation.isPending

  return (
    <div class="tag-suggestions">
      <div class="tag-suggestions-header">
        <h2>Auto Tags</h2>
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
        <div class="tag-suggestions-actions">
          <button
            class="btn btn-sm btn-primary"
            onClick={handleAutoTag}
            disabled={generating}
          >
            {generating ? "Tagging..." : "Auto-Tag All"}
          </button>
          {autoTags.length > 0 && (
            <>
              <button class="btn btn-sm" onClick={handleRefineAll} disabled={refineMutation.isPending}>
                Refine All
              </button>
              <button class="btn btn-sm" onClick={handleAcceptAll} disabled={acceptMutation.isPending}>
                Accept All
              </button>
              <button class="btn btn-sm btn-danger" onClick={handleDismissAll} disabled={deleteMutation.isPending}>
                Dismiss All
              </button>
            </>
          )}
        </div>
      </div>

      {autoTagResult && (
        <div class="auto-tag-result">
          Tagged {autoTagResult.tagged} of {autoTagResult.count} files with {autoTagResult.tag_count} tags
          <button class="auto-tag-result-dismiss" onClick={() => setAutoTagResult(null)}>x</button>
        </div>
      )}

      {tagsQuery.isLoading && <div class="empty">Loading...</div>}
      {!tagsQuery.isLoading && autoTags.length === 0 && (
        <div class="empty">No auto tags. Click Auto-Tag All to generate.</div>
      )}

      <div class="tag-suggestion-list">
        {autoTags.map((tag) => (
          <TagSuggestionCard
            key={tag.id}
            tag={tag}
            onAccept={() => handleAccept(tag.id)}
            onDelete={() => handleDelete(tag.id)}
            onRefine={() => handleRefine(tag.id)}
            accepting={acceptMutation.isPending}
            deleting={deleteMutation.isPending}
            refining={refiningId === tag.id}
          />
        ))}
      </div>
    </div>
  )
}

function TagSuggestionCard({ tag, onAccept, onDelete, onRefine, accepting, deleting, refining }: {
  tag: TagWithCount
  onAccept: () => void
  onDelete: () => void
  onRefine: () => void
  accepting: boolean
  deleting: boolean
  refining: boolean
}) {
  return (
    <div class="tag-suggestion-card" onClick={() => route(`/tags/${tag.id}`)}>
      <div class="tag-suggestion-card-header">
        <span class="tag-suggestion-name">{tag.name}</span>
        <span class="tag-suggestion-count">{tag.file_count} file{tag.file_count !== 1 ? "s" : ""}</span>
      </div>
      {tag.description && (
        <div class="tag-suggestion-desc">{tag.description}</div>
      )}
      <div class="tag-suggestion-actions" onClick={(e) => e.stopPropagation()}>
        <button class="btn btn-sm btn-primary" onClick={onRefine} disabled={refining}>
          {refining ? "Refining..." : "Refine"}
        </button>
        <button class="btn btn-sm btn-primary" onClick={onAccept} disabled={accepting}>
          Accept
        </button>
        <button class="btn btn-sm btn-danger" onClick={onDelete} disabled={deleting}>
          Dismiss
        </button>
      </div>
    </div>
  )
}
