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

import React from 'react'
import { useGet } from 'restful-react'
import type { TypesCommit } from 'services/code'
import { normalizeGitRef, type GitInfoProps } from 'utils/GitUtils'
import { voidFn, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { usePageIndex } from 'hooks/usePageIndex'
import { useStrings } from 'framework/strings'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'

export const PullRequestCommits: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = ({
  repoMetadata,
  pullRequestMetadata
}) => {
  const limit = LIST_FETCHING_LIMIT
  const [page, setPage] = usePageIndex()
  const { getString } = useStrings()
  const { data, error, loading, refetch, response } = useGet<{
    commits: TypesCommit[]
  }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit,
      page,
      git_ref: normalizeGitRef(pullRequestMetadata.source_sha),
      after: normalizeGitRef(pullRequestMetadata.merge_base_sha)
    },
    lazy: !repoMetadata
  })

  return (
    <PullRequestTabContentWrapper loading={loading} error={error} onRetry={voidFn(refetch)}>
      <CommitsView
        commits={data?.commits || []}
        repoMetadata={repoMetadata}
        emptyTitle={getString('noCommits')}
        emptyMessage={getString('noCommitsPR')}
        pullRequestMetadata={pullRequestMetadata}
      />

      <ResourceListingPagination response={response} page={page} setPage={setPage} />
    </PullRequestTabContentWrapper>
  )
}
