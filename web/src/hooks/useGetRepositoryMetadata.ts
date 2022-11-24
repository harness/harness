import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import type { CODEPathProps } from 'RouteDefinitions'
import type { TypesRepository } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import { useGetSpaceParam } from './useGetSpaceParam'

export function useGetRepositoryMetadata() {
  const space = useGetSpaceParam()
  const { repoName, gitRef, resourcePath = '', commitRef = '', ...otherPathParams } = useParams<CODEPathProps>()
  const {
    data: repoMetadata,
    error,
    loading,
    refetch,
    response
  } = useGet<TypesRepository>({
    path: `/api/v1/repos/${space}/${repoName}/+/`
  })

  return {
    space,
    repoName,
    repoMetadata: repoMetadata || undefined,
    error: getErrorMessage(error),
    loading,
    refetch,
    response,
    gitRef: gitRef || repoMetadata?.defaultBranch || '',
    resourcePath,
    commitRef,
    ...otherPathParams
  }
}

// TODO: Repository metadata is rarely changed. It might be good to implement
// some caching strategy in here
