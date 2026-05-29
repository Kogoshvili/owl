import { useState } from "preact/hooks"
import { useSuggestionDetail, useRemoveFileFromSuggestion, useUpdateSuggestion, useDeleteSuggestion, useAddFilesToSuggestion, useRefineSuggestion, useAcceptSuggestion } from "../hooks/queries"
import { FilePickerDialog } from "../components/file-picker-dialog"
import { route } from "preact-router"
import type { FolderSuggestion } from "../api"

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i]
}

function SuggestionHeader({ suggestion, updateMutation, deleteMutation, refineMutation, acceptMutation }: {
  suggestion: FolderSuggestion
  updateMutation: ReturnType<typeof useUpdateSuggestion>
  deleteMutation: ReturnType<typeof useDeleteSuggestion>
  refineMutation: ReturnType<typeof useRefineSuggestion>
  acceptMutation: ReturnType<typeof useAcceptSuggestion>
}) {
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(suggestion.name)
  const [desc, setDesc] = useState(suggestion.description)
  const [error, setError] = useState("")
  const [showAccept, setShowAccept] = useState(false)
  const [acceptDest, setAcceptDest] = useState("")

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({ id: suggestion.id, name: name.trim() || undefined, description: desc.trim() || undefined })
      setEditing(false)
    } catch (e: any) {
      setError(e.message)
    }
  }

  const handleDismiss = async () => {
    try {
      await deleteMutation.mutateAsync(suggestion.id)
      route("/suggestions")
    } catch (e: any) {
      setError(e.message)
    }
  }

  const handleRefine = async () => {
    try {
      await refineMutation.mutateAsync(suggestion.id)
    } catch (e: any) {
      setError(e.message)
    }
  }

  const handleAccept = async () => {
    try {
      await acceptMutation.mutateAsync({ id: suggestion.id, destination: acceptDest || undefined })
      setShowAccept(false)
    } catch (e: any) {
      setError(e.message)
    }
  }

  if (editing) {
    return (
      <div class="folder-detail-header editing">
        <input type="text" value={name} onInput={(e) => setName((e.target as HTMLInputElement).value)} />
        <input type="text" value={desc} onInput={(e) => setDesc((e.target as HTMLInputElement).value)} placeholder="Description" />
        <div class="folder-header-actions">
          <button class="btn btn-sm btn-primary" onClick={handleSave} disabled={updateMutation.isPending}>Save</button>
          <button class="btn btn-sm" onClick={() => setEditing(false)}>Cancel</button>
        </div>
        {error && <div class="error-msg">{error}</div>}
      </div>
    )
  }

  return (
    <div class="folder-detail-header">
      <div class="folder-detail-title-row">
        <h2>{suggestion.name}</h2>
        {suggestion.materialized_at && <span class="badge badge-materialized">Materialized</span>}
      </div>
      {suggestion.description && <p class="folder-detail-desc">{suggestion.description}</p>}
      {suggestion.materialized_path && (
        <p class="folder-detail-path">→ {suggestion.materialized_path}</p>
      )}
      {suggestion.confidence > 0 && (
        <span class="badge badge-confidence">{Math.round(suggestion.confidence * 100)}%</span>
      )}
      <div class="folder-header-actions">
        <button class="btn btn-sm" onClick={() => setEditing(true)}>Edit</button>
        {!suggestion.materialized_at && (
          <button class="btn btn-sm btn-primary" onClick={() => setShowAccept(true)} disabled={acceptMutation.isPending}>
            {acceptMutation.isPending ? "Moving..." : "Accept"}
          </button>
        )}
        <button class="btn btn-sm btn-primary" onClick={handleRefine} disabled={refineMutation.isPending}>
          {refineMutation.isPending ? "Refining..." : "Refine"}
        </button>
        <button class="btn btn-sm btn-danger" onClick={handleDismiss} disabled={deleteMutation.isPending}>
          Dismiss
        </button>
      </div>

      {showAccept && (
        <div class="modal-overlay" onClick={() => setShowAccept(false)}>
          <div class="modal" onClick={(e) => e.stopPropagation()}>
            <h3>Accept: {suggestion.name}</h3>
            <p>This will move the files into a new folder.</p>
            <label>
              Destination base path
              <input
                type="text"
                value={acceptDest}
                onInput={(e) => setAcceptDest((e.target as HTMLInputElement).value)}
                placeholder="~/Owl-organized (default)"
              />
            </label>
            <div class="modal-actions">
              <button class="btn btn-primary" onClick={handleAccept} disabled={acceptMutation.isPending}>
                {acceptMutation.isPending ? "Moving..." : "Accept & Materialize"}
              </button>
              <button class="btn" onClick={() => setShowAccept(false)}>Cancel</button>
            </div>
            {error && <div class="error-msg">{error}</div>}
          </div>
        </div>
      )}
    </div>
  )
}

