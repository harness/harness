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
import { useAtomValue } from 'jotai'
import type { TypesBranchExtended } from 'services/code'
import { LIST_FETCHING_LIMIT } from 'utils/Utils'
import { repoMetadataAtom } from 'atoms/repoMetadata'

export function useFindGitBranch(branchName?: string, includeCommit = false) {
  const repoMetadata = useAtomValue(repoMetadataAtom)
  const { data } = useGet<TypesBranchExtended[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/branches`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      include_commit: includeCommit,
      query: branchName
    },
    lazy: !repoMetadata || !branchName
  })

  return data?.find(branch => branch.name === branchName)
}
