import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Container, Layout, PageBody, Tabs, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useGet } from 'restful-react'
import { Render } from 'react-jsx-match'
import { useHistory } from 'react-router-dom'
import { compact } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { voidFn, getErrorMessage, PullRequestSection } from 'utils/Utils'
import { CodeIcon } from 'utils/GitUtils'
import type { TypesPullReq, TypesPullReqStats, TypesRepository } from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { TabTitleWithCount, tabContainerCSS } from 'components/TabTitleWithCount/TabTitleWithCount'
import { usePRChecksDecision } from 'hooks/usePRChecksDecision'
import { PRCheckExecutionStatus } from 'components/PRCheckExecutionStatus/PRCheckExecutionStatus'
import { PullRequestMetaLine } from './PullRequestMetaLine'
import { Conversation } from './Conversation/Conversation'
import { Checks } from './Checks/Checks'
import { Changes } from '../../components/Changes/Changes'
import { PullRequestCommits } from './PullRequestCommits/PullRequestCommits'
import { PullRequestTitle } from './PullRequestTitle'
import css from './PullRequest.module.scss'

export default function PullRequest() {
  const history = useHistory()
  const { getString } = useStrings()
  const { routes } = useAppContext()
  const {
    repoMetadata,
    error,
    loading,
    refetch,
    pullRequestId,
    pullRequestSection = PullRequestSection.CONVERSATION,
    commitSHA
  } = useGetRepositoryMetadata()
  const path = useMemo(
    () => `/api/v1/repos/${repoMetadata?.path}/+/pullreq/${pullRequestId}`,
    [repoMetadata?.path, pullRequestId]
  )
  const {
    data: pullRequestData,
    error: prError,
    loading: prLoading,
    refetch: refetchPullRequest
  } = useGet<TypesPullReq>({
    path,
    lazy: !repoMetadata
  })
  const [prData, setPrData] = useState<TypesPullReq>()
  const prChecksDecisionResult = usePRChecksDecision({
    repoMetadata,
    pullRequestMetadata: prData
  })
  const showSpinner = useMemo(() => {
    return loading || (prLoading && !prData)
  }, [loading, prLoading, prData])
  const [stats, setStats] = useState<TypesPullReqStats>()
  const [showEditDescription, setShowEditDescription] = useState(false)
  const prHasChanged = useMemo(() => {
    if (stats && prData?.stats) {
      if (
        stats.commits !== prData.stats.commits ||
        stats.conversations !== prData.stats.conversations ||
        stats.files_changed !== prData.stats.files_changed
      ) {
        window.setTimeout(() => setStats(prData.stats), 50)
        return true
      }
    }
    return false
  }, [prData?.stats, stats])
  const onAddDescriptionClick = useCallback(() => {
    setShowEditDescription(true)
    history.replace(
      routes.toCODEPullRequest({
        repoPath: repoMetadata?.path as string,
        pullRequestId,
        pullRequestSection: PullRequestSection.CONVERSATION
      })
    )
  }, [history, routes, repoMetadata?.path, pullRequestId])

  useEffect(
    function setStatsIfNotSet() {
      if (!stats && prData?.stats) {
        setStats(prData.stats)
      }
    },
    [prData?.stats, stats]
  )

  // prData holds the latest good PR data to make sure page is not broken
  // when polling fails
  useEffect(
    function setPrDataIfNotSet() {
      if (pullRequestData) {
        setPrData(pullRequestData)
      }
    },
    [pullRequestData]
  )

  useEffect(() => {
    const fn = () => {
      if (repoMetadata) {
        refetchPullRequest().then(() => {
          interval = window.setTimeout(fn, PR_POLLING_INTERVAL)
        })
      }
    }
    let interval = window.setTimeout(fn, PR_POLLING_INTERVAL)

    return () => window.clearTimeout(interval)
  }, [repoMetadata, refetchPullRequest, path])

  const activeTab = useMemo(
    () =>
      Object.values(PullRequestSection).find(value => value === pullRequestSection)
        ? pullRequestSection
        : PullRequestSection.CONVERSATION,
    [pullRequestSection]
  )

  return (
    <Container className={css.main}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={
          repoMetadata && prData ? (
            <PullRequestTitle repoMetadata={repoMetadata} {...prData} onAddDescriptionClick={onAddDescriptionClick} />
          ) : (
            ''
          )
        }
        dataTooltipId="repositoryPullRequests"
        extraBreadcrumbLinks={
          repoMetadata && [
            {
              label: getString('pullRequests'),
              url: routes.toCODEPullRequests({ repoPath: repoMetadata.path as string })
            }
          ]
        }
      />
      <PageBody error={!prData && getErrorMessage(error || prError)} retryOnError={voidFn(refetch)}>
        <LoadingSpinner visible={showSpinner} />

        <Render when={repoMetadata && prData}>
          <>
            <PullRequestMetaLine repoMetadata={repoMetadata as TypesRepository} {...prData} />
            <Container className={tabContainerCSS.tabsContainer}>
              <Tabs
                id="prTabs"
                defaultSelectedTabId={activeTab}
                selectedTabId={activeTab}
                large={false}
                onChange={tabId => {
                  history.replace(
                    routes.toCODEPullRequest({
                      repoPath: repoMetadata?.path as string,
                      pullRequestId,
                      pullRequestSection: tabId !== PullRequestSection.CONVERSATION ? (tabId as string) : undefined
                    })
                  )
                }}
                tabList={[
                  {
                    id: PullRequestSection.CONVERSATION,
                    title: (
                      <TabTitleWithCount
                        icon={CodeIcon.Chat}
                        title={getString('conversation')}
                        count={prData?.stats?.conversations || 0}
                      />
                    ),
                    panel: (
                      <Conversation
                        repoMetadata={repoMetadata as TypesRepository}
                        pullRequestMetadata={prData as TypesPullReq}
                        prChecksDecisionResult={prChecksDecisionResult}
                        onCommentUpdate={() => {
                          setShowEditDescription(false)
                          refetchPullRequest()
                        }}
                        prHasChanged={prHasChanged}
                        showEditDescription={showEditDescription}
                        onCancelEditDescription={() => setShowEditDescription(false)}
                      />
                    )
                  },
                  {
                    id: PullRequestSection.COMMITS,
                    title: (
                      <TabTitleWithCount
                        icon={CodeIcon.Commit}
                        title={getString('commits')}
                        count={prData?.stats?.commits || 0}
                        padding={{ left: 'medium' }}
                      />
                    ),
                    panel: (
                      <PullRequestCommits
                        repoMetadata={repoMetadata as TypesRepository}
                        pullRequestMetadata={prData as TypesPullReq}
                        prHasChanged={prHasChanged}
                        handleRefresh={voidFn(refetchPullRequest)}
                      />
                    )
                  },
                  {
                    id: PullRequestSection.FILES_CHANGED,
                    title: (
                      <TabTitleWithCount
                        icon={CodeIcon.File}
                        title={getString('filesChanged')}
                        count={prData?.stats?.files_changed || 0}
                        padding={{ left: 'medium' }}
                      />
                    ),
                    panel: (
                      <Container className={css.changes}>
                        <Changes
                          repoMetadata={repoMetadata as TypesRepository}
                          pullRequestMetadata={prData as TypesPullReq}
                          defaultCommitRange={compact(commitSHA?.split(/~1\.\.\.|\.\.\./g))}
                          targetRef={prData?.merge_base_sha}
                          sourceRef={prData?.source_sha}
                          emptyTitle={getString('noChanges')}
                          emptyMessage={getString('noChangesPR')}
                          onCommentUpdate={voidFn(refetchPullRequest)}
                          prHasChanged={prHasChanged}
                        />
                      </Container>
                    )
                  },
                  {
                    id: PullRequestSection.CHECKS,
                    title: (
                      <TabTitleWithCount
                        icon="main-search"
                        iconSize={14}
                        title={getString('checks')}
                        countElement={
                          prChecksDecisionResult?.overallStatus ? (
                            <Container className={css.checksCount}>
                              <Layout.Horizontal className={css.checksCountLayout}>
                                <PRCheckExecutionStatus
                                  status={prChecksDecisionResult?.overallStatus}
                                  noBackground
                                  iconOnly
                                  iconSize={15}
                                />

                                <Text
                                  color={prChecksDecisionResult?.color}
                                  padding={{ left: 'xsmall' }}
                                  tag="span"
                                  font={{ variation: FontVariation.FORM_MESSAGE_WARNING }}>
                                  {prChecksDecisionResult?.count[prChecksDecisionResult?.overallStatus]}
                                </Text>
                              </Layout.Horizontal>
                            </Container>
                          ) : null
                        }
                        count={prChecksDecisionResult?.count?.failure || 0}
                        padding={{ left: 'medium' }}
                      />
                    ),
                    panel: (
                      <Checks
                        repoMetadata={repoMetadata as TypesRepository}
                        pullRequestMetadata={prData as TypesPullReq}
                        prChecksDecisionResult={prChecksDecisionResult}
                      />
                    )
                  }
                ]}
              />
            </Container>
          </>
        </Render>
      </PageBody>
    </Container>
  )
}

const PR_POLLING_INTERVAL = 10000
