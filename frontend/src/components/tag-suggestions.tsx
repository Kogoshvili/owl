import { useState } from "preact/hooks"
import { useTags, useTagFiles, useAcceptTag, useDeleteTag } from "../hooks/queries"
import { route } from "preact-router"
import type { TagWithCount } from "../api"

export function TagSuggestions() {
  const tagsQuery = useTags("auto")
  const autoTagMutation = useTagFiles()
  const acceptMutation = useAcceptTag()
  const deleteMutation = useDeleteTag()
  const [autoTagResult, setAutoTagResult] = useState<{count: number; tagged: number; tag_count: number} | null>(null)

  const autoTags = tagsQuery.data ?? []

  const handleAutoTag = async () => {
    setAutoTagResult(null)
    try {
      const result = await autoTagMutation.mutateAsync({})
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

  const generating = autoTagMutation.isPending

  return (
    <div class="tag-suggestions">
      <div class="tag-suggestions-header">
        <h2>Auto Tags</h2>
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
            accepting={acceptMutation.isPending}
            deleting={deleteMutation.isPending}
          />
        ))}
      </div>
    </div>
  )
}

function TagSuggestionCard({ tag, onAccept, onDelete, accepting, deleting }: {
  tag: TagWithCount
  onAccept: () => void
  onDelete: () => void
  accepting: boolean
  deleting: boolean
}) {
  return (
    <div class="tag-suggestion-card" onClick={() => route(`/tags/${tag.id}`)}>
      <div class="tag-suggestion-card-header">
        <span class="tag-suggestion-name">{tag.name}</span>
        <span class="tag-suggestion-count">{tag.file_count} file{tag.file_count !== 1 ? "s" : ""}</span>
      </div>
      <div class="tag-suggestion-actions" onClick={(e) => e.stopPropagation()}>
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
