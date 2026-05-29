import { useState } from "preact/hooks"
import { useFileDetail, useUpsertComment, useDeleteComment, useExtractFile } from "../hooks/queries"
import { useToast } from "../hooks/toast"
import { getFileRawUrl, type File } from "../api"

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
  return d.toLocaleString()
}

const IMAGE_EXTS = new Set([".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg"])
const TEXT_EXTS = new Set([".txt", ".md", ".log", ".csv", ".json", ".xml", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf", ".sh", ".bat", ".ps1", ".py", ".js", ".ts", ".go", ".rs", ".java", ".c", ".cpp", ".h", ".hpp", ".rb", ".php", ".sql", ".env", ".gitignore", ".html", ".htm", ".css", ".scss"])

function isImageViewable(ext: string): boolean {
  return IMAGE_EXTS.has(ext.toLowerCase())
}

function isTextViewable(ext: string): boolean {
  return TEXT_EXTS.has(ext.toLowerCase())
}

export function FileDetailPage({ id }: { id?: string }) {
  const fileId = parseInt(id || "0", 10)
  const query = useFileDetail(fileId)
  const [showExtracted, setShowExtracted] = useState(false)

  if (query.isLoading) return <div class="page"><div class="empty">Loading...</div></div>
  if (query.isError) return <div class="page"><div class="empty">Error: {query.error?.message}</div></div>
  if (!query.data) return <div class="page"><div class="empty">File not found</div></div>

  const { file, comment, extracted_content } = query.data

  return (
    <div class="page">
      <div class="detail-back">
        <button class="btn btn-sm" onClick={() => window.history.back()}>Back</button>
      </div>

      {file.processing_status === "failed" && file.processing_error && (
        <div class="detail-error-banner">
          <strong>Extraction failed:</strong> {file.processing_error}
        </div>
      )}

      <MetadataSection file={file} />
      <FileViewer file={file} />
      <ExtractedContentPreview content={extracted_content} show={showExtracted} onToggle={() => setShowExtracted(!showExtracted)} />
      <CommentSection fileId={file.id} comment={comment} />
    </div>
  )
}

function MetadataSection({ file }: { file: File }) {
  const toast = useToast()
  const extractMutation = useExtractFile()

  const handleExtract = async () => {
    try {
      await extractMutation.mutateAsync(file.id)
      toast.show({ type: "success", message: "File queued for extraction" })
    } catch (e: any) {
      toast.show({ type: "error", message: e.message })
    }
  }

  const metaEntries: { label: string, value: string }[] = [
    { label: "Path", value: file.path },
    { label: "Extension", value: file.extension || "-" },
    { label: "MIME Type", value: file.mime_type || "-" },
    { label: "Size", value: formatBytes(file.size) },
    { label: "Status", value: file.status },
    { label: "Modified", value: file.modified_at ? new Date(file.modified_at).toLocaleString() : "-" },
    { label: "Indexed", value: file.indexed_at ? new Date(file.indexed_at).toLocaleString() : "-" },
    { label: "Content Indexed", value: file.content_indexed_at ? new Date(file.content_indexed_at).toLocaleString() : "-" },
    { label: "Processing", value: file.processing_status },
  ]

  if (file.processing_error) {
    metaEntries.push({ label: "Error", value: file.processing_error })
  }

  if (file.file_metadata) {
    for (const [key, val] of Object.entries(file.file_metadata)) {
      metaEntries.push({ label: key, value: String(val) })
    }
  }

  return (
    <section class="detail-section">
      <div class="detail-section-header">
        <h2>File Info</h2>
        {file.processing_status !== "processed" && file.processing_status !== "processing" && file.processing_status !== "queued" && (
          <button
            class="btn btn-sm"
            disabled={extractMutation.isPending}
            onClick={handleExtract}
          >
            {extractMutation.isPending ? "Queuing..." : "Extract"}
          </button>
        )}
      </div>
      <div class="detail-meta-grid">
        {metaEntries.map(({ label, value }) => (
          <div class="detail-meta-row" key={label}>
            <span class="detail-meta-label">{label}</span>
            <span class="detail-meta-value" title={value}>{value}</span>
          </div>
        ))}
      </div>
    </section>
  )
}

function FileViewer({ file }: { file: File }) {
  if (isImageViewable(file.extension)) {
    return (
      <section class="detail-section">
        <h2>Preview</h2>
        <div class="detail-image-preview">
          <img src={getFileRawUrl(file.id)} alt={file.name} />
        </div>
      </section>
    )
  }

  if (isTextViewable(file.extension)) {
    return <TextViewer fileId={file.id} />
  }

  return null
}

function TextViewer({ fileId }: { fileId: number }) {
  const [text, setText] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [shown, setShown] = useState(false)

  const loadText = () => {
    if (shown) {
      setShown(false)
      return
    }
    if (text !== null) {
      setShown(true)
      return
    }
    setLoading(true)
    setError(null)
    fetch(getFileRawUrl(fileId))
      .then(res => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        return res.text()
      })
      .then(t => {
        setText(t)
        setShown(true)
      })
      .catch(err => setError(err.message))
      .finally(() => setLoading(false))
  }

  return (
    <section class="detail-section">
      <div class="detail-section-header">
        <h2>File Content</h2>
        <button class="btn btn-sm" onClick={loadText} disabled={loading}>
          {loading ? "Loading..." : shown ? "Hide" : "Show"}
        </button>
      </div>
      {error && <div class="error-msg">{error}</div>}
      {shown && text !== null && (
        <pre class="detail-text-content">{text}</pre>
      )}
    </section>
  )
}

function ExtractedContentPreview({ content, show, onToggle }: { content: string | null; show: boolean; onToggle: () => void }) {
  if (!content) return null

  return (
    <section class="detail-section">
      <div class="detail-section-header">
        <h2>Extracted Content (Debug)</h2>
        <button class="btn btn-sm" onClick={onToggle}>
          {show ? "Hide" : "Show"}
        </button>
      </div>
      {show && (
        <pre class="detail-text-content detail-extracted">{content}</pre>
      )}
    </section>
  )
}

function CommentSection({ fileId, comment }: { fileId: number; comment: { content: string } | null }) {
  const toast = useToast()
  const [text, setText] = useState(comment?.content || "")
  const [editing, setEditing] = useState(!comment)
  const upsertMutation = useUpsertComment()
  const deleteMutation = useDeleteComment()

  const handleSave = async () => {
    if (!text.trim()) return
    try {
      await upsertMutation.mutateAsync({ fileId, content: text })
      setEditing(false)
      toast.show({ type: "success", message: "Comment saved" })
    } catch (e: any) {
      toast.show({ type: "error", message: e.message })
    }
  }

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(fileId)
      setText("")
      setEditing(true)
      toast.show({ type: "success", message: "Comment deleted" })
    } catch (e: any) {
      toast.show({ type: "error", message: e.message })
    }
  }

  return (
    <section class="detail-section">
      <div class="detail-section-header">
        <h2>Comment</h2>
        {comment && editing && (
          <button class="btn btn-sm btn-danger" onClick={handleDelete} disabled={deleteMutation.isPending}>
            Delete
          </button>
        )}
      </div>
      {editing ? (
        <div class="detail-comment-form">
          <textarea
            class="detail-comment-input"
            value={text}
            onInput={(e) => setText((e.target as HTMLTextAreaElement).value)}
            placeholder="Add a comment..."
            rows={3}
          />
          <div class="detail-comment-actions">
            <button class="btn btn-sm btn-primary" onClick={handleSave} disabled={upsertMutation.isPending || !text.trim()}>
              {upsertMutation.isPending ? "Saving..." : "Save"}
            </button>
            {comment && (
              <button class="btn btn-sm" onClick={() => { setText(comment.content); setEditing(false) }}>
                Cancel
              </button>
            )}
          </div>
        </div>
      ) : (
        <div class="detail-comment-display" onClick={() => setEditing(true)}>
          {comment?.content ? <p>{comment.content}</p> : <p class="empty">No comment. Click to add one.</p>}
        </div>
      )}
    </section>
  )
}
