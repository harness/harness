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

interface CommitProps extends Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'> {
  prStatsChanged: Number
  handleRefresh: () => void
}

export const PullRequestCommits: React.FC<CommitProps> = ({
  repoMetadata,
  pullRequestMetadata,
  prStatsChanged,
  handleRefresh
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
      git_ref: pullRequestMetadata.source_sha,
      after: pullRequestMetadata.merge_base_sha
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
        prStatsChanged={prStatsChanged}
        handleRefresh={voidFn(handleRefresh)}
        pullRequestMetadata={pullRequestMetadata}
      />

      <ResourceListingPagination response={response} page={page} setPage={setPage} />
    </PullRequestTabContentWrapper>
  )
}
