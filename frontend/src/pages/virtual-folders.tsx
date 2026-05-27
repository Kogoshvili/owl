import { useVirtualFolders, useCreateVirtualFolder, useUpdateVirtualFolder, useDeleteVirtualFolder } from "../hooks/queries"
import { VirtualFolders } from "../components/virtual-folders"
import { FolderSuggestions } from "../components/folder-suggestions"

export function VirtualFoldersPage() {
  const foldersQuery = useVirtualFolders()
  const createMutation = useCreateVirtualFolder()
  const updateMutation = useUpdateVirtualFolder()
  const deleteMutation = useDeleteVirtualFolder()

  return (
    <div class="page vf-page-layout">
      <div class="vf-page-main">
        <VirtualFolders
          folders={foldersQuery.data ?? []}
          loading={foldersQuery.isLoading}
          createMutation={createMutation}
          updateMutation={updateMutation}
          deleteMutation={deleteMutation}
        />
      </div>
      <div class="vf-page-sidebar">
        <FolderSuggestions />
      </div>
    </div>
  )
}