export function SuggestionDetailPage({ id }: { id: string }) {
  const suggestionId = parseInt(id, 10)
  const detailQuery = useSuggestionDetail(isNaN(suggestionId) ? null : suggestionId)
  const removeMutation = useRemoveFileFromSuggestion()
  const updateMutation = useUpdateSuggestion()
  const deleteMutation = useDeleteSuggestion()
  const addFilesMutation = useAddFilesToSuggestion()
  const refineMutation = useRefineSuggestion()
  const acceptMutation = useAcceptSuggestion()
  const [pickerOpen, setPickerOpen] = useState(false)

  if (isNaN(suggestionId)) return <div class="page"><div class="error-msg">Invalid suggestion ID</div></div>
  if (detailQuery.isLoading) return <div class="page"><div class="empty">Loading...</div></div>
  if (!detailQuery.data) return <div class="page"><div class="empty">Suggestion not found</div></div>

  const suggestion = detailQuery.data
  const files = suggestion.files ?? []
  const existingFileIds = new Set(files.map((f) => f.id))

  const handleRemoveFile = async (fileId: number) => {
    try {
      await removeMutation.mutateAsync({ suggestionId, fileId })
    } catch (e: any) {
      console.error(e)
    }
  }

  const handleAddFiles = async (fileIds: number[]) => {
    try {
      await addFilesMutation.mutateAsync({ suggestionId, fileIds })
      setPickerOpen(false)
    } catch (e: any) {
      console.error(e)
    }
  }

  return (
    <div class="page folder-detail-page">
      <button class="btn btn-back" onClick={() => route("/suggestions")}>&larr; Back</button>

      <SuggestionHeader suggestion={suggestion} updateMutation={updateMutation} deleteMutation={deleteMutation} refineMutation={refineMutation} acceptMutation={acceptMutation} />

      <div class="folder-section">
        <div class="folder-section-header">
          <h3>Files ({files.length})</h3>
          {!suggestion.materialized_at && (
            <button class="btn btn-primary btn-sm" onClick={() => setPickerOpen(true)}>Add Files</button>
          )}
        </div>

        {files.length === 0 ? (
          <div class="empty">No files in this suggestion</div>
        ) : (
          <table class="folder-files-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Extension</th>
                <th>Size</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {files.map((f) => (
                <tr key={f.id}>
                  <td>
                    <a href={`/files/${f.id}`} onClick={(e) => { e.preventDefault(); route(`/files/${f.id}`) }}>{f.name}</a>
                  </td>
                  <td>{f.extension || "-"}</td>
                  <td>{formatBytes(f.size)}</td>
                  <td>
                    {!suggestion.materialized_at && (
                      <button
                        class="btn btn-sm btn-danger"
                        disabled={removeMutation.isPending}
                        onClick={() => handleRemoveFile(f.id)}
                      >
                        Remove
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {pickerOpen && (
        <FilePickerDialog
          existingFileIds={existingFileIds}
          onAdd={handleAddFiles}
          onClose={() => setPickerOpen(false)}
          adding={addFilesMutation.isPending}
        />
      )}
    </div>
  )
}
