import React, { useState } from 'react'
import { Container } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import type { RepoBranch, RepoCommitTag } from 'services/code'
import { usePageIndex } from 'hooks/usePageIndex'
import { LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import type { GitInfoProps } from 'utils/GitUtils'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { BranchesContent } from '../../RepositoryBranches/RepositoryBranchesContent/BranchesContent/BranchesContent'
import css from './RepositoryTagsContent.module.scss'
import { RepositoryTagsContentHeader } from '../RepositoryTagsContentHeader/RepositoryTagsContentHeader'
import { TagsContent } from '../TagsContent/TagsContent'

export function RepositoryTagsContent({ repoMetadata }: Pick<GitInfoProps, 'repoMetadata'>) {
  const { routes } = useAppContext()
  const history = useHistory()
  const [searchTerm, setSearchTerm] = useState('')
  const [page, setPage] = usePageIndex()
  const {
    data: branches,
    response,
    error,
    loading,
    refetch
  } = useGet<RepoCommitTag[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/tags`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      sort: 'date',
      order: 'desc',
      include_commit: true,
      query: searchTerm
    }
  })

  useShowRequestError(error)

  return (
    <Container padding="xlarge" className={css.resourceContent}>
      <RepositoryTagsContentHeader
        loading={loading}
        repoMetadata={repoMetadata}
        onBranchTypeSwitched={gitRef => {
          setPage(1)
          history.push(
            routes.toCODECommits({
              repoPath: repoMetadata.path as string,
              commitRef: gitRef
            })
          )
        }}
        onSearchTermChanged={value => {
          setSearchTerm(value)
          setPage(1)
        }}
        onNewBranchCreated={refetch}
      />

      {!!branches?.length && (
        <TagsContent
          branches={branches}
          repoMetadata={repoMetadata}
          searchTerm={searchTerm}
          onDeleteSuccess={refetch}
        />
      )}

      <NoResultCard showWhen={() => !!branches && branches.length === 0 && !!searchTerm?.length} forSearch={true} />

      <ResourceListingPagination response={response} page={page} setPage={setPage} />
    </Container>
  )
}
