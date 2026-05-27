import { useState } from "preact/hooks"
import { useTags, useCreateTag, useDeleteTag } from "../hooks/queries"
import { route } from "preact-router"
import type { TagWithCount } from "../api"

export function TagsList() {
  const tagsQuery = useTags("manual")
  const createMutation = useCreateTag()
  const deleteMutation = useDeleteTag()
  const [newTagName, setNewTagName] = useState("")

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

  const tags = tagsQuery.data ?? []

  return (
    <div class="tags-list">
      <div class="tags-list-header">
        <h2>Tags</h2>
      </div>

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
            deleting={deleteMutation.isPending}
          />
        ))}
      </div>
    </div>
  )
}

function TagCard({ tag, onDelete, deleting }: {
  tag: TagWithCount
  onDelete: () => void
  deleting: boolean
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
      {tag.description && (
        <div class="tag-card-desc">{tag.description}</div>
      )}
      <div class="tag-card-actions" onClick={(e) => e.stopPropagation()}>
        <button class="btn btn-sm btn-danger" onClick={onDelete} disabled={deleting}>
          Delete
        </button>
      </div>
    </div>
  )
}
