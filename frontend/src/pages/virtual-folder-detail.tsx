import { useState } from "preact/hooks"
import { useVirtualFolderDetail, useRemoveFileFromFolder, useUpdateVirtualFolder, useDeleteVirtualFolder, useAddFilesToFolder } from "../hooks/queries"
import { FilePickerDialog } from "../components/file-picker-dialog"
import { route } from "preact-router"
import type { VirtualFolder } from "../api"

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i]
}

function FolderHeader({ folder, updateMutation, deleteMutation }: {
  folder: VirtualFolder
  updateMutation: ReturnType<typeof useUpdateVirtualFolder>
  deleteMutation: ReturnType<typeof useDeleteVirtualFolder>
}) {
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(folder.name)
  const [desc, setDesc] = useState(folder.description)
  const [error, setError] = useState("")

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({ id: folder.id, name: name.trim() || undefined, description: desc.trim() || undefined })
      setEditing(false)
    } catch (e: any) {
      setError(e.message)
    }
  }

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(folder.id)
      route("/virtual-folders")
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
        <h2>{folder.name}</h2>
        {folder.auto_generated && <span class="badge badge-auto">auto</span>}
        {folder.materialized && <span class="badge badge-materialized">materialized</span>}
      </div>
      {folder.description && <p class="folder-detail-desc">{folder.description}</p>}
      <div class="folder-header-actions">
        <button class="btn btn-sm" onClick={() => setEditing(true)}>Edit</button>
        <button class="btn btn-sm btn-danger" onClick={handleDelete} disabled={deleteMutation.isPending}>Delete</button>
      </div>
    </div>
  )
}

export function VirtualFolderDetailPage({ id }: { id: string }) {
  const folderId = parseInt(id, 10)
  const detailQuery = useVirtualFolderDetail(isNaN(folderId) ? null : folderId)
  const removeMutation = useRemoveFileFromFolder()
  const updateMutation = useUpdateVirtualFolder()
  const deleteMutation = useDeleteVirtualFolder()
  const addFilesMutation = useAddFilesToFolder()
  const [pickerOpen, setPickerOpen] = useState(false)

  if (isNaN(folderId)) return <div class="page"><div class="error-msg">Invalid folder ID</div></div>
  if (detailQuery.isLoading) return <div class="page"><div class="empty">Loading...</div></div>
  if (!detailQuery.data) return <div class="page"><div class="empty">Folder not found</div></div>

  const folder = detailQuery.data
  const files = folder.files ?? []
  const notes = folder.notes ?? []
  const existingFileIds = new Set(files.map((f) => f.id))

  const handleRemoveFile = async (fileId: number) => {
    try {
      await removeMutation.mutateAsync({ folderId, fileId })
    } catch (e: any) {
      console.error(e)
    }
  }

  const handleAddFiles = async (fileIds: number[]) => {
    try {
      await addFilesMutation.mutateAsync({ folderId, fileIds, source: "manual" })
      setPickerOpen(false)
    } catch (e: any) {
      console.error(e)
    }
  }

  return (
    <div class="page folder-detail-page">
      <button class="btn btn-back" onClick={() => route("/virtual-folders")}>&larr; Back</button>

      <FolderHeader folder={folder} updateMutation={updateMutation} deleteMutation={deleteMutation} />

      <div class="folder-section">
        <div class="folder-section-header">
          <h3>Files ({files.length})</h3>
          <button class="btn btn-primary btn-sm" onClick={() => setPickerOpen(true)}>Add Files</button>
        </div>

        {files.length === 0 ? (
          <div class="empty">No files in this folder</div>
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
                    <button
                      class="btn btn-sm btn-danger"
                      disabled={removeMutation.isPending}
                      onClick={() => handleRemoveFile(f.id)}
                    >
                      Remove
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div class="folder-section">
        <h3>Notes ({notes.length})</h3>
        {notes.length === 0 ? (
          <div class="empty notes-placeholder">Notes coming in v2</div>
        ) : (
          <ul class="folder-notes-list">
            {notes.map((n) => (
              <li key={n.id}>{n.title}</li>
            ))}
          </ul>
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
