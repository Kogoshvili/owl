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
  getProcessingStats,
  getFileDetail,
  upsertComment,
  deleteComment,
  getFileExtensions,
  listSuggestions,
  getSuggestionDetail,
  createSuggestion,
  updateSuggestion,
  deleteSuggestion,
  addFilesToSuggestion,
  removeFileFromSuggestion,
  acceptSuggestion,
  listPhysicalFolders,
  listFolderGuards,
  setFolderGuard,
  listFolderSuggestions,
  generateSuggestions,
  dismissSuggestion,
  refineSuggestion,
  refineAllSuggestions,
  listStrategies,
  runGuard,
  extractOrphans,
  getGuardStatus,
  getLlmStatus,
  getScanStatus,
  getExtractStatus,
  getGenerationStatus,
} from "../api"
import type { FileListFilterState, RunningStatus } from "../api"

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
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["watchedDirs"] })
      qc.invalidateQueries({ queryKey: ["scanStatus"] })
    },
  })
}

export function useScanDir() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => scanDir(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["watchedDirs"] })
      qc.invalidateQueries({ queryKey: ["scanStatus"] })
    },
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

export function useProcessingStats() {
  return useQuery({
    queryKey: ["processingStats"],
    queryFn: getProcessingStats,
    refetchInterval: 5000,
  })
}

export function useAcceptSuggestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, destination, name }: { id: number; destination?: string; name?: string }) =>
      acceptSuggestion(id, destination, name),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["suggestions"] })
      qc.invalidateQueries({ queryKey: ["folderSuggestions"] })
      qc.invalidateQueries({ queryKey: ["suggestion"] })
    },
  })
}

export function useRefineSuggestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => refineSuggestion(id),
    onSuccess: () => {
      setTimeout(() => {
        qc.invalidateQueries({ queryKey: ["suggestions"] })
        qc.invalidateQueries({ queryKey: ["suggestion", id] })
      }, 3000)
    },
  })
}

export function useRefineAllSuggestions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => refineAllSuggestions(),
    onSuccess: () => {
      setTimeout(() => {
        qc.invalidateQueries({ queryKey: ["folderSuggestions"] })
        qc.invalidateQueries({ queryKey: ["suggestions"] })
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

export function useSuggestions() {
  return useQuery({
    queryKey: ["suggestions"],
    queryFn: listSuggestions,
  })
}

export function useCreateSuggestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ name, description }: { name: string; description: string }) =>
      createSuggestion(name, description),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["suggestions"] }),
  })
}

export function useUpdateSuggestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name, description }: { id: number; name?: string; description?: string }) =>
      updateSuggestion(id, name, description),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: ["suggestions"] })
      qc.invalidateQueries({ queryKey: ["suggestion", id] })
    },
  })
}

export function useDeleteSuggestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteSuggestion(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["suggestions"] }),
  })
}

export function useSuggestionDetail(id: number | null) {
  return useQuery({
    queryKey: ["suggestion", id],
    queryFn: () => getSuggestionDetail(id!),
    enabled: id !== null,
  })
}

export function useAddFilesToSuggestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ suggestionId, fileIds }: { suggestionId: number; fileIds: number[] }) =>
      addFilesToSuggestion(suggestionId, fileIds),
    onSuccess: (_data, { suggestionId }) => {
      qc.invalidateQueries({ queryKey: ["suggestion", suggestionId] })
    },
  })
}

export function useRemoveFileFromSuggestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ suggestionId, fileId }: { suggestionId: number; fileId: number }) =>
      removeFileFromSuggestion(suggestionId, fileId),
    onSuccess: (_data, { suggestionId }) => {
      qc.invalidateQueries({ queryKey: ["suggestion", suggestionId] })
    },
  })
}

export function useFolderSuggestions() {
  return useQuery({
    queryKey: ["folderSuggestions"],
    queryFn: listFolderSuggestions,
  })
}

export function useGenerateSuggestions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (params?: Parameters<typeof generateSuggestions>[0]) =>
      generateSuggestions(params),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["genStatus"] })
      qc.invalidateQueries({ queryKey: ["folderSuggestions"] })
      qc.invalidateQueries({ queryKey: ["suggestions"] })
    },
  })
}

export function useDismissSuggestion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => dismissSuggestion(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["folderSuggestions"] }),
  })
}

export function useRunGuard() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => runGuard(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["guardStatus"] })
      qc.invalidateQueries({ queryKey: ["folderGuards"] })
      qc.invalidateQueries({ queryKey: ["physicalFolders"] })
      qc.invalidateQueries({ queryKey: ["processingStats"] })
    },
  })
}

export function useExtractOrphans() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => extractOrphans(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["extractStatus"] })
      qc.invalidateQueries({ queryKey: ["unprocessedCount"] })
      qc.invalidateQueries({ queryKey: ["processingStats"] })
    },
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

function useRunningStatus(key: string, fn: () => Promise<RunningStatus>) {
  return useQuery({
    queryKey: [key],
    queryFn: fn,
    refetchInterval: (data) => (data?.running ? 2000 : 30000),
  })
}

export function useScanStatus() {
  return useRunningStatus("scanStatus", getScanStatus)
}

export function useExtractStatus() {
  return useRunningStatus("extractStatus", getExtractStatus)
}

export function useGuardStatus() {
  return useRunningStatus("guardStatus", getGuardStatus)
}

export function useGenStatus() {
  return useRunningStatus("genStatus", getGenerationStatus)
}

export function useLlmStatus() {
  return useQuery({
    queryKey: ["llmStatus"],
    queryFn: getLlmStatus,
    refetchInterval: 30000,
  })
}


