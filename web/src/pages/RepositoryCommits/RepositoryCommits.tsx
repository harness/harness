import React from 'react'
import { Container, FlexExpander, Layout, PageBody, Pagination } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useAppContext } from 'AppContext'
import { usePageIndex } from 'hooks/usePageIndex'
import type { RepoCommit } from 'services/code'
import { getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useGetPaginationInfo } from 'hooks/useGetPaginationInfo'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { CommitsView } from '../../components/CommitsView/CommitsView'
import css from './RepositoryCommits.module.scss'

export default function RepositoryCommits() {
  const { repoMetadata, error, loading, commitRef, refetch } = useGetRepositoryMetadata()
  const { routes } = useAppContext()
  const history = useHistory()
  const { getString } = useStrings()
  const [pageIndex, setPageIndex] = usePageIndex()
  const {
    data: commits,
    response,
    error: errorCommits,
    loading: loadingCommits
  } = useGet<RepoCommit[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page: pageIndex + 1,
      git_ref: commitRef || repoMetadata?.default_branch
    },
    lazy: !repoMetadata
  })
  const { totalItems, totalPages, pageSize } = useGetPaginationInfo(response)

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('commits')}
        dataTooltipId="repositoryCommits"
      />

      <PageBody
        loading={loading || loadingCommits}
        error={getErrorMessage(error || errorCommits)}
        retryOnError={() => refetch()}>
        {(repoMetadata && !!commits?.length && (
          <Container padding="xlarge" className={css.resourceContent}>
            <Container className={css.contentHeader}>
              <Layout.Horizontal spacing="medium">
                <BranchTagSelect
                  repoMetadata={repoMetadata}
                  disableBranchCreation
                  disableViewAllBranches
                  gitRef={commitRef || (repoMetadata.default_branch as string)}
                  onSelect={ref => {
                    setPageIndex(0)
                    history.push(
                      routes.toCODECommits({
                        repoPath: repoMetadata.path as string,
                        commitRef: ref
                      })
                    )
                  }}
                />
                <FlexExpander />
              </Layout.Horizontal>
            </Container>

            <CommitsView commits={commits} repoMetadata={repoMetadata} />

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
          </Container>
        )) ||
          null}
      </PageBody>
    </Container>
  )
}
