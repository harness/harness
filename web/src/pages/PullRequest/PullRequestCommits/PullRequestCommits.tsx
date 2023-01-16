import React from 'react'
import { useGet } from 'restful-react'
import type { RepoCommit } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
import { voidFn, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { usePageIndex } from 'hooks/usePageIndex'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { PrevNextPagination } from 'components/PrevNextPagination/PrevNextPagination'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'

export const PullRequestCommits: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = ({
  repoMetadata,
  pullRequestMetadata
}) => {
  const limit = LIST_FETCHING_LIMIT
  const [pageIndex, setPageIndex] = usePageIndex()
  const {
    data: commits,
    error,
    loading,
    refetch
  } = useGet<RepoCommit[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit,
      page: pageIndex + 1,
      git_ref: pullRequestMetadata.source_branch,
      after: pullRequestMetadata.target_branch
    },
    lazy: !repoMetadata
  })

  return (
    <PullRequestTabContentWrapper loading={loading} error={error} onRetry={voidFn(refetch)}>
      {!!commits?.length && <CommitsView commits={commits} repoMetadata={repoMetadata} />}

      <PrevNextPagination
        onPrev={pageIndex > 0 && (() => setPageIndex(pageIndex - 1))}
        onNext={commits?.length === limit && (() => setPageIndex(pageIndex + 1))}
      />
    </PullRequestTabContentWrapper>
  )
}
