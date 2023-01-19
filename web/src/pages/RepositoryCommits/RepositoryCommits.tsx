import React from 'react'
import { Container, FlexExpander, Layout, PageBody } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useAppContext } from 'AppContext'
import { usePageIndex } from 'hooks/usePageIndex'
import type { RepoCommit } from 'services/code'
import { voidFn, getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { CommitsView } from 'components/CommitsView/CommitsView'
import css from './RepositoryCommits.module.scss'

export default function RepositoryCommits() {
  const { repoMetadata, error, loading, commitRef, refetch } = useGetRepositoryMetadata()
  const { routes } = useAppContext()
  const history = useHistory()
  const { getString } = useStrings()
  const [page, setPage] = usePageIndex()
  const {
    data: commits,
    response,
    error: errorCommits,
    loading: loadingCommits
  } = useGet<RepoCommit[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      git_ref: commitRef || repoMetadata?.default_branch
    },
    lazy: !repoMetadata
  })

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('commits')}
        dataTooltipId="repositoryCommits"
      />

      <PageBody error={getErrorMessage(error || errorCommits)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || loadingCommits} withBorder={!!commits && loadingCommits} />

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
                    setPage(1)
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

            <CommitsView
              commits={commits}
              repoMetadata={repoMetadata}
              emptyTitle={getString('noCommits')}
              emptyMessage={getString('noCommitsMessage')}
            />

            <ResourceListingPagination response={response} page={page} setPage={setPage} />
          </Container>
        )) ||
          null}
      </PageBody>
    </Container>
  )
}
