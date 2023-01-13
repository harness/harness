import React, { useState } from 'react'
import { Container } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import type { RepoBranch } from 'services/code'
import { usePageIndex } from 'hooks/usePageIndex'
import { useGetPaginationInfo } from 'hooks/useGetPaginationInfo'
import { LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import type { GitInfoProps } from 'utils/GitUtils'
import { PrevNextPagination } from 'components/PrevNextPagination/PrevNextPagination'
import { BranchesContentHeader } from './BranchesContentHeader/BranchesContentHeader'
import { BranchesContent } from './BranchesContent/BranchesContent'
import css from './RepositoryBranchesContent.module.scss'

export function RepositoryBranchesContent({ repoMetadata }: Pick<GitInfoProps, 'repoMetadata'>) {
  const { routes } = useAppContext()
  const history = useHistory()
  const [searchTerm, setSearchTerm] = useState('')
  const [pageIndex, setPageIndex] = usePageIndex()
  const {
    data: branches,
    response /*error, loading,*/,
    refetch
  } = useGet<RepoBranch[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/branches`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page: pageIndex + 1,
      sort: 'date',
      order: 'desc',
      include_commit: true,
      query: searchTerm
    }
  })
  const { X_NEXT_PAGE, X_PREV_PAGE } = useGetPaginationInfo(response)

  return (
    <Container padding="xlarge" className={css.resourceContent}>
      <BranchesContentHeader
        repoMetadata={repoMetadata}
        onBranchTypeSwitched={gitRef => {
          setPageIndex(0)
          history.push(
            routes.toCODECommits({
              repoPath: repoMetadata.path as string,
              commitRef: gitRef
            })
          )
        }}
        onSearchTermChanged={value => {
          setSearchTerm(value)
          setPageIndex(0)
        }}
        onNewBranchCreated={refetch}
      />

      {!!branches?.length && (
        <BranchesContent
          branches={branches}
          repoMetadata={repoMetadata}
          searchTerm={searchTerm}
          onDeleteSuccess={refetch}
        />
      )}

      <PrevNextPagination
        onPrev={!!X_PREV_PAGE && (() => setPageIndex(pageIndex - 1))}
        onNext={!!X_NEXT_PAGE && (() => setPageIndex(pageIndex + 1))}
      />
    </Container>
  )
}

// TODO: Handle loading and error
