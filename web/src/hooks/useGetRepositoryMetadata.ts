import { useParams } from 'react-router-dom'
import { useGet } from 'restful-react'
import type { SCMPathProps } from 'RouteDefinitions'
import type { TypesRepository } from 'services/scm'
import { getErrorMessage } from 'utils/Utils'
import { useGetSpaceParam } from './useGetSpaceParam'

export function useGetRepositoryMetadata() {
  const space = useGetSpaceParam()
  const { repoName, ...otherPathParams } = useParams<SCMPathProps>()
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
    repoMetadata,
    error: getErrorMessage(error),
    loading,
    refetch,
    response,
    ...otherPathParams
  }
}
