import { useQuery, useMutation, useQueryClient } from "@tanstack/preact-query"
import {
  getWatchedDirs,
  getAllFiles,
  getFilesByDir,
  addWatchedDir,
  scanDir,
  deleteDir,
} from "../api"

export function useWatchedDirs() {
  return useQuery({
    queryKey: ["watchedDirs"],
    queryFn: getWatchedDirs,
  })
}

export function useAllFiles() {
  return useQuery({
    queryKey: ["files"],
    queryFn: () => getAllFiles(),
  })
}

export function useFilesByDir(dirId: number | null) {
  return useQuery({
    queryKey: ["files", "dir", dirId],
    queryFn: () => getFilesByDir(dirId!),
    enabled: dirId !== null,
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
