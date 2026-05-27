import { useVirtualFolders, useCreateVirtualFolder, useUpdateVirtualFolder, useDeleteVirtualFolder } from "../hooks/queries"
import { VirtualFolders } from "../components/virtual-folders"

export function VirtualFoldersPage() {
  const foldersQuery = useVirtualFolders()
  const createMutation = useCreateVirtualFolder()
  const updateMutation = useUpdateVirtualFolder()
  const deleteMutation = useDeleteVirtualFolder()

  return (
    <div class="page">
      <VirtualFolders
        folders={foldersQuery.data ?? []}
        loading={foldersQuery.isLoading}
        createMutation={createMutation}
        updateMutation={updateMutation}
        deleteMutation={deleteMutation}
      />
    </div>
  )
}
