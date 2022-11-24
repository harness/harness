import { useGet } from 'restful-react'
import type { OpenapiGetContentOutput } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'

interface UseGetResourceContentParams
  extends Optional<Pick<GitInfoProps, 'repoMetadata' | 'gitRef' | 'resourcePath'>, 'repoMetadata'> {
  includeCommit?: boolean
}

export function useGetResourceContent({
  repoMetadata,
  gitRef,
  resourcePath,
  includeCommit = false
}: UseGetResourceContentParams) {
  const { data, error, loading, refetch, response } = useGet<OpenapiGetContentOutput>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/content${resourcePath ? '/' + resourcePath : ''}`,
    queryParams: {
      include_commit: String(includeCommit),
      git_ref: gitRef
    },
    lazy: !repoMetadata
  })

  return { data, error, loading, refetch, response }
}
