import React, { useEffect, useState } from 'react'
import { Container } from '@harness/uicore'
import { useGet } from 'restful-react'
import { useHistory } from 'react-router-dom'
import type { RepoBranch } from 'services/code'
import { usePageIndex } from 'hooks/usePageIndex'
import { LIST_FETCHING_LIMIT, PageBrowserProps } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import type { GitInfoProps } from 'utils/GitUtils'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { useShowRequestError } from 'hooks/useShowRequestError'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import { NoResultCard } from 'components/NoResultCard/NoResultCard'
import { BranchesContentHeader } from './BranchesContentHeader/BranchesContentHeader'
import { BranchesContent } from './BranchesContent/BranchesContent'
import css from './RepositoryBranchesContent.module.scss'

export function RepositoryBranchesContent({ repoMetadata }: Pick<GitInfoProps, 'repoMetadata'>) {
  const { routes } = useAppContext()
  const history = useHistory()
  const [searchTerm, setSearchTerm] = useState('')
  const { updateQueryParams } = useUpdateQueryParams()

  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)
  const {
    data: branches,
    response,
    error,
    loading,
    refetch
  } = useGet<RepoBranch[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/branches`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      sort: 'date',
      order: 'desc',
      include_commit: true,
      query: searchTerm
    }
  })

  useEffect(() => {
    if (page > 1) {
      updateQueryParams({ page: page.toString() })
    }
  }, [setPage]) // eslint-disable-line react-hooks/exhaustive-deps

  useShowRequestError(error)

  return (
    <Container padding="xlarge" className={css.resourceContent}>
      <BranchesContentHeader
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
        <BranchesContent
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
