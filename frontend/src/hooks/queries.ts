import { useQuery, useMutation, useQueryClient } from "@tanstack/preact-query"
import {
  getWatchedDirs,
  getAllFiles,
  getFilesByDir,
  addWatchedDir,
  scanDir,
  deleteDir,
  extractDir,
  extractFile,
  searchFiles,
  getFileDetail,
  upsertComment,
  deleteComment,
  addFileTag,
  removeFileTag,
  listTags,
  getFileExtensions,
  getVirtualFolders,
  getVirtualFolderDetail,
  createVirtualFolder,
  updateVirtualFolder,
  deleteVirtualFolder,
  addFilesToFolder,
  removeFileFromFolder,
  type SearchScope,
} from "../api"
import type { FilterState } from "../components/file-list"

export function useWatchedDirs() {
  return useQuery({
    queryKey: ["watchedDirs"],
    queryFn: getWatchedDirs,
  })
}

export function useAllFiles(filters?: FilterState) {
  const limit = filters?.limit ?? 50
  const offset = (filters?.page ?? 1) > 1 ? ((filters?.page ?? 1) - 1) * limit : 0

  return useQuery({
    queryKey: ["files", filters],
    queryFn: () => getAllFiles({
      extension: filters?.extension,
      processing_status: filters?.processing_status,
      sort: filters?.sort,
      order: filters?.order,
      limit,
      offset,
    }),
  })
}

export function useFilesByDir(dirId: number | null, filters?: FilterState) {
  const limit = filters?.limit ?? 50
  const offset = (filters?.page ?? 1) > 1 ? ((filters?.page ?? 1) - 1) * limit : 0

  return useQuery({
    queryKey: ["files", "dir", dirId, filters],
    queryFn: () => getFilesByDir(dirId!, {
      extension: filters?.extension,
      processing_status: filters?.processing_status,
      sort: filters?.sort,
      order: filters?.order,
      limit,
      offset,
    }),
    enabled: dirId !== null,
  })
}

export function useFileExtensions() {
  return useQuery({
    queryKey: ["fileExtensions"],
    queryFn: getFileExtensions,
  })
}

export function useAddWatchedDir() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (path: string) => addWatchedDir(path),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["watchedDirs"] }),
  })
}

export function useScanDir() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => scanDir(id),
    onSuccess: (_data, id) => {
      setTimeout(() => {
        qc.invalidateQueries({ queryKey: ["watchedDirs"] })
        qc.invalidateQueries({ queryKey: ["files"] })
        qc.invalidateQueries({ queryKey: ["files", "dir", id] })
      }, 1000)
    },
  })
}

export function useDeleteDir() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteDir(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["watchedDirs"] })
      qc.invalidateQueries({ queryKey: ["files"] })
    },
  })
}

export function useExtractDir() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => extractDir(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ["watchedDirs"] })
      qc.invalidateQueries({ queryKey: ["files"] })
      qc.invalidateQueries({ queryKey: ["files", "dir", id] })
    },
  })
}

export function useExtractFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => extractFile(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["files"] })
    },
  })
}

export function useSearch(query: string, scopes?: SearchScope[]) {
  return useQuery({
    queryKey: ["search", query, scopes],
    queryFn: () => searchFiles(query, scopes),
    enabled: query.length >= 2,
  })
}

export function useFileDetail(id: number) {
  return useQuery({
    queryKey: ["file", id],
    queryFn: () => getFileDetail(id),
  })
}

export function useUpsertComment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ fileId, content }: { fileId: number; content: string }) =>
      upsertComment(fileId, content),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ["file", vars.fileId] })
    },
  })
}

export function useDeleteComment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (fileId: number) => deleteComment(fileId),
    onSuccess: (_data, fileId) => {
      qc.invalidateQueries({ queryKey: ["file", fileId] })
    },
  })
}

export function useTags() {
  return useQuery({
    queryKey: ["tags"],
    queryFn: listTags,
  })
}

export function useAddFileTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ fileId, name }: { fileId: number; name: string }) =>
      addFileTag(fileId, name),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ["file", vars.fileId] })
      qc.invalidateQueries({ queryKey: ["tags"] })
    },
  })
}

export function useRemoveFileTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ fileId, tagId }: { fileId: number; tagId: number }) =>
      removeFileTag(fileId, tagId),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ["file", vars.fileId] })
    },
  })
}

export function useVirtualFolders() {
  return useQuery({
    queryKey: ["virtualFolders"],
    queryFn: getVirtualFolders,
  })
}

export function useVirtualFolderDetail(id: number | null) {
  return useQuery({
    queryKey: ["virtualFolder", id],
    queryFn: () => getVirtualFolderDetail(id!),
    enabled: id !== null,
  })
}

export function useCreateVirtualFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, description }: { name: string; description?: string }) =>
      createVirtualFolder(name, description),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["virtualFolders"] }),
  })
}

export function useUpdateVirtualFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name, description }: { id: number; name?: string; description?: string }) =>
      updateVirtualFolder(id, name, description),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ["virtualFolders"] })
      qc.invalidateQueries({ queryKey: ["virtualFolder", vars.id] })
    },
  })
}

export function useDeleteVirtualFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteVirtualFolder(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["virtualFolders"] }),
  })
}

export function useAddFilesToFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ folderId, fileIds, source }: { folderId: number; fileIds: number[]; source?: string }) =>
      addFilesToFolder(folderId, fileIds, source),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ["virtualFolder", vars.folderId] })
      qc.invalidateQueries({ queryKey: ["virtualFolders"] })
    },
  })
}

export function useRemoveFileFromFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ folderId, fileId }: { folderId: number; fileId: number }) =>
      removeFileFromFolder(folderId, fileId),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ["virtualFolder", vars.folderId] })
    },
  })
}
