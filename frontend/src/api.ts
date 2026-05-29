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

const BASE = "/api"

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

export function addWatchedDir(path: string): Promise<WatchedDir> {
  return request<WatchedDir>("/watched-directories", {
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

export interface FileListParams {
  extension?: string
  status?: string
  processing_status?: string
  supported?: string
  sort?: string
  order?: string
  limit?: number
  offset?: number
}

export interface FileListFilterState {
  extension?: string
  processing_status?: string
  supported?: string
  sort: string
  order: string
  page: number
  limit: number
}

export interface FileListResponse {
  files: File[]
  total: number
  limit: number
  offset: number
}

export function getFilesByDir(dirId: number, params?: FileListParams): Promise<FileListResponse> {
  return request<FileListResponse>(`/watched-directories/${dirId}/files?${buildFileParams(params)}`)
}

export function getAllFiles(params?: FileListParams): Promise<FileListResponse> {
  return request<FileListResponse>(`/files?${buildFileParams(params)}`)
}

function buildFileParams(params?: FileListParams): string {
  const p = new URLSearchParams()
  if (params?.extension) p.set("extension", params.extension)
  if (params?.status) p.set("status", params.status)
  if (params?.processing_status) p.set("processing_status", params.processing_status)
  if (params?.supported) p.set("supported", params.supported)
  if (params?.sort) p.set("sort", params.sort)
  if (params?.order) p.set("order", params.order)
  p.set("limit", String(params?.limit ?? 50))
  if (params?.offset) p.set("offset", String(params.offset))
  return p.toString()
}

export function getFileExtensions(): Promise<string[]> {
  return request<string[]>("/files/extensions")
}

export function extractDir(id: number): Promise<{queued: number}> {
  return request<{queued: number}>(`/watched-directories/${id}/extract`, {
    method: "POST",
  })
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
  materialized_at: string | null
  materialized_path: string
  created_at: string
}

export interface FolderSuggestionDetail {
  id: number
  name: string
  description: string
  suggestion_type: string
  confidence: number
  materialized_at: string | null
  materialized_path: string
  created_at: string
  files: File[]
}

export interface MaterializeResult {
  suggestion_id: number
  folder_path: string
  moved: number
  failed: string[]
}

export function acceptSuggestion(id: number, destination?: string): Promise<MaterializeResult> {
  return request<MaterializeResult>(`/suggestions/${id}/materialize`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ destination: destination || "" }),
  })
}

export function listSuggestions(): Promise<FolderSuggestion[]> {
  return request<FolderSuggestion[]>("/suggestions")
}

export function createSuggestion(name: string, description?: string): Promise<FolderSuggestion> {
  return request<FolderSuggestion>("/suggestions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, description: description || "" }),
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

export interface TagWithCount {
  id: number
  name: string
  source: string
  file_count: number
  created_at: string
}

export interface FolderSuggestionDisplay {
  id: number
  name: string
  description: string
  suggestion_type: string
  confidence: number
  materialized_at: string | null
  materialized_path: string
  file_count: number
  preview: string[]
  created_at: string
}

export interface StrategyInfo {
  id: string
  display_name: string
  description: string
  available: boolean
  speed_hint: string
}

export interface PhysicalFolder {
  path: string
  name: string
  depth: number
  file_count: number
  children?: PhysicalFolder[]
}

export interface OutlierFile {
  id: number
  name: string
  avg_similarity_to_others: number
}

export interface FolderCoherence {
  path: string
  file_count: number
  avg_similarity: number
  is_coherent: boolean
  outlier_files: OutlierFile[]
}

export interface FolderGuardClassification {
  id: number
  path: string
  guarded: boolean
  reason: string
  source: "llm" | "user"
  classified_at: string
}

export function listStrategies(): Promise<StrategyInfo[]> {
  return request<StrategyInfo[]>("/intelligence/strategies")
}

export function listPhysicalFolders(watchedDirId?: number): Promise<PhysicalFolder[]> {
  if (watchedDirId === undefined) {
    return request<PhysicalFolder[]>("/intelligence/folders/physical")
  }
  return request<PhysicalFolder[]>(`/intelligence/folders/physical?watched_dir_id=${watchedDirId}`)
}

export function listPhysicalFolderFiles(path: string): Promise<{path: string, files: File[], count: number}> {
  return request<{path: string, files: File[], count: number}>(`/intelligence/folders/physical/files?path=${encodeURIComponent(path)}`)
}

export function analyzeFolderCoherence(path: string): Promise<FolderCoherence> {
  return request<FolderCoherence>(`/intelligence/folders/physical/coherence?path=${encodeURIComponent(path)}`)
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

export function getUnprocessedCount(watchedDirId?: number): Promise<{count: number}> {
  if (watchedDirId) {
    return request<{count: number}>(`/intelligence/files/unprocessed/count?watched_dir_id=${watchedDirId}`)
  }
  return request<{count: number}>("/intelligence/files/unprocessed/count")
}

export function runGuard(): Promise<{status: string}> {
  return request<{status: string}>("/intelligence/guard/run", { method: "POST" })
}

export function extractOrphans(): Promise<{status: string}> {
  return request<{status: string}>("/intelligence/files/extract-orphans", { method: "POST" })
}

export interface ProcessingStats {
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
