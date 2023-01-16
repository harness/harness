import { useMemo } from 'react'
import { useGet } from 'restful-react'
import type { OpenapiGetContentOutput } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'

interface UseGetResourceContentParams
  extends Optional<Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath'>, 'repoMetadata'> {
  includeCommit?: boolean
  lazy?: boolean
}

export function useGetResourceContent({
  repoMetadata,
  gitRef,
  resourcePath,
  includeCommit = false,
  lazy = false
}: UseGetResourceContentParams) {
  const { data, error, loading, refetch, response } = useGet<OpenapiGetContentOutput>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/content${resourcePath ? '/' + resourcePath : ''}`,
    queryParams: {
      include_commit: String(includeCommit),
      git_ref: gitRef
    },
    lazy: !repoMetadata || lazy
  })
  const isRepositoryEmpty = useMemo(
    () => (repoMetadata && resourcePath === '' && error && response?.status === 404) || false,
    [repoMetadata, resourcePath, error, response]
  )

  return { data, error: isRepositoryEmpty ? undefined : error, loading, refetch, response, isRepositoryEmpty }
}
