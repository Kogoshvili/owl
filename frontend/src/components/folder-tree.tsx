import { useState } from "preact/hooks"
import type { PhysicalFolder } from "../api"

interface Props {
  roots: PhysicalFolder[]
  onSelectFolder: (path: string) => void
  selectedPath: string | null
  coherenceMap?: Record<string, { is_coherent: boolean; avg_similarity: number; file_count: number }>
}

export function FolderTree({ roots, onSelectFolder, selectedPath, coherenceMap }: Props) {
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
        />
      ))}
    </div>
  )
}

function FolderTreeNode({ folder, depth, onSelect, selectedPath, coherenceMap }: {
  folder: PhysicalFolder
  depth: number
  onSelect: (path: string) => void
  selectedPath: string | null
  coherenceMap?: Record<string, { is_coherent: boolean; avg_similarity: number; file_count: number }>
}) {
  const [expanded, setExpanded] = useState(depth < 2)
  const hasChildren = folder.children && folder.children.length > 0
  const isSelected = selectedPath === folder.path
  const coherence = coherenceMap?.[folder.path]
  const showCoherence = depth > 0 && coherence

  const toggleExpanded = () => {
    if (hasChildren) {
      setExpanded(!expanded)
    }
  }

  return (
    <div class="folder-tree-node">
      <div
        class={`folder-tree-row ${isSelected ? "selected" : ""}`}
        style={{ "--depth": String(depth) } as any}
        onClick={toggleExpanded}
      >
        <span class={`folder-tree-toggle ${hasChildren ? "" : "invisible"}`}>
          {expanded ? "▾" : "▸"}
        </span>
        <span class="folder-tree-icon">{hasChildren ? (expanded ? "📂" : "📁") : "📄"}</span>
        <span class="folder-tree-name">{folder.name}</span>
        {folder.file_count > 0 && (
          <span class="folder-tree-count">({folder.file_count})</span>
        )}
        {showCoherence && (
          <span class={`folder-coherence-badge ${coherence.is_coherent ? "coherent" : "incoherent"}`}>
            {coherence.is_coherent ? "coherent" : "mixed"}
          </span>
        )}
      </div>
      {expanded && hasChildren && (
        <div class="folder-tree-children">
          {folder.children!.map((child) => (
            <FolderTreeNode
              key={child.path}
              folder={child}
              depth={depth + 1}
              onSelect={onSelect}
              selectedPath={selectedPath}
              coherenceMap={coherenceMap}
            />
          ))}
        </div>
      )}
    </div>
  )
}
