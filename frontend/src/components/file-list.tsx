import { type File as OwlFile } from "../api"
import { useExtractFile } from "../hooks/queries"
import { route } from "preact-router"

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
  const extractMutation = useExtractFile();

  const title = dirName ? `Files in ${dirName}` : "All Files";

  const getStatusClass = (status: string) => {
    switch (status) {
      case "unprocessed":
        return "status-unprocessed";
      case "queued":
        return "status-queued";
      case "processing":
        return "status-processing";
      case "processed":
        return "status-processed";
      case "stale":
        return "status-stale";
      case "failed":
        return "status-failed";
      default:
        return "status-other";
    }
  }

  const getStatusText = (file: OwlFile) => {
    if (file.processing_status === "unprocessed" && file.processable === false) {
      return "unsupported";
    }
    return file.processing_status;
  }

  const handleExtract = async (fileId: number) => {
    try {
      await extractMutation.mutateAsync(fileId);
      // Refresh the data after extraction is queued
      onRefresh();
    } catch (error) {
      console.error("Failed to extract file:", error);
    }
  };

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
              <th>File Status</th>
              <th>Processing</th>
              <th>Indexed</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {files.map((f: OwlFile) => (
              <tr key={f.id}>
                <td class="file-name" title={f.path}>
                  <a href={`/files/${f.id}`} class="file-link" onClick={(e) => { e.preventDefault(); route(`/files/${f.id}`) }}>{f.name}</a>
                </td>
                <td>{f.extension || "-"}</td>
                <td>{formatBytes(f.size)}</td>
                <td class="file-mime">{f.mime_type}</td>
                <td><span class={`status-badge status-${f.status}`}>{f.status}</span></td>
                <td>
                  <span class="status-badge-wrap">
                    <span
                      class={`status-badge ${getStatusClass(f.processing_status)}`}
                    >
                      {getStatusText(f)}
                    </span>
                    {f.processing_error && (
                      <span class="status-tooltip">{f.processing_error}</span>
                    )}
                  </span>
                </td>
                <td>{formatTime(f.indexed_at)}</td>
                <td>
                  <button 
                    class="btn btn-sm"
                    disabled={
                      f.processing_status === "processing" ||
                      f.processing_status === "processed" ||
                      f.processing_status === "queued" ||
                      (f.processing_status === "unprocessed" && f.processable === false)
                    }
                    onClick={() => handleExtract(f.id)}
                  >
                    Extract
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div class="file-count">{files.length} file{files.length !== 1 ? "s" : ""}</div>
    </div>
  )
}