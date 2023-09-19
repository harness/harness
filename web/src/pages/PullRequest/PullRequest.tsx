import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Container, Layout, PageBody, Tabs, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { useGet, useMutate } from 'restful-react'
import { Render } from 'react-jsx-match'
import { useHistory } from 'react-router-dom'
import { compact, isEqual } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { voidFn, getErrorMessage, PullRequestSection, MergeCheckStatus } from 'utils/Utils'
import { CodeIcon } from 'utils/GitUtils'
import type { TypesPullReq, TypesPullReqStats, TypesRepository } from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { TabTitleWithCount, tabContainerCSS } from 'components/TabTitleWithCount/TabTitleWithCount'
import { usePRChecksDecision } from 'hooks/usePRChecksDecision'
import { ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
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
  const { routes, standalone } = useAppContext()
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
  const [showEditDescription, setShowEditDescription] = useState(false)

  const [stats, setStats] = useState<TypesPullReqStats>()
  // simple value one can listen on to react on stats changes (boolean is NOT enough)
  const [prStatsChanged, setPrStatsChanged] = useState(0)
  useMemo(() => {
    if (isEqual(stats, prData?.stats)) {
      return
    }

    setStats(stats)
    setPrStatsChanged(Date.now())
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
  const recheckPath = useMemo(
    () => `/api/v1/repos/${repoMetadata?.path}/+/pullreq/${pullRequestId}/recheck`,
    [repoMetadata?.path, pullRequestId]
  )
  const { mutate: recheckPR, loading: loadingRecheckPR } = useMutate({
    verb: 'POST',
    path: recheckPath
  })

  // prData holds the latest good PR data to make sure page is not broken
  // when polling fails
  useEffect(
    function setPrDataIfNotSet() {
      if (!pullRequestData || (prData && isEqual(prData, pullRequestData))) {
        return
      }

      // recheck pr (merge-check, ...) in case it's unavailable
      // Approximation of identifying target branch update:
      //   1. branch got updated before page was loaded (status is unchecked and prData is empty)
      //      NOTE: This doesn't guarantee the status is UNCHECKED due to target branch update and can cause duplicate
      //      PR merge checks being run on PR creation or source branch update.
      //   2. branch got updated while we are on the page (same source_sha but status changed to UNCHECKED)
      //      NOTE: This doesn't cover the case in which the status changed back to UNCHECKED before the PR is refetched.
      //      In that case, the user will have to re-open the PR - better than us spamming the backend with rechecks.
      // This is a TEMPORARY SOLUTION and will most likely change in the future (more so on backend side)
      if (
        pullRequestData.state == 'open' &&
        pullRequestData.merge_check_status == MergeCheckStatus.UNCHECKED &&
        // case 1:
        (!prData ||
          // case 2:
          (prData?.merge_check_status != MergeCheckStatus.UNCHECKED &&
            prData?.source_sha == pullRequestData.source_sha)) &&
        !loadingRecheckPR
      ) {
        // best effort attempt to recheck PR - fail silently
        recheckPR({})
      }

      setPrData(pullRequestData)
    },
    [pullRequestData]
  )

  useEffect(() => {
    let pollingInterval = 1000
    const fn = () => {
      if (repoMetadata) {
        refetchPullRequest().then(() => {
          pollingInterval = Math.min(pollingInterval + 1000, PR_MAX_POLLING_INTERVAL)
          interval = window.setTimeout(fn, pollingInterval)
        })
      }
    }
    let interval = window.setTimeout(fn, pollingInterval)

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
                        prStatsChanged={prStatsChanged}
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
                        prStatsChanged={prStatsChanged}
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
                          prStatsChanged={prStatsChanged}
                          scrollElement={
                            (standalone
                              ? document.querySelector(`.${css.main}`)?.parentElement || window
                              : window) as HTMLElement
                          }
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
                                <ExecutionStatus
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

const PR_MAX_POLLING_INTERVAL = 15000
