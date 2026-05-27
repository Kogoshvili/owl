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
}

const BASE = "/api"

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, options)
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

export function getFilesByDir(dirId: number): Promise<File[]> {
  return request<File[]>(`/watched-directories/${dirId}/files`)
}

export function getAllFiles(limit = 200): Promise<File[]> {
  return request<File[]>(`/files?limit=${limit}`)
}
