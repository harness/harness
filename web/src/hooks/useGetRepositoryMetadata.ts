import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import type { CODEProps } from 'RouteDefinitions'
import type { TypesRepository } from 'services/code'
import { diffRefsToRefs, makeDiffRefs } from 'utils/GitUtils'
import { getErrorMessage } from 'utils/Utils'
import { useGetSpaceParam } from './useGetSpaceParam'

export function useGetRepositoryMetadata() {
  const space = useGetSpaceParam()
  const {
    repoName,
    gitRef,
    resourcePath = '',
    commitRef = '',
    pullRequestId = '',
    diffRefs,
    ...otherPathParams
  } = useParams<CODEProps>()
  const {
    data: repoMetadata,
    error,
    loading,
    refetch,
    response
  } = useGet<TypesRepository>({
    path: `/api/v1/repos/${space}/${repoName}/+/`
  })
  const defaultBranch = repoMetadata?.defaultBranch || ''

  return {
    space,
    repoName,
    repoMetadata: repoMetadata || undefined,
    error: getErrorMessage(error),
    loading,
    refetch,
    response,
    gitRef: gitRef || defaultBranch,
    resourcePath,
    commitRef,
    diffRefs: diffRefsToRefs(diffRefs || makeDiffRefs(defaultBranch, defaultBranch)),
    pullRequestId,
    ...otherPathParams
  }
}

// TODO: Repository metadata is rarely changed. It might be good to implement
// some caching strategy in here
