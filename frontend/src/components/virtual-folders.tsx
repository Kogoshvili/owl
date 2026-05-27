import { useState } from "preact/hooks"
import type { UseMutationResult } from "@tanstack/preact-query"
import { type VirtualFolder } from "../api"
import { route } from "preact-router"

interface Props {
  folders: VirtualFolder[]
  loading?: boolean
  createMutation: UseMutationResult<VirtualFolder, Error, { name: string; description?: string }>
  updateMutation: UseMutationResult<VirtualFolder, Error, { id: number; name?: string; description?: string }>
  deleteMutation: UseMutationResult<void, Error, number>
}

export function VirtualFolders({ folders, loading, createMutation, updateMutation, deleteMutation }: Props) {
  const [name, setName] = useState("")
  const [description, setDescription] = useState("")
  const [error, setError] = useState("")
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editName, setEditName] = useState("")
  const [editDesc, setEditDesc] = useState("")

  const handleCreate = async () => {
    const n = name.trim()
    if (!n) return
    setError("")
    try {
      await createMutation.mutateAsync({ name: n, description: description.trim() || undefined })
      setName("")
      setDescription("")
    } catch (e: any) {
      setError(e.message)
    }
  }

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Enter") handleCreate()
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
    } catch (e: any) {
      setError(e.message)
    }
  }

  const startEdit = (f: VirtualFolder) => {
    setEditingId(f.id)
    setEditName(f.name)
    setEditDesc(f.description)
  }

  const cancelEdit = () => {
    setEditingId(null)
    setEditName("")
    setEditDesc("")
  }

  const saveEdit = async (id: number) => {
    try {
      await updateMutation.mutateAsync({ id, name: editName.trim() || undefined, description: editDesc.trim() || undefined })
      cancelEdit()
    } catch (e: any) {
      setError(e.message)
    }
  }

  const creating = createMutation.isPending

  return (
    <div class="virtual-folders">
      <h2>Virtual Folders</h2>

      <div class="add-folder-form">
        <input
          type="text"
          placeholder="Folder name"
          value={name}
          onInput={(e) => setName((e.target as HTMLInputElement).value)}
          onKeyDown={handleKeyDown}
          disabled={creating}
        />
        <input
          type="text"
          placeholder="Description (optional)"
          value={description}
          onInput={(e) => setDescription((e.target as HTMLInputElement).value)}
          disabled={creating}
        />
        <button class="btn btn-primary" onClick={handleCreate} disabled={creating || !name.trim()}>
          {creating ? "..." : "Create"}
        </button>
      </div>

      {error && <div class="error-msg">{error}</div>}

      <div class="folder-list">
        {loading && <div class="empty">Loading...</div>}
        {!loading && folders.length === 0 && <div class="empty">No virtual folders yet</div>}
        {folders.map((f) => (
          <div class="folder-card" onClick={() => route(`/virtual-folders/${f.id}`)}>
            {editingId === f.id ? (
              <div class="folder-edit" onClick={(e) => e.stopPropagation()}>
                <input
                  type="text"
                  value={editName}
                  onInput={(e) => setEditName((e.target as HTMLInputElement).value)}
                />
                <input
                  type="text"
                  value={editDesc}
                  onInput={(e) => setEditDesc((e.target as HTMLInputElement).value)}
                  placeholder="Description"
                />
                <button class="btn btn-sm btn-primary" onClick={() => saveEdit(f.id)} disabled={updateMutation.isPending}>
                  Save
                </button>
                <button class="btn btn-sm" onClick={cancelEdit}>Cancel</button>
              </div>
            ) : (
              <>
                <div class="folder-card-header">
                  <span class="folder-name">{f.name}</span>
                  <span class={`badge ${f.source === "auto" ? "badge-auto" : "badge-manual"}`}>{f.source}</span>
                  {f.materialized && <span class="badge badge-materialized">materialized</span>}
                </div>
                {f.description && <div class="folder-desc">{f.description}</div>}
                <div class="folder-card-meta">
                  <span class="folder-created">{new Date(f.created_at).toLocaleDateString()}</span>
                </div>
                <div class="folder-card-actions">
                  <button
                    class="btn btn-sm"
                    disabled={updateMutation.isPending}
                    onClick={(e) => { e.stopPropagation(); startEdit(f) }}
                  >
                    Edit
                  </button>
                  <button
                    class="btn btn-sm btn-danger"
                    disabled={deleteMutation.isPending}
                    onClick={(e) => { e.stopPropagation(); handleDelete(f.id) }}
                  >
                    Delete
                  </button>
                </div>
              </>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
