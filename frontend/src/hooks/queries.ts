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
  getUnprocessedCount,
  searchFiles,
  getFileDetail,
  upsertComment,
  deleteComment,
  addFileTag,
  removeFileTag,
  listTags,
  listFileTags,
  getFileExtensions,
  getVirtualFolders,
  getVirtualFolderDetail,
  createVirtualFolder,
  updateVirtualFolder,
  deleteVirtualFolder,
  addFilesToFolder,
  removeFileFromFolder,
  listPhysicalFolders,
  listFolderGuards,
  setFolderGuard,
  tagFile,
  tagFiles,
  tagWatchedDir,
  listFolderSuggestions,
  generateFolderSuggestions,
  acceptFolderSuggestion,
  dismissFolderSuggestion,
  createTag,
  listTagFiles,
  deleteTag,
  acceptTag,
  refineFolder,
  refineAllFolderSuggestions,
  refineTag,
  listStrategies,
  type SearchScope,
} from "../api"
import type { FileListFilterState } from "../api"

export function useWatchedDirs() {
  return useQuery({
    queryKey: ["watchedDirs"],
    queryFn: getWatchedDirs,
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
    onSuccess: () => qc.invalidateQueries({ queryKey: ["watchedDirs"] }),
  })
}

export function useDeleteDir() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteDir(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["watchedDirs"] }),
  })
}

export function useExtractDir() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => extractDir(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["watchedDirs"] }),
  })
}

export function useExtractFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => extractFile(id),
  })
}

export function useFileExtensions() {
  return useQuery({
    queryKey: ["fileExtensions"],
    queryFn: getFileExtensions,
  })
}

export function useAllFiles(filters?: FileListFilterState) {
  const limit = filters?.limit ?? 50
  const offset = (filters?.page ?? 1) > 1 ? ((filters?.page ?? 1) - 1) * limit : 0

  return useQuery({
    queryKey: ["files", filters],
    queryFn: () => getAllFiles({
      extension: filters?.extension,
      processing_status: filters?.processing_status,
      supported: filters?.supported,
      sort: filters?.sort,
      order: filters?.order,
      limit,
      offset,
    }),
  })
}

export function useFilesByDir(dirId: number | null, filters?: FileListFilterState) {
  const limit = filters?.limit ?? 50
  const offset = (filters?.page ?? 1) > 1 ? ((filters?.page ?? 1) - 1) * limit : 0
  return useQuery({
    queryKey: ["files", "dir", dirId, filters],
    queryFn: () => getFilesByDir(dirId!, {
      extension: filters?.extension,
      processing_status: filters?.processing_status,
      supported: filters?.supported,
      sort: filters?.sort,
      order: filters?.order,
      limit,
      offset,
    }),
    enabled: dirId !== null,
  })
}

export function usePhysicalFolders() {
  return useQuery({
    queryKey: ["physicalFolders"],
    queryFn: () => listPhysicalFolders(),
  })
}

export function useUnprocessedCount() {
  return useQuery({
    queryKey: ["unprocessedCount"],
    queryFn: () => getUnprocessedCount(),
  })
}

export function useRefineFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => refineFolder(id),
    onSuccess: (_data, id) => {
      setTimeout(() => {
        qc.invalidateQueries({ queryKey: ["virtualFolders"] })
        qc.invalidateQueries({ queryKey: ["virtualFolder", id] })
      }, 3000)
    },
  })
}

export function useRefineAllFolderSuggestions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => refineAllFolderSuggestions(),
    onSuccess: () => {
      setTimeout(() => {
        qc.invalidateQueries({ queryKey: ["folderSuggestions"] })
        qc.invalidateQueries({ queryKey: ["virtualFolders"] })
      }, 3000)
    },
  })
}

export function useRefineTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => refineTag(id),
    onSuccess: () => {
      setTimeout(() => {
        qc.invalidateQueries({ queryKey: ["tags"] })
      }, 3000)
    },
  })
}

export function useFolderGuards() {
  return useQuery({
    queryKey: ["folderGuards"],
    queryFn: listFolderGuards,
  })
}

export function useSetFolderGuard() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ path, guarded }: { path: string; guarded: boolean }) =>
      setFolderGuard(path, guarded),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["folderGuards"] })
      queryClient.invalidateQueries({ queryKey: ["physicalFolders"] })
    },
  })
}

export function useVirtualFolders(source?: string) {
  return useQuery({
    queryKey: ["virtualFolders", source],
    queryFn: () => getVirtualFolders(source),
  })
}

