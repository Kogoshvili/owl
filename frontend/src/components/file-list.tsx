import { type File as OwlFile } from "../api"

interface Props {
  files: OwlFile[]
  loading: boolean
  dirName: string | null
  onRefresh: () => void
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i]
}

function formatTime(iso: string | null): string {
  if (!iso) return "-"
  const d = new Date(iso)
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

export function FileList({ files, loading, dirName, onRefresh }: Props) {
  const title = dirName ? `Files in ${dirName}` : "All Files"

  return (
    <div class="file-list">
      <div class="file-list-header">
        <h2>{title}</h2>
        <button class="btn" onClick={onRefresh}>Refresh</button>
      </div>

      {loading ? (
        <div class="empty">Loading...</div>
      ) : files.length === 0 ? (
        <div class="empty">No files found</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Ext</th>
              <th>Size</th>
              <th>Type</th>
              <th>Status</th>
              <th>Indexed</th>
            </tr>
          </thead>
          <tbody>
            {files.map((f: OwlFile) => (
              <tr key={f.id}>
                <td class="file-name" title={f.path}>{f.name}</td>
                <td>{f.extension || "-"}</td>
                <td>{formatBytes(f.size)}</td>
                <td class="file-mime">{f.mime_type}</td>
                <td>
                  <span class={`status-badge status-${f.status}`}>{f.status}</span>
                </td>
                <td>{formatTime(f.indexed_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div class="file-count">{files.length} file{files.length !== 1 ? "s" : ""}</div>
    </div>
  )
}
