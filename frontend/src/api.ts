export interface WatchedDir {
  id: number
  path: string
  enabled: boolean
  last_scanned_at: string | null
  created_at: string
}

export interface File {
  id: number
  path: string
  name: string
  extension: string
  mime_type: string
  size: number
  parent_dir: string
  watched_dir_id: number | null
  status: string
  modified_at: string
  indexed_at: string | null
  content_indexed_at: string | null
  processing_status: string
  processing_error: string | null
  file_metadata: Record<string, unknown> | null
  processable?: boolean | null
}

export interface Comment {
  id: number
  file_id: number
  content: string
  created_at: string
  updated_at: string
}

export interface FileDetail {
  file: File
  comment: Comment | null
  extracted_content: string | null
}

const isTauri = typeof window !== "undefined" && (window as any).__TAURI_INTERNALS__
const BASE = isTauri ? "http://127.0.0.1:3721/api" : "/api"

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, { cache: "no-store", ...options })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `HTTP ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  const json = await res.json()
  return json.data as T
}

export function getWatchedDirs(): Promise<WatchedDir[]> {
  return request<WatchedDir[]>("/watched-directories")
}

export function addWatchedDir(path: string): Promise<{id: number}> {
  return request<{id: number}>("/watched-directories", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  })
}

export function scanDir(id: number): Promise<WatchedDir> {
  return request<WatchedDir>(`/watched-directories/${id}/scan`, { method: "POST" })
}

export function deleteDir(id: number): Promise<void> {
  return request<void>(`/watched-directories/${id}`, { method: "DELETE" })
}

export interface FileListResponse {
  files: File[]
  total: number
  limit: number
  offset: number
}

export function getAllFiles(offset?: number): Promise<FileListResponse> {
  return request<FileListResponse>(`/files?limit=50&offset=${offset ?? 0}`)
}

export function extractFile(id: number): Promise<{status: string}> {
  return request<{status: string}>(`/files/${id}/extract`, {
    method: "POST",
  })
}

export function getFileDetail(id: number): Promise<FileDetail> {
  return request<FileDetail>(`/files/${id}`)
}

export function getFileRawUrl(id: number): string {
  return `${BASE}/files/${id}/raw`
}

export function upsertComment(fileId: number, content: string): Promise<Comment> {
  return request<Comment>(`/files/${fileId}/comment`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  })
}

export function deleteComment(fileId: number): Promise<void> {
  return request<void>(`/files/${fileId}/comment`, { method: "DELETE" })
}

export interface FolderSuggestion {
  id: number
  name: string
  description: string
  suggestion_type: string
  confidence: number
  created_at: string
}

export interface FolderSuggestionDetail {
  id: number
  name: string
  description: string
  suggestion_type: string
  confidence: number
  created_at: string
  files: File[]
}

export interface MaterializeResult {
  suggestion_id: number
  folder_path: string
  moved: number
  failed: string[]
}

export interface FolderSuggestionDisplay {
  id: number
  name: string
  description: string
  suggestion_type: string
  confidence: number
  file_count: number
  preview: string[]
  created_at: string
}

export function acceptSuggestion(id: number, destination?: string, name?: string): Promise<MaterializeResult> {
  return request<MaterializeResult>(`/suggestions/${id}/materialize`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ destination: destination || "", name: name || "" }),
  })
}

export function updateSuggestion(id: number, name?: string, description?: string): Promise<FolderSuggestion> {
  return request<FolderSuggestion>(`/suggestions/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, description }),
  })
}

export function deleteSuggestion(id: number): Promise<void> {
  return request<void>(`/suggestions/${id}`, { method: "DELETE" })
}

export function getSuggestionDetail(id: number): Promise<FolderSuggestionDetail> {
  return request<FolderSuggestionDetail>(`/suggestions/${id}`)
}

export function addFilesToSuggestion(suggestionId: number, fileIds: number[]): Promise<{status: string}> {
  return request<{status: string}>(`/suggestions/${suggestionId}/files`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ file_ids: fileIds }),
  })
}

