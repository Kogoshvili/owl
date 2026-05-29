import { useState } from "preact/hooks"
import { useVirtualFolders, useCreateVirtualFolder, useUpdateVirtualFolder, useDeleteVirtualFolder } from "../hooks/queries"
import { VirtualFolders } from "../components/virtual-folders"
import { FolderSuggestions } from "../components/folder-suggestions"

export function VirtualFoldersPage() {
  const [showVirtual, setShowVirtual] = useState(true)

  const foldersQuery = useVirtualFolders("manual")
  const createMutation = useCreateVirtualFolder()
  const updateMutation = useUpdateVirtualFolder()
  const deleteMutation = useDeleteVirtualFolder()

  return (
    <div class="page vf-page-layout">
      <div class="vf-page-main">
        <div class="toggle-section" onClick={() => setShowVirtual(!showVirtual)}>
          <h3>
            <span class={`toggle-arrow ${showVirtual ? "expanded" : ""}`}>▸</span>
            Virtual Folders
          </h3>
        </div>

        {showVirtual && (
          <VirtualFolders
            folders={foldersQuery.data ?? []}
            loading={foldersQuery.isLoading}
            createMutation={createMutation}
            updateMutation={updateMutation}
            deleteMutation={deleteMutation}
          />
        )}
      </div>

      <div class="vf-page-sidebar">
        <FolderSuggestions />
      </div>
    </div>
  )
}