export function useCreateVirtualFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, description }: { name: string; description: string }) =>
      createVirtualFolder(name, description),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["virtualFolders"] }),
  })
}

export function useUpdateVirtualFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name, description }: { id: number; name?: string; description?: string }) =>
      updateVirtualFolder(id, name, description),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: ["virtualFolders"] })
      qc.invalidateQueries({ queryKey: ["virtualFolder", id] })
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

export function useVirtualFolderDetail(id: number | null) {
  return useQuery({
    queryKey: ["virtualFolder", id],
    queryFn: () => getVirtualFolderDetail(id!),
    enabled: id !== null,
  })
}

export function useAddFilesToFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ folderId, fileIds, source }: { folderId: number; fileIds: number[]; source: string }) =>
      addFilesToFolder(folderId, fileIds, source),
    onSuccess: (_data, { folderId }) => {
      qc.invalidateQueries({ queryKey: ["virtualFolder", folderId] })
    },
  })
}

export function useRemoveFileFromFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ folderId, fileId }: { folderId: number; fileId: number }) =>
      removeFileFromFolder(folderId, fileId),
    onSuccess: (_data, { folderId }) => {
      qc.invalidateQueries({ queryKey: ["virtualFolder", folderId] })
    },
  })
}

export function useFolderSuggestions() {
  return useQuery({
    queryKey: ["folderSuggestions"],
    queryFn: listFolderSuggestions,
  })
}

export function useGenerateFolderSuggestions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params?: Parameters<typeof generateFolderSuggestions>[0]) =>
      generateFolderSuggestions(params),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["folderSuggestions"] })
      qc.invalidateQueries({ queryKey: ["virtualFolders"] })
    },
  })
}

export function useAcceptFolderSuggestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => acceptFolderSuggestion(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["folderSuggestions"] })
      qc.invalidateQueries({ queryKey: ["virtualFolders"] })
    },
  })
}

export function useDismissFolderSuggestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => dismissFolderSuggestion(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["folderSuggestions"] }),
  })
}

export function useFileDetail(id: number | null) {
  return useQuery({
    queryKey: ["file", id],
    queryFn: () => getFileDetail(id!),
    enabled: id !== null,
  })
}

export function useUpsertComment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ fileId, content }: { fileId: number; content: string }) =>
      upsertComment(fileId, content),
    onSuccess: (_data, { fileId }) => {
      qc.invalidateQueries({ queryKey: ["file", fileId] })
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

export function useListTags() {
  return useQuery({
    queryKey: ["tags"],
    queryFn: () => listTags(),
  })
}

export function useListFileTags(fileId: number) {
  return useQuery({
    queryKey: ["fileTags", fileId],
    queryFn: () => listFileTags(fileId),
  })
}

export function useTags(source?: string) {
  return useQuery({
    queryKey: ["tags", source],
    queryFn: () => listTags(source),
  })
}

export function useCreateTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, source }: { name: string; source: string }) =>
      createTag(name, source),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tags"] }),
  })
}

export function useDeleteTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteTag(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tags"] }),
  })
}

export function useAcceptTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => acceptTag(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tags"] }),
  })
}

export function useTagFilesList(id: number, filters?: FileListFilterState) {
  const limit = filters?.limit ?? 50
  const offset = (filters?.page ?? 1) > 1 ? ((filters?.page ?? 1) - 1) * limit : 0
  return useQuery({
    queryKey: ["tagFiles", id, filters],
    queryFn: () => listTagFiles(id, {
      extension: filters?.extension,
      processing_status: filters?.processing_status,
      sort: filters?.sort,
      order: filters?.order,
      limit,
      offset,
    }),
  })
}

export function useAddFileTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ fileId, tagId, source }: { fileId: number; tagId: number; source: string }) =>
      addFileTag(fileId, tagId, source),
  })
}

export function useRemoveFileTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ fileId, tagId }: { fileId: number; tagId: number }) =>
      removeFileTag(fileId, tagId),
  })
}

export function useTagFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (fileId: number) => tagFile(fileId),
  })
}

export function useTagFiles() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params?: Parameters<typeof tagFiles>[0]) => tagFiles(params),
  })
}

export function useTagWatchedDir() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => tagWatchedDir(id),
  })
}

export function useSearch(query: string, scopes: SearchScope[]) {
  return useQuery({
    queryKey: ["search", query, scopes],
    queryFn: () => searchFiles(query, scopes),
    enabled: query.length >= 2,
  })
}