export function removeFileFromSuggestion(suggestionId: number, fileId: number): Promise<void> {
  return request<void>(`/suggestions/${suggestionId}/files/${fileId}`, { method: "DELETE" })
}

export interface PhysicalFolder {
  path: string
  name: string
  depth: number
  file_count: number
  children?: PhysicalFolder[]
}

export function listPhysicalFolders(watchedDirId?: number): Promise<PhysicalFolder[]> {
  if (watchedDirId === undefined) {
    return request<PhysicalFolder[]>("/intelligence/folders/physical")
  }
  return request<PhysicalFolder[]>(`/intelligence/folders/physical?watched_dir_id=${watchedDirId}`)
}

export interface FolderGuardClassification {
  id: number
  path: string
  guarded: boolean
  reason: string
  source: "llm" | "user"
  classified_at: string
}

export function listFolderGuards(): Promise<FolderGuardClassification[]> {
  return request<FolderGuardClassification[]>("/intelligence/folders/guards")
}

export function setFolderGuard(path: string, guarded: boolean): Promise<{status: string}> {
  return request<{status: string}>("/intelligence/folders/guards", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, guarded }),
  })
}

export function listFolderSuggestions(): Promise<Record<string, FolderSuggestionDisplay>> {
  return request<Record<string, FolderSuggestionDisplay>>("/intelligence/folders/suggestions")
}

export function generateSuggestions(params?: {
  name?: string
  description?: string
  min_files?: number
  min_similarity?: number
  strategy?: string
}): Promise<{status: string, message: string}> {
  return request<{status: string, message: string}>("/intelligence/folders/suggestions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params || {}),
  })
}

export function dismissSuggestion(id: number): Promise<void> {
  return request<void>(`/intelligence/folders/suggestions/${id}`, {
    method: "DELETE",
  })
}

export function refineSuggestion(id: number): Promise<{status: string, id: number}> {
  return request<{status: string, id: number}>(`/intelligence/refine/folder/${id}`, {
    method: "POST",
  })
}

export function refineAllSuggestions(): Promise<{status: string, count: number}> {
  return request<{status: string, count: number}>(`/intelligence/folders/suggestions/refine-all`, {
    method: "POST",
  })
}

export function runGuard(): Promise<{status: string}> {
  return request<{status: string}>("/intelligence/guard/run", { method: "POST" })
}

export function getGuardStatus(): Promise<RunningStatus> {
  return request<RunningStatus>("/intelligence/guard/status")
}

export function getLlmStatus(): Promise<{llm_available: boolean; embedding_available: boolean}> {
  return request<{llm_available: boolean; embedding_available: boolean}>("/intelligence/llm/status")
}

export interface LlmSetupStatus {
  state: "not_started" | "downloading" | "starting" | "pulling_model" | "ready" | "error"
  message: string
  progress?: number
  total?: number
}

export function startLlmSetup(): Promise<{status: string}> {
  return request<{status: string}>("/intelligence/llm/setup", { method: "POST" })
}

export function getLlmSetupStatus(): Promise<LlmSetupStatus> {
  return request<LlmSetupStatus>("/intelligence/llm/setup-status")
}

export function extractOrphans(): Promise<{status: string}> {
  return request<{status: string}>("/intelligence/files/extract-orphans", { method: "POST" })
}

export function getGenerationStatus(): Promise<RunningStatus> {
  return request<RunningStatus>("/intelligence/folders/suggestions/status")
}

export function getScanStatus(): Promise<RunningStatus> {
  return request<RunningStatus>("/intelligence/scan/status")
}

export function getExtractStatus(): Promise<RunningStatus> {
  return request<RunningStatus>("/intelligence/extract/status")
}

export interface RunningStatus {
  running: boolean
  stage?: string
  progress?: number
  total?: number
  message?: string
  started_at?: string
  completed_at?: string
  error?: string
}

export interface ProcessingStats {
  total_files: number
  guarded: number
  open: number
  extractable: number
  queued: number
  processing: number
  processed: number
  failed: number
}

export function getProcessingStats(): Promise<ProcessingStats> {
  return request<ProcessingStats>("/intelligence/files/processing-stats")
}