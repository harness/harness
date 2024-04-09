import { useGet } from 'restful-react'
import type { TypesRepository } from 'services/code'

export function useGetImportProgress(repo: TypesRepository) {
  const { data: importStatus, loading } = useGet({
    path: `/api/v1/repos/${repo.path}/+/import-progress`,
    lazy: !repo.importing
  })
  //   const { data: importStatus, loading } = { data: {}, loading: false }

  return { importStatus, loading }
}
