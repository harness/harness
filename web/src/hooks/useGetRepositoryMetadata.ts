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
    commitSHA = '',
    ruleId = '',
    diffRefs,
    settingSectionMode = '',
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
    commitSHA,
    ruleId,
    settingSectionMode,
    ...otherPathParams
  }
}

// TODO: Repository metadata is rarely changed. It might be good to implement
// some caching strategy in here
