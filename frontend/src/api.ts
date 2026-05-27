export interface WatchedDir {
  id: number
  path: string
  recursive: boolean
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

export interface Tag {
  id: number
  name: string
  source: string
  description: string
  created_at: string
}

export interface TagWithCount {
  id: number
  name: string
  source: string
  description: string
  file_count: number
  created_at: string
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
  tags: Tag[]
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

export function addWatchedDir(path: string, recursive = true): Promise<WatchedDir> {
  return request<WatchedDir>("/watched-directories", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, recursive }),
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
  sort?: string
  order?: string
  limit?: number
  offset?: number
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

export function listFileTags(fileId: number): Promise<Tag[]> {
  return request<Tag[]>(`/files/${fileId}/tags`)
}

export function addFileTag(fileId: number, name: string): Promise<Tag> {
  return request<Tag>(`/files/${fileId}/tags`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  })
}

export function removeFileTag(fileId: number, tagId: number): Promise<void> {
  return request<void>(`/files/${fileId}/tags/${tagId}`, { method: "DELETE" })
}

export interface VirtualFolder {
  id: number
  name: string
  description: string
  source: string
  materialized: boolean
  materialized_path: string | null
  created_at: string
}

export interface Note {
  id: number
  title: string
  content: string
  materialized: boolean
  materialized_path: string | null
  created_at: string
  updated_at: string
}

export interface VirtualFolderDetail {
  id: number
  name: string
  description: string
  source: string
  materialized: boolean
  materialized_path: string | null
  created_at: string
  files: File[]
  notes: Note[]
}

export function getVirtualFolders(source?: string): Promise<VirtualFolder[]> {
  const p = new URLSearchParams()
  if (source) p.set("source", source)
  const qs = p.toString()
  return request<VirtualFolder[]>(`/virtual-folders${qs ? "?" + qs : ""}`)
}

export function createVirtualFolder(name: string, description?: string): Promise<VirtualFolder> {
  return request<VirtualFolder>("/virtual-folders", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, description: description || "" }),
  })
}

export function updateVirtualFolder(id: number, name?: string, description?: string): Promise<VirtualFolder> {
  return request<VirtualFolder>(`/virtual-folders/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, description }),
  })
}

export function deleteVirtualFolder(id: number): Promise<void> {
  return request<void>(`/virtual-folders/${id}`, { method: "DELETE" })
}

export function getVirtualFolderDetail(id: number): Promise<VirtualFolderDetail> {
  return request<VirtualFolderDetail>(`/virtual-folders/${id}`)
}

export function addFilesToFolder(folderId: number, fileIds: number[], source?: string): Promise<{status: string}> {
  return request<{status: string}>(`/virtual-folders/${folderId}/files`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ file_ids: fileIds, source: source || "manual" }),
  })
}

export function removeFileFromFolder(folderId: number, fileId: number): Promise<void> {
  return request<void>(`/virtual-folders/${folderId}/files/${fileId}`, { method: "DELETE" })
}

export interface SearchFileResult {
  file_id: number
  name: string
  path: string
  extension: string
  rank: number
  snippet: string
  match_sources: string[]
}

export interface SearchNoteResult {
  note_id: number
  title: string
  rank: number
  snippet: string
  match_sources: string[]
}

export interface SearchResults {
  files: SearchFileResult[]
  notes: SearchNoteResult[]
}

export const ALL_SCOPES = ["filenames", "content", "comments", "tags", "notes"] as const
export type SearchScope = typeof ALL_SCOPES[number]

export function searchFiles(query: string, scopes?: SearchScope[]): Promise<SearchResults> {
  const params = new URLSearchParams({ q: query })
  if (scopes && scopes.length < ALL_SCOPES.length) {
    params.set("scopes", scopes.join(","))
  }
  return request<SearchResults>(`/search?${params}`)
}

export interface TagWithCount {
  id: number
  name: string
  source: string
  file_count: number
  created_at: string
}

export interface FolderSuggestion {
  id: number
  name: string
  description: string
  file_count: number
  preview: string[]
  created_at: string
}

export function tagFile(id: number): Promise<{tags: Tag[]}> {
  return request<{tags: Tag[]}>(`/intelligence/files/${id}/tag`, { method: "POST" })
}

export function tagFiles(params?: {
  watched_dir_id?: number
  extension?: string
  processing_status?: string
  limit?: number
}): Promise<{count: number, tagged: number, tag_count: number}> {
  const p = new URLSearchParams()
  if (params?.watched_dir_id) p.set("watched_dir_id", String(params.watched_dir_id))
  if (params?.extension) p.set("extension", params.extension)
  if (params?.processing_status) p.set("processing_status", params.processing_status)
  if (params?.limit) p.set("limit", String(params.limit))
  return request<{count: number, tagged: number, tag_count: number}>(`/intelligence/files/tag?${p}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params || {}),
  })
}

export function tagWatchedDir(id: number): Promise<{count: number, tagged: number, tag_count: number}> {
  return request<{count: number, tagged: number, tag_count: number}>(`/intelligence/watched-directories/${id}/tag`, {
    method: "POST",
  })
}

export function listFolderSuggestions(): Promise<Record<string, FolderSuggestion>> {
  return request<Record<string, FolderSuggestion>>("/intelligence/folders/suggestions")
}

export function generateFolderSuggestions(params?: {
  name?: string
  description?: string
  min_files?: number
  min_similarity?: number
}): Promise<{created: VirtualFolder[]}> {
  return request<{created: VirtualFolder[]}>(`/intelligence/folders/suggestions`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params || {}),
  })
}

export function acceptFolderSuggestion(id: number): Promise<VirtualFolder> {
  return request<VirtualFolder>(`/intelligence/folders/suggestions/${id}/accept`, {
    method: "POST",
  })
}

export function dismissFolderSuggestion(id: number): Promise<void> {
  return request<void>(`/intelligence/folders/suggestions/${id}`, {
    method: "DELETE",
  })
}

export function listTags(source?: "auto" | "manual"): Promise<TagWithCount[]> {
  const p = new URLSearchParams()
  if (source) p.set("source", source)
  return request<TagWithCount[]>(`/intelligence/tags?${p}`)
}

export function createTag(name: string): Promise<Tag> {
  return request<Tag>("/intelligence/tags", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  })
}

export function listTagFiles(id: number, params?: FileListParams): Promise<FileListResponse> {
  return request<FileListResponse>(`/intelligence/tags/${id}/files?${buildFileParams(params)}`)
}

export function deleteTag(id: number): Promise<void> {
  return request<void>(`/intelligence/tags/${id}`, {
    method: "DELETE",
  })
}

export function acceptTag(id: number): Promise<Tag> {
  return request<Tag>(`/intelligence/tags/${id}/accept`, {
    method: "POST",
  })
}

export function refineFolder(id: number): Promise<{related: boolean, action: string, folder?: VirtualFolder}> {
  return request<{related: boolean, action: string, folder?: VirtualFolder}>(`/intelligence/refine/folder/${id}`, {
    method: "POST",
  })
}

export function refineTag(id: number): Promise<{meaningful: boolean, action: string, tag?: Tag}> {
  return request<{meaningful: boolean, action: string, tag?: Tag}>(`/intelligence/refine/tag/${id}`, {
    method: "POST",
  })
}
