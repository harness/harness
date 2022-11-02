import React, { useMemo, useState } from 'react'
import { Container, Pagination } from '@harness/uicore'
import { useGet } from 'restful-react'
import type { RepoCommit, TypesRepository } from 'services/scm'
import { LIST_FETCHING_PER_PAGE, X_TOTAL, X_TOTAL_PAGES, X_PER_PAGE } from 'utils/Utils'
import { CommitsContentHeader } from './CommitsContentHeader/CommitsContentHeader'
import { CommitsContent } from './CommitsContent/CommitsContent'
import css from './RepositoryCommitsContent.module.scss'

interface RepositoryCommitsContentProps {
  commitRef: string
  repoMetadata: TypesRepository
}

export function RepositoryCommitsContent({ repoMetadata, commitRef }: RepositoryCommitsContentProps): JSX.Element {
  const [pageIndex, setPageIndex] = useState(0)
  const { data, response /*error, loading, refetch */ } = useGet<RepoCommit[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/commits?per_page=${LIST_FETCHING_PER_PAGE}&page=${pageIndex + 1}` // TODO: Pass commitRef
  })
  const itemCount = useMemo(() => parseInt(response?.headers?.get(X_TOTAL) || '0'), [response])
  const pageCount = useMemo(() => parseInt(response?.headers?.get(X_TOTAL_PAGES) || '0'), [response])
  const pageSize = useMemo(() => parseInt(response?.headers?.get(X_PER_PAGE) || '0'), [response])

  // TODO: Reset pageIndex when branch/tags is switched
  // useEffect(() => {
  //   setPageIndex(0)
  // }, [space])

  // TODO: Handle loading and error

  return (
    <Container padding="xlarge" className={css.resourceContent}>
      <CommitsContentHeader repoMetadata={repoMetadata} />
      {!!data?.length && (
        <>
          <CommitsContent contentInfo={data} repoMetadata={repoMetadata} />
          <Container margin={{ left: 'large', right: 'large' }}>
            <Pagination
              className={css.pagination}
              hidePageNumbers
              gotoPage={index => setPageIndex(index)}
              itemCount={itemCount}
              pageCount={pageCount}
              pageIndex={pageIndex}
              pageSize={pageSize}
            />
          </Container>
        </>
      )}
    </Container>
  )
}
