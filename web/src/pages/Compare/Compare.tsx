import { noop } from 'lodash-es'
import React, { useState } from 'react'
import { Container, PageBody, NoDataCard, Tabs } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { voidFn, getErrorMessage, LIST_FETCHING_LIMIT } from 'utils/Utils'
import { Images } from 'images'
import { makeDiffRefs } from 'utils/GitUtils'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { Changes } from 'components/Changes/Changes'
import type { TypesCommit } from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { usePageIndex } from 'hooks/usePageIndex'
import { ResourceListingPagination } from 'components/ResourceListingPagination/ResourceListingPagination'
import { CompareContentHeader } from './CompareContentHeader/CompareContentHeader'
import css from './Compare.module.scss'

export default function Compare() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const { repoMetadata, error, loading, diffRefs } = useGetRepositoryMetadata()
  const [sourceGitRef, setSourceGitRef] = useState(diffRefs.sourceGitRef)
  const [targetGitRef, setTargetGitRef] = useState(diffRefs.targetGitRef)
  const [page, setPage] = usePageIndex()
  const limit = LIST_FETCHING_LIMIT
  const {
    data: commits,
    error: commitsError,
    refetch,
    response
  } = useGet<TypesCommit[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      limit,
      page,
      git_ref: sourceGitRef,
      after: targetGitRef
    },
    lazy: !repoMetadata
  })

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={getString('comparingChanges')}
        dataTooltipId="comparingChanges"
      />
      <PageBody error={getErrorMessage(error || commitsError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={loading} />

        {repoMetadata && (
          <CompareContentHeader
            repoMetadata={repoMetadata}
            targetGitRef={targetGitRef}
            onTargetGitRefChanged={gitRef => {
              setTargetGitRef(gitRef)
              history.replace(
                routes.toCODECompare({
                  repoPath: repoMetadata.path as string,
                  diffRefs: makeDiffRefs(gitRef, sourceGitRef)
                })
              )
            }}
            sourceGitRef={sourceGitRef}
            onSourceGitRefChanged={gitRef => {
              setSourceGitRef(gitRef)
              history.replace(
                routes.toCODECompare({
                  repoPath: repoMetadata.path as string,
                  diffRefs: makeDiffRefs(targetGitRef, gitRef)
                })
              )
            }}
          />
        )}

        {(!targetGitRef || !sourceGitRef) && (
          <Container className={css.noDataContainer}>
            <NoDataCard image={Images.EmptyState} message={getString('selectToViewMore')} />
          </Container>
        )}

        {!!repoMetadata && !!targetGitRef && !!sourceGitRef && (
          <Container className={css.tabsContainer}>
            <Tabs
              id="branchesTags"
              defaultSelectedTabId="diff"
              large={false}
              onChange={() => setPage(1)}
              tabList={[
                {
                  id: 'commits',
                  title: getString('commits'),
                  panel: (
                    <Container padding="xlarge">
                      <CommitsView
                        commits={commits || []}
                        repoMetadata={repoMetadata}
                        emptyTitle={getString('compareEmptyDiffTitle')}
                        emptyMessage={getString('compareEmptyDiffMessage')}
                      />
                      <ResourceListingPagination response={response} page={page} setPage={setPage} />
                    </Container>
                  )
                },
                {
                  id: 'diff',
                  title: getString('filesChanged'),
                  panel: (
                    <Container className={css.changesContainer}>
                      <Changes
                        readOnly
                        repoMetadata={repoMetadata}
                        targetBranch={targetGitRef}
                        sourceBranch={sourceGitRef}
                        emptyTitle={getString('noChanges')}
                        emptyMessage={getString('noChangesCompare')}
                        onCommentUpdate={noop} // TODO: Update tab stats
                      />
                    </Container>
                  )
                }
              ]}
            />
          </Container>
        )}
      </PageBody>
    </Container>
  )
}
