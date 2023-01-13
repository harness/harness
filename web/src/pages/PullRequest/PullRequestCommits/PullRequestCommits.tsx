import React from 'react'
import { useGet } from 'restful-react'
import type { RepoCommit } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { PullRequestTabContentWrapper } from '../PullRequestTabContentWrapper'

export const PullRequestCommits: React.FC<Pick<GitInfoProps, 'repoMetadata' | 'pullRequestMetadata'>> = ({
  repoMetadata,
  pullRequestMetadata
}) => {
  const {
    data: commits,
    error,
    loading,
    refetch
  } = useGet<RepoCommit[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      git_ref: pullRequestMetadata.source_branch,
      after: pullRequestMetadata.target_branch
    },
    lazy: !repoMetadata
  })

  return (
    <PullRequestTabContentWrapper loading={loading} error={error} onRetry={() => refetch()}>
      {!!commits?.length && <CommitsView commits={commits} repoMetadata={repoMetadata} />}
    </PullRequestTabContentWrapper>
  )
}
