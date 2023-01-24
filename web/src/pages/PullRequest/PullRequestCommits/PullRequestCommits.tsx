import React from 'react'
import { useGet } from 'restful-react'
import type { TypesCommit } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
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
  const {
    data: commits,
    error,
    loading,
    refetch,
    response
  } = useGet<TypesCommit[]>({
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
      <CommitsView
        commits={commits || []}
        repoMetadata={repoMetadata}
        emptyTitle={getString('noCommits')}
        emptyMessage={getString('noCommitsPR')}
      />

      <ResourceListingPagination response={response} page={page} setPage={setPage} />
    </PullRequestTabContentWrapper>
  )
}
