import React from 'react'
import { useGet } from 'restful-react'
import type { TypesCommit } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
import { voidFn, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { usePageIndex } from 'hooks/usePageIndex'
import { useStrings } from 'framework/strings'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { TabContentWrapper } from 'components/TabContentWrapper/TabContentWrapper'

interface CommitProps extends Pick<GitInfoProps, 'repoMetadata'> {
  sourceSha?: string
  targetSha?: string
  handleRefresh: () => void
}

export const CompareCommits: React.FC<CommitProps> = ({ repoMetadata, sourceSha, targetSha, handleRefresh }) => {
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
      git_ref: sourceSha,
      after: targetSha
    },
    lazy: !repoMetadata
  })

  return (
    <TabContentWrapper loading={loading} error={error} onRetry={voidFn(refetch)}>
      <CommitsView
        commits={data?.commits || []}
        repoMetadata={repoMetadata}
        emptyTitle={getString('compareEmptyDiffTitle')}
        emptyMessage={getString('compareEmptyDiffMessage')}
        handleRefresh={voidFn(handleRefresh)}
      />
      <ResourceListingPagination response={response} page={page} setPage={setPage} />
    </TabContentWrapper>
  )
}
