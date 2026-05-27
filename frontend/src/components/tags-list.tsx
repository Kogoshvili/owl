import { useState } from "preact/hooks"
import { useTags, useCreateTag, useDeleteTag, useAcceptTag, useTagFiles } from "../hooks/queries"
import { route } from "preact-router"
import type { TagWithCount } from "../api"

interface Props {
  source?: "auto" | "manual"
}

export function TagsList({ source }: Props) {
  const tagsQuery = useTags(source)
  const createMutation = useCreateTag()
  const deleteMutation = useDeleteTag()
  const acceptMutation = useAcceptTag()
  const autoTagMutation = useTagFiles()
  const [newTagName, setNewTagName] = useState("")
  const [filter, setFilter] = useState<"all" | "auto" | "manual">("all")
  const [autoTagResult, setAutoTagResult] = useState<{count: number; tagged: number; tag_count: number} | null>(null)

  const handleCreate = async () => {
    const name = newTagName.trim()
    if (!name) return
    try {
      await createMutation.mutateAsync(name)
      setNewTagName("")
    } catch (e: any) {
      console.error(e)
    }
  }

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Enter") handleCreate()
  }

  const handleAutoTag = async () => {
    setAutoTagResult(null)
    try {
      const result = await autoTagMutation.mutateAsync({})
      setAutoTagResult(result)
    } catch (e: any) {
      console.error(e)
    }
  }

  const tags = tagsQuery.data ?? []

  return (
    <div class="tags-list">
      <div class="tags-list-header">
        <h2>Tags</h2>
        <div class="tags-header-right">
          <button
            class="btn btn-sm btn-primary"
            onClick={handleAutoTag}
            disabled={autoTagMutation.isPending}
          >
            {autoTagMutation.isPending ? "Tagging..." : "Auto-Tag All"}
          </button>
          <div class="tags-source-pills">
            <button
              class={`scope-pill ${filter === "all" ? "active" : ""}`}
              onClick={() => setFilter("all")}
            >
              All
            </button>
            <button
              class={`scope-pill ${filter === "manual" ? "active" : ""}`}
              onClick={() => setFilter("manual")}
            >
              Manual
            </button>
            <button
              class={`scope-pill ${filter === "auto" ? "active" : ""}`}
              onClick={() => setFilter("auto")}
            >
              Auto
            </button>
          </div>
        </div>
      </div>

      {autoTagResult && (
        <div class="auto-tag-result">
          Tagged {autoTagResult.tagged} of {autoTagResult.count} files with {autoTagResult.tag_count} tags
          <button class="auto-tag-result-dismiss" onClick={() => setAutoTagResult(null)}>x</button>
        </div>
      )}

      <div class="add-tag-form">
        <input
          type="text"
          placeholder="New tag name"
          value={newTagName}
          onInput={(e) => setNewTagName((e.target as HTMLInputElement).value)}
          onKeyDown={handleKeyDown}
          disabled={createMutation.isPending}
        />
        <button
          class="btn btn-primary btn-sm"
          onClick={handleCreate}
          disabled={createMutation.isPending || !newTagName.trim()}
        >
          Create
        </button>
      </div>

      {tagsQuery.isLoading && <div class="empty">Loading...</div>}
      {!tagsQuery.isLoading && tags.length === 0 && (
        <div class="empty">No tags yet</div>
      )}

      <div class="tags-grid">
        {tags.map((tag) => (
          <TagCard
            key={tag.id}
            tag={tag}
            onDelete={() => deleteMutation.mutate(tag.id)}
            onAccept={() => acceptMutation.mutate(tag.id)}
            deleting={deleteMutation.isPending}
            accepting={acceptMutation.isPending}
          />
        ))}
      </div>
    </div>
  )
}

function TagCard({ tag, onDelete, onAccept, deleting, accepting }: {
  tag: TagWithCount
  onDelete: () => void
  onAccept: () => void
  deleting: boolean
  accepting: boolean
}) {
  return (
    <div class="tag-card" onClick={() => route(`/tags/${tag.id}`)}>
      <div class="tag-card-header">
        <span class="tag-card-name">{tag.name}</span>
        <span class={`badge ${tag.source === "auto" ? "badge-auto" : "badge-manual"}`}>
          {tag.source}
        </span>
      </div>
      <div class="tag-card-count">{tag.file_count} file{tag.file_count !== 1 ? "s" : ""}</div>
      <div class="tag-card-actions" onClick={(e) => e.stopPropagation()}>
        {tag.source === "auto" && (
          <button class="btn btn-sm" onClick={onAccept} disabled={accepting}>
            Accept
          </button>
        )}
        <button class="btn btn-sm btn-danger" onClick={onDelete} disabled={deleting}>
          Delete
        </button>
      </div>
    </div>
  )
}
