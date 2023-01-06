import React, { useState } from 'react'
import { Container, PageBody, NoDataCard, Tabs } from '@harness/uicore'
import { useHistory } from 'react-router-dom'
import { useGet } from 'restful-react'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage } from 'utils/Utils'
import emptyStateImage from 'images/empty-state.svg'
import { makeDiffRefs } from 'utils/GitUtils'
import { CommitsView } from 'components/CommitsView/CommitsView'
import { Changes } from 'components/Changes/Changes'
import type { RepoCommit } from 'services/code'
import { CompareContentHeader } from './CompareContentHeader/CompareContentHeader'
import css from './Compare.module.scss'

export default function Compare() {
  const { getString } = useStrings()
  const history = useHistory()
  const { routes } = useAppContext()
  const { repoMetadata, error, loading, diffRefs } = useGetRepositoryMetadata()
  const [sourceGitRef, setSourceGitRef] = useState(diffRefs.sourceGitRef)
  const [targetGitRef, setTargetGitRef] = useState(diffRefs.targetGitRef)
  const {
    data: commits,
    error: commitsError,
    loading: commitsLoading,
    refetch
  } = useGet<RepoCommit[]>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/commits`,
    queryParams: {
      git_ref: sourceGitRef
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
      <PageBody
        loading={loading || commitsLoading}
        error={getErrorMessage(error || commitsError)}
        retryOnError={() => refetch()}>
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

        {!targetGitRef ||
          (!sourceGitRef && (
            <Container className={css.noDataContainer}>
              <NoDataCard image={emptyStateImage} message={getString('selectToViewMore')} />
            </Container>
          ))}

        {!!repoMetadata && !!targetGitRef && !!sourceGitRef && (
          <Container className={css.tabsContainer}>
            <Tabs
              id="branchesTags"
              defaultSelectedTabId={'commits'}
              large={false}
              tabList={[
                {
                  id: 'commits',
                  title: getString('commits'),
                  panel: commits?.length ? (
                    <Container padding="xlarge">
                      <CommitsView commits={commits} repoMetadata={repoMetadata} />
                    </Container>
                  ) : (
                    <></>
                  )
                },
                {
                  id: 'diff',
                  title: getString('diff'),
                  panel: (
                    <Container>
                      <Changes
                        readOnly
                        repoMetadata={repoMetadata}
                        targetBranch={targetGitRef}
                        sourceBranch={sourceGitRef}
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
