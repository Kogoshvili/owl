import { useState } from "preact/hooks"
import { useVirtualFolders, useCreateVirtualFolder, useUpdateVirtualFolder, useDeleteVirtualFolder, usePhysicalFolders, useFolderGuards, useSetFolderGuard } from "../hooks/queries"
import { VirtualFolders } from "../components/virtual-folders"
import { FolderTree } from "../components/folder-tree"
import { FolderSuggestions } from "../components/folder-suggestions"

export function VirtualFoldersPage() {
  const [showTree, setShowTree] = useState(true)
  const [showVirtual, setShowVirtual] = useState(false)

  const physicalFoldersQuery = usePhysicalFolders()
  const folderGuardsQuery = useFolderGuards()
  const setFolderGuardMutation = useSetFolderGuard()

  const foldersQuery = useVirtualFolders("manual")
  const createMutation = useCreateVirtualFolder()
  const updateMutation = useUpdateVirtualFolder()
  const deleteMutation = useDeleteVirtualFolder()

  const guardMap = folderGuardsQuery.data ? folderGuardsQuery.data.reduce((acc, g) => {
    acc[g.path] = g.guarded
    return acc
  }, {} as Record<string, boolean>) : {}

  const handleToggleGuard = (path: string, guarded: boolean) => {
    setFolderGuardMutation.mutate({ path, guarded })
  }

  return (
    <div class="page vf-page-layout">
      <div class="vf-page-main">
        <div class="toggle-section" onClick={() => setShowTree(!showTree)}>
          <h3>
            <span class={`toggle-arrow ${showTree ? "expanded" : ""}`}>▸</span>
            Folders
          </h3>
        </div>

        {showTree && physicalFoldersQuery.data && (
          <div class="folder-tree-section">
            <FolderTree
              roots={physicalFoldersQuery.data}
              onSelectFolder={() => {}}
              selectedPath={null}
              guardMap={guardMap}
              onToggleGuard={handleToggleGuard}
            />
          </div>
        )}

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
