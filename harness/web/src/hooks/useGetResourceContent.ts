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
    lazy: !repoMetadata?.path || lazy
  })

  return {
    data,
    error: repoMetadata?.is_empty ? undefined : error,
    loading,
    refetch,
    response,
    isRepositoryEmpty: !!repoMetadata?.is_empty
  }
}
