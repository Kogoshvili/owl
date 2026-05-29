import { useState, useEffect } from "preact/hooks"
import type { PhysicalFolder, File as OwlFile } from "../api"
import { listPhysicalFolderFiles } from "../api"

interface Props {
  roots: PhysicalFolder[]
  onSelectFolder: (path: string) => void
  selectedPath: string | null
  coherenceMap?: Record<string, { is_coherent: boolean; avg_similarity: number; file_count: number }>
  guardMap?: Record<string, boolean>
  onToggleGuard?: (path: string, guarded: boolean) => void
}

export function FolderTree({ roots, onSelectFolder, selectedPath, coherenceMap, guardMap, onToggleGuard }: Props) {
  return (
    <div class="folder-tree">
      {roots.map((root) => (
        <FolderTreeNode
          key={root.path}
          folder={root}
          depth={0}
          onSelect={onSelectFolder}
          selectedPath={selectedPath}
          coherenceMap={coherenceMap}
          guardMap={guardMap}
          onToggleGuard={onToggleGuard}
        />
      ))}
    </div>
  )
}

function FolderTreeNode({ folder, depth, onSelect, selectedPath, coherenceMap, guardMap, onToggleGuard }: {
  folder: PhysicalFolder
  depth: number
  onSelect: (path: string) => void
  selectedPath: string | null
  coherenceMap?: Record<string, { is_coherent: boolean; avg_similarity: number; file_count: number }>
  guardMap?: Record<string, boolean>
  onToggleGuard?: (path: string, guarded: boolean) => void
}) {
  const [expanded, setExpanded] = useState(false)
  const [files, setFiles] = useState<OwlFile[] | null>(null)
  const [loadingFiles, setLoadingFiles] = useState(false)
  const hasChildren = folder.children && folder.children.length > 0
  const canExpand = hasChildren || folder.file_count > 0
  const isSelected = selectedPath === folder.path
  const coherence = coherenceMap?.[folder.path]
  const showCoherence = depth > 0 && coherence
  const isGuarded = guardMap?.[folder.path]

  useEffect(() => {
    if (expanded && files === null && folder.file_count > 0 && !loadingFiles) {
      setLoadingFiles(true)
      listPhysicalFolderFiles(folder.path).then((res) => {
        setFiles(res.files)
      }).catch(() => {
        setFiles([])
      }).finally(() => {
        setLoadingFiles(false)
      })
    }
  }, [expanded, files, folder.path, folder.file_count, loadingFiles])

  const toggleExpanded = () => {
    setExpanded(!expanded)
  }

  const toggleGuard = (e: MouseEvent) => {
    e.stopPropagation()
    if (onToggleGuard) {
      onToggleGuard(folder.path, !isGuarded)
    }
  }

  return (
    <div class="folder-tree-node">
      <div
        class={`folder-tree-row ${isSelected ? "selected" : ""}`}
        style={{ "--depth": String(depth) } as any}
        onClick={toggleExpanded}
      >
        <span class={`folder-tree-toggle ${canExpand ? "" : "invisible"}`}>
          {expanded ? "▾" : "▸"}
        </span>
        <span class="folder-tree-icon">{canExpand ? (expanded ? "📂" : "📁") : "📄"}</span>
        <span class="folder-tree-name">{folder.name}</span>
        {folder.file_count > 0 && (
          <span class="folder-tree-count">({folder.file_count})</span>
        )}
        {showCoherence && (
          <span class={`folder-coherence-badge ${coherence.is_coherent ? "coherent" : "incoherent"}`}>
            {coherence.is_coherent ? "coherent" : "mixed"}
          </span>
        )}
        {onToggleGuard && depth > 0 && (
          <span
            class={`folder-guard-badge ${isGuarded ? "guarded" : "open"}`}
            onClick={toggleGuard}
            title={isGuarded ? "Guarded (click to unguard)" : "Open (click to guard)"}
          >
            {isGuarded ? "🔒" : "🔓"}
          </span>
        )}
      </div>
      {expanded && (
        <div class="folder-tree-children">
          {hasChildren && folder.children!.map((child) => (
            <FolderTreeNode
              key={child.path}
              folder={child}
              depth={depth + 1}
              onSelect={onSelect}
              selectedPath={selectedPath}
              coherenceMap={coherenceMap}
              guardMap={guardMap}
              onToggleGuard={onToggleGuard}
            />
          ))}
          {files?.map((f) => (
            <div class="folder-tree-node" key={f.id}>
              <div class="folder-tree-row" style={{ "--depth": String(depth + 1) } as any}>
                <span class="folder-tree-toggle invisible">▸</span>
                <span class="folder-tree-icon">📄</span>
                <span class="folder-tree-name">{f.name}</span>
              </div>
            </div>
          ))}
          {loadingFiles && (
            <div class="folder-tree-row" style={{ "--depth": String(depth + 1) } as any}>
              <span class="folder-tree-name">Loading files...</span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
