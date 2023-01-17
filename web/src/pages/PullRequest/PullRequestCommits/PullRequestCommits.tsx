import React from 'react'
import { useGet } from 'restful-react'
import type { RepoCommit } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
import { voidFn, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { usePageIndex } from 'hooks/usePageIndex'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'

export const PullRequestCommits: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = ({
  repoMetadata,
  pullRequestMetadata
}) => {
  const limit = LIST_FETCHING_LIMIT
  const [page, setPage] = usePageIndex()
  const {
    data: commits,
    error,
    loading,
    refetch,
    response
  } = useGet<RepoCommit[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit,
      page,
      git_ref: pullRequestMetadata.source_branch,
      after: pullRequestMetadata.target_branch
    },
    lazy: !repoMetadata
  })

  return (
    <PullRequestTabContentWrapper loading={loading} error={error} onRetry={voidFn(refetch)}>
      {!!commits?.length && <CommitsView commits={commits} repoMetadata={repoMetadata} />}

      <ResourceListingPagination response={response} page={page} setPage={setPage} />
    </PullRequestTabContentWrapper>
  )
}
