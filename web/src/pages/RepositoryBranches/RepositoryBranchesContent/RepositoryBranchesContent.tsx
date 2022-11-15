import React, { useState } from 'react'
import { Container, Pagination } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import type { RepoBranch } from 'services/scm'
import { usePageIndex } from 'hooks/usePageIndex'
import { useGetPaginationInfo } from 'hooks/useGetPaginationInfo'
import { LIST_FETCHING_PER_PAGE } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import type { GitInfoProps } from 'utils/GitUtils'
import { BranchesContentHeader } from './BranchesContentHeader/BranchesContentHeader'
import { BranchesContent } from './BranchesContent/BranchesContent'
import css from './RepositoryBranchesContent.module.scss'

export function RepositoryBranchesContent({ repoMetadata }: Pick<GitInfoProps, 'repoMetadata'>) {
  const { routes } = useAppContext()
  const history = useHistory()
  const [searchTerm, setSearchTerm] = useState('')
  const [pageIndex, setPageIndex] = usePageIndex()
  const { data: branches, response /*error, loading, refetch */ } = useGet<RepoBranch[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/branches`,
    queryParams: {
      per_page: LIST_FETCHING_PER_PAGE,
      page: pageIndex + 1,
      direction: 'desc',
      include_commit: true,
      query: searchTerm
    }
  })
  const { totalItems, totalPages, pageSize } = useGetPaginationInfo(response)

  return (
    <Container padding="xlarge" className={css.resourceContent}>
      <BranchesContentHeader
        repoMetadata={repoMetadata}
        onBranchTypeSwitched={gitRef => {
          setPageIndex(0)
          history.push(
            routes.toSCMRepositoryCommits({
              repoPath: repoMetadata.path as string,
              commitRef: gitRef
            })
          )
        }}
        onSearchTermChanged={value => {
          setSearchTerm(value)
        }}
      />
      {!!branches?.length && (
        <>
          <BranchesContent branches={branches} repoMetadata={repoMetadata} searchTerm={searchTerm} />
          <Container margin={{ left: 'large', right: 'large' }}>
            <Pagination
              className={css.pagination}
              hidePageNumbers
              gotoPage={index => setPageIndex(index)}
              itemCount={totalItems}
              pageCount={totalPages}
              pageIndex={pageIndex}
              pageSize={pageSize}
            />
          </Container>
        </>
      )}
    </Container>
  )
}
