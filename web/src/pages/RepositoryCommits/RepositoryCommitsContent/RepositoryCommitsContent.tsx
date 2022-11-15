import React from 'react'
import { Container, Pagination } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import type { RepoCommit } from 'services/scm'
import { usePageIndex } from 'hooks/usePageIndex'
import { useGetPaginationInfo } from 'hooks/useGetPaginationInfo'
import { LIST_FETCHING_PER_PAGE } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import type { GitInfoProps } from 'utils/GitUtils'
import { CommitsContentHeader } from './CommitsContentHeader/CommitsContentHeader'
import { CommitsContent } from './CommitsContent/CommitsContent'
import css from './RepositoryCommitsContent.module.scss'

export function RepositoryCommitsContent({
  repoMetadata,
  commitRef
}: Pick<GitInfoProps, 'repoMetadata' | 'commitRef'>) {
  const { routes } = useAppContext()
  const history = useHistory()
  const [pageIndex, setPageIndex] = usePageIndex()
  const { data: commits, response /*error, loading, refetch */ } = useGet<RepoCommit[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/commits`,
    queryParams: {
      per_page: LIST_FETCHING_PER_PAGE,
      page: pageIndex + 1,
      git_ref: commitRef || repoMetadata.defaultBranch
    }
  })
  const { totalItems, totalPages, pageSize } = useGetPaginationInfo(response)

  return (
    <Container padding="xlarge" className={css.resourceContent}>
      <CommitsContentHeader
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
          <CommitsContent commits={commits} repoMetadata={repoMetadata} />
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
