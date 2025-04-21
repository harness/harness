/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { useAtom } from 'jotai'
import { useEffect, useMemo } from 'react'
import { useParams, useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import type { CODEProps } from 'RouteDefinitions'
import type { RepoRepositoryOutput } from 'services/code'
import { diffRefsToRefs, makeDiffRefs } from 'utils/GitUtils'
import { useAppContext } from 'AppContext'
import { getErrorMessage } from 'utils/Utils'
import { newCacheStrategy } from 'utils/CacheStrategy'
import { repoMetadataAtom } from 'atoms/repoMetadata'
import { useGetSpaceParam } from './useGetSpaceParam'

export function useGetRepositoryMetadata() {
  const space = useGetSpaceParam()
  const { routes } = useAppContext()
  const history = useHistory()
  const {
    repoName,
    gitRef,
    resourcePath = '',
    commitRef = '',
    pullRequestId = '',
    webhookId = '',
    commitSHA = '',
    ruleId = '',
    diffRefs,
    settingSectionMode = '',
    ...otherPathParams
  } = useParams<CODEProps>()
  const repoPath = useMemo(() => `${space}/${repoName}`, [space, repoName])
  const { data, error, loading, refetch, response } = useGet<RepoRepositoryOutput>({
    path: `/api/v1/repos/${repoPath}/+/`,
    lazy: true
  })
  const [repoMetadata, setRepoMetadata] = useAtom(repoMetadataAtom)
  const defaultBranch = repoMetadata?.default_branch || ''

  useEffect(() => {
    // Fetch repoMetadata when repoName exists and
    //  - cache does not exist yet
    //  - or cache is expired
    //  - or repoPath is changed
    if (repoName && (!repoMetadata || cacheStrategy.isExpired())) {
      refetch()
    } else if (repoMetadata && repoMetadata.path !== repoPath) {
      refetch().then(() => {
        if (repoPath?.toLocaleLowerCase() === repoMetadata.path?.toLocaleLowerCase()) {
          history.replace(routes.toCODERepository({ repoPath: repoMetadata.path as string }))
        }
      })
    }
  }, [repoName, refetch, repoMetadata, repoPath])

  useEffect(() => {
    if (data) {
      setRepoMetadata(data)
      cacheStrategy.update()
    }
  }, [data, setRepoMetadata])

  return {
    space,
    repoName,
    repoMetadata: repoName && repoMetadata?.path === repoPath ? repoMetadata : undefined,
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
    commitSHA,
    ruleId,
    settingSectionMode,
    ...otherPathParams
  }
}

const cacheStrategy = newCacheStrategy()
