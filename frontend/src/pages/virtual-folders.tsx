import { useState } from "preact/hooks"
import { useWatchedDirs, useVirtualFolders, useCreateVirtualFolder, useUpdateVirtualFolder, useDeleteVirtualFolder, usePhysicalFolders, useSmartSuggestions, useAcceptSmartSuggestion, useGenerateSmartSuggestions, useUnprocessedCount } from "../hooks/queries"
import { VirtualFolders } from "../components/virtual-folders"
import { FolderTree } from "../components/folder-tree"
import { SmartSuggestions } from "../components/smart-suggestions"
import type { SmartSuggestion } from "../api"

export function VirtualFoldersPage() {
  const [selectedWatchedDir, setSelectedWatchedDir] = useState<number | null>(null)
  const [showTree, setShowTree] = useState(true)
  const [showVirtual, setShowVirtual] = useState(false)
  const [suggestions, setSuggestions] = useState<SmartSuggestion[]>([])

  const watchedDirsQuery = useWatchedDirs()
  const physicalFoldersQuery = usePhysicalFolders(selectedWatchedDir)
  const smartSuggestionsQuery = useSmartSuggestions(selectedWatchedDir)
  const unprocessedCountQuery = useUnprocessedCount(selectedWatchedDir)
  const acceptSmartMutation = useAcceptSmartSuggestion()
  const generateMutation = useGenerateSmartSuggestions()

  const foldersQuery = useVirtualFolders("manual")
  const createMutation = useCreateVirtualFolder()
  const updateMutation = useUpdateVirtualFolder()
  const deleteMutation = useDeleteVirtualFolder()

  const watchedDirs = watchedDirsQuery.data ?? []

  const handleDismiss = (s: SmartSuggestion) => {
    setSuggestions((prev) => prev.filter(
      (item) => !(item.type === s.type && 
      (item.name === s.name || item.target_path === s.target_path || 
      item.source_paths?.join("+") === s.source_paths?.join("+")))
    ))
  }

  return (
    <div class="page vf-page-layout">
      <div class="vf-page-main">
        <div class="folder-selector">
          <label>Watched Directory:</label>
          <select
            value={selectedWatchedDir ?? ""}
            onChange={(e) => {
              const v = (e.target as HTMLSelectElement).value
              setSelectedWatchedDir(v ? Number(v) : null)
              setSuggestions([])
            }}
          >
            <option value="">Select a directory...</option>
            {watchedDirs.map((wd) => (
              <option key={wd.id} value={wd.id}>{wd.path}</option>
            ))}
          </select>
        </div>

        <div class="toggle-section" onClick={() => setShowTree(!showTree)}>
          <h3>
            <span class={`toggle-arrow ${showTree ? "expanded" : ""}`}>▸</span>
            Folders
          </h3>
        </div>

        {showTree && selectedWatchedDir && physicalFoldersQuery.data && (
          <div class="folder-tree-section">
            <FolderTree
              roots={physicalFoldersQuery.data}
              onSelectFolder={() => {}}
              selectedPath={null}
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
        <SmartSuggestions
          suggestions={suggestions.length > 0 ? suggestions : (smartSuggestionsQuery.data?.suggestions ?? [])}
          loading={smartSuggestionsQuery.isLoading || generateMutation.isPending}
          acceptMutation={acceptSmartMutation}
          generateMutation={generateMutation}
          watchedDirId={selectedWatchedDir}
          unprocessedCount={unprocessedCountQuery.data?.count}
          onDismiss={handleDismiss}
        />
      </div>
    </div>
  )
}
