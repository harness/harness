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
    webhookId = '',
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
    path: `/api/v1/repos/${space}/${repoName}/+/`,
    lazy: !repoName
  })
  const defaultBranch = repoMetadata?.default_branch || ''

  return {
    space,
    repoName,
    repoMetadata: repoName ? repoMetadata || undefined : undefined,
    error: getErrorMessage(error),
    loading,
    refetch,
    response,
    gitRef: gitRef || defaultBranch,
    resourcePath,
    commitRef,
    diffRefs: diffRefsToRefs(diffRefs || makeDiffRefs(defaultBranch, defaultBranch)),
    pullRequestId,
    webhookId,
    ...otherPathParams
  }
}

// TODO: Repository metadata is rarely changed. It might be good to implement
// some caching strategy in here
