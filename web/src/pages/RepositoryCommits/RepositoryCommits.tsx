import React, { useEffect, useMemo } from 'react'
import { Container, FlexExpander, Layout, PageBody } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { noop } from 'lodash-es'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useAppContext } from 'AppContext'
import { usePageIndex } from 'hooks/usePageIndex'
import { useQueryParams } from 'hooks/useQueryParams'
import { useUpdateQueryParams } from 'hooks/useUpdateQueryParams'
import type { TypesCommit } from 'services/code'
import { voidFn, getErrorMessage, LIST_FETCHING_LIMIT, PageBrowserProps } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { BranchTagSelect } from 'components/BranchTagSelect/BranchTagSelect'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { Changes } from 'components/Changes/Changes'
import CommitInfo from 'components/CommitInfo/CommitInfo'
import css from './RepositoryCommits.module.scss'

export default function RepositoryCommits() {
  const { repoMetadata, error, loading, commitRef, refetch } = useGetRepositoryMetadata()
  const { routes } = useAppContext()
  const history = useHistory()
  const { getString } = useStrings()
  const { updateQueryParams } = useUpdateQueryParams()

  const pageBrowser = useQueryParams<PageBrowserProps>()
  const pageInit = pageBrowser.page ? parseInt(pageBrowser.page) : 1
  const [page, setPage] = usePageIndex(pageInit)

  const {
    data: commits,
    response,
    error: errorCommits,
    loading: loadingCommits
  } = useGet<{ commits: TypesCommit[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      page,
      git_ref: commitRef || repoMetadata?.default_branch
    },
    lazy: !repoMetadata
  })

  useEffect(() => {
    if (pageBrowser.page) {
      updateQueryParams({ page: page.toString() })
    }
  }, [setPage]) // eslint-disable-line react-hooks/exhaustive-deps

  const ChangesTab = useMemo(() => {
    if (repoMetadata) {
      return (
        <Container className={css.changesContainer}>
          <Changes
            readOnly
            repoMetadata={repoMetadata}
            targetBranch={`${commitRef}~1`}
            sourceBranch={commitRef}
            emptyTitle={getString('noChanges')}
            emptyMessage={getString('noChangesCompare')}
            onCommentUpdate={noop}
          />
        </Container>
      )
    }
  }, [repoMetadata, commitRef, getString, response])

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('commits')}
        dataTooltipId="repositoryCommits"
        extraBreadcrumbLinks={
          commitRef && repoMetadata
            ? [
                {
                  label: getString('commits'),
                  url: routes.toCODECommits({ repoPath: repoMetadata.path as string, commitRef: '' })
                }
              ]
            : undefined
        }
      />

      <PageBody error={getErrorMessage(error || errorCommits)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading || loadingCommits} withBorder={!!commits && loadingCommits} />
        {(repoMetadata && commitRef && !pageBrowser.page && !!commits?.commits?.length && (
          <Container padding="xlarge" className={css.resourceContent}>
            <Container className={css.contentHeader}>
              <Layout.Horizontal>
                <CommitInfo repoMetadata={repoMetadata} commitRef={commitRef} />
                <FlexExpander />
              </Layout.Horizontal>
            </Container>
            {ChangesTab}
          </Container>
        )) ||
          null}
        {(repoMetadata && (!commitRef || pageBrowser.page) && !!commits?.commits?.length && (
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
                        commitRef: `${ref}?page=1`
                      })
                    )
                  }}
                />
                <FlexExpander />
              </Layout.Horizontal>
            </Container>

            <CommitsView
              commits={commits?.commits}
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
