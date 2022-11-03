import React from 'react'
import { Container, Pagination } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import type { RepoCommit, TypesRepository } from 'services/scm'
import { usePageIndex } from 'hooks/usePageIndex'
import { useGetPaginationInfo } from 'hooks/useGetPaginationInfo'
import { LIST_FETCHING_PER_PAGE } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import { BranchesContentHeader } from './BranchesContentHeader/BranchesContentHeader'
import { BranchesContent } from './BranchesContent/BranchesContent'
import css from './RepositoryBranchesContent.module.scss'

interface RepositoryBranchesContentProps {
  commitRef: string
  repoMetadata: TypesRepository
}

export function RepositoryBranchesContent({ repoMetadata, commitRef }: RepositoryBranchesContentProps) {
  const { routes } = useAppContext()
  const history = useHistory()
  const [pageIndex, setPageIndex] = usePageIndex()
  const { data: commits, response /*error, loading, refetch */ } = useGet<RepoCommit[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/commits?per_page=${LIST_FETCHING_PER_PAGE}&page=${
      pageIndex + 1
    }&git_ref=${commitRef || repoMetadata.defaultBranch}`
  })
  const { totalItems, totalPages, pageSize } = useGetPaginationInfo(response)

  return (
    <Container padding="xlarge" className={css.resourceContent}>
      <BranchesContentHeader
        repoMetadata={repoMetadata}
        onSwitch={gitRef => {
          setPageIndex(0)
          history.push(
            routes.toSCMRepositoryCommits({
              repoPath: repoMetadata.path as string,
              commitRef: gitRef
            })
          )
        }}
      />
      {!!commits?.length && (
        <>
          <BranchesContent commits={commits} repoMetadata={repoMetadata} />
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
