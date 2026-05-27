import { useAllFiles } from "../hooks/queries"
import { FileList } from "../components/file-list"

export function FilesPage() {
  const filesQuery = useAllFiles()

  return (
    <div class="page">
      <FileList
        files={filesQuery.data ?? []}
        loading={filesQuery.isLoading}
        dirName={null}
        onRefresh={() => filesQuery.refetch()}
      />
    </div>
  )
}
