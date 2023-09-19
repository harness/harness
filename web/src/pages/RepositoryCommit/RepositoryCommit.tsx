import React, { useMemo } from 'react'
import { Container, FlexExpander, Layout, PageBody } from '@harnessio/uicore'
import { useGet } from 'restful-react'
import { noop } from 'lodash-es'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useAppContext } from 'AppContext'
import type { TypesCommit } from 'services/code'
import { voidFn, getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { Changes } from 'components/Changes/Changes'
import CommitInfo from 'components/CommitInfo/CommitInfo'
import css from './RepositoryCommit.module.scss'

export default function RepositoryCommits() {
  const { repoMetadata, error, loading, commitRef, refetch } = useGetRepositoryMetadata()
  const { routes, standalone } = useAppContext()
  const { getString } = useStrings()

  const {
    data: commits,
    error: errorCommits,
    loading: loadingCommits
  } = useGet<{ commits: TypesCommit[] }>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit: LIST_FETCHING_LIMIT,
      git_ref: commitRef || repoMetadata?.default_branch
    },
    lazy: !repoMetadata
  })

  const ChangesTab = useMemo(() => {
    if (repoMetadata) {
      return (
        <Container className={css.changesContainer}>
          <Changes
            readOnly={true}
            repoMetadata={repoMetadata}
            commitSHA={commitRef}
            emptyTitle={getString('noChanges')}
            emptyMessage={getString('noChangesCompare')}
            onCommentUpdate={noop}
            prStatsChanged={0}
            scrollElement={(standalone ? document.querySelector(`.${css.main}`)?.parentElement || window : window) as HTMLElement}
          />
        </Container>
      )
    }
  }, [repoMetadata, commitRef, getString, standalone])

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
        {(repoMetadata && commitRef && !!commits?.commits?.length && (
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
      </PageBody>
    </Container>
  )
}
