/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { STOAppCustomProps } from '@harness/microfrontends'
import { FontVariation } from '@harnessio/design-system'
import { Container, Layout, PageBody, PageSpinner, Tabs, Text } from '@harnessio/uicore'
import { useAppContext } from 'AppContext'
import { ExecutionState, ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { tabContainerCSS, TabTitleWithCount } from 'components/TabTitleWithCount/TabTitleWithCount'
import ChildAppMounter from 'framework/microfrontends/ChildAppMounter'
import { useStrings } from 'framework/strings'
import { useScrollTop } from 'hooks/useScrollTop'
import { useSetPageContainerWidthVar } from 'hooks/useSetPageContainerWidthVar'
import { compact } from 'lodash-es'
import React, { lazy, useCallback, useMemo, useRef, useState } from 'react'
import { Render } from 'react-jsx-match'
import { useHistory, useParams } from 'react-router-dom'
import type { TypesPullReq, TypesRepository } from 'services/code'
import { useFrontendExecutionForRepo, useFrontendExecutionIssueCounts } from 'services/sto/stoComponents'
import { CodeIcon } from 'utils/GitUtils'
import type { Identifier } from 'utils/types'
import { getErrorMessage, PullRequestSection } from 'utils/Utils'
import { Changes } from '../../components/Changes/Changes'
import { Checks } from './Checks/Checks'
import { Conversation } from './Conversation/Conversation'
import css from './PullRequest.module.scss'
import { PullRequestCommits } from './PullRequestCommits/PullRequestCommits'
import { PullRequestMetaLine } from './PullRequestMetaLine'
import { PullRequestTitle } from './PullRequestTitle'
import { useGetPullRequestInfo } from './useGetPullRequestInfo'

// @ts-ignore
const RemoteSTOApp = lazy(() => import(`stoV2/App`))
// @ts-ignore
const RemotePipelineSecurityView = lazy(() => import(`stoV2/PipelineSecurityView`))

export default function PullRequest() {
  const history = useHistory()
  const { accountId, orgIdentifier = '', projectIdentifier = '' } = useParams<Identifier>()
  const { getString } = useStrings()
  const { routes, standalone, routingId } = useAppContext()
  const { parentContextObj } = useAppContext()
  const {
    repoMetadata,
    loading,
    error,
    pullReqChecksDecision,
    showEditDescription,
    setShowEditDescription,
    pullReqMetadata,
    pullReqStats,
    pullReqCommits,
    pullRequestId,
    pullRequestSection,
    commitSHA,
    refetchActivities,
    refetchCommits,
    retryOnErrorFunc
  } = useGetPullRequestInfo()

  console.debug('overallStatus:', pullReqChecksDecision?.overallStatus)

  const gitUrl = repoMetadata?.git_url
  const prNumber = pullReqMetadata?.number
  const { data } = useFrontendExecutionForRepo(
    {
      queryParams: {
        accountId,
        orgId: orgIdentifier,
        projectId: projectIdentifier,
        gitUrl: gitUrl || '',
        prNumber,
      },
    },
    { enabled: gitUrl !== undefined && prNumber !== undefined }
  )

  const { data: countData } = useFrontendExecutionIssueCounts(
    {
      queryParams: {
        accountId,
        orgId: orgIdentifier,
        projectId: projectIdentifier,
        executionIds: data?.executionId || ''
      },
    },
    { enabled: data?.executionId !== undefined }
  )
  console.debug('countData:', countData)
  const counts = countData && data?.executionId !== undefined && countData[data?.executionId] ? countData[data?.executionId] : undefined
  console.debug('counts:', counts)
  const issueCount = counts ? counts.critical + counts.high + counts.medium + counts.low : undefined

  const onAddDescriptionClick = useCallback(() => {
    setShowEditDescription(true)
    history.replace(
      routes.toCODEPullRequest({
        repoPath: repoMetadata?.path as string,
        pullRequestId,
        pullRequestSection: PullRequestSection.CONVERSATION
      })
    )
  }, [history, routes, repoMetadata?.path, pullRequestId, setShowEditDescription])

  const activeTab = useMemo(
    () =>
      Object.values(PullRequestSection).find(value => value === pullRequestSection)
        ? pullRequestSection
        : PullRequestSection.CONVERSATION,
    [pullRequestSection]
  )

  const [pullReqChangesCount, setPullReqChangesCount] = useState(0)

  const domRef = useRef<HTMLDivElement>(null)
  useSetPageContainerWidthVar({ domRef })
  useScrollTop()

  return (
    <Container className={css.main} ref={domRef}>
      <RepositoryPageHeader
        repoMetadata={repoMetadata}
        title={
          repoMetadata && pullReqMetadata ? (
            <PullRequestTitle
              repoMetadata={repoMetadata}
              {...pullReqMetadata}
              onAddDescriptionClick={onAddDescriptionClick}
            />
          ) : (
            ''
          )
        }
        dataTooltipId="repositoryPullRequest"
        extraBreadcrumbLinks={
          repoMetadata && [
            {
              label: getString('pullRequests'),
              url: routes.toCODEPullRequests({ repoPath: repoMetadata.path as string })
            }
          ]
        }
      />
      <PageBody error={getErrorMessage(error)} retryOnError={retryOnErrorFunc}>
        <LoadingSpinner visible={loading} />

        <Render when={repoMetadata && pullReqMetadata}>
          <>
            <PullRequestMetaLine repoMetadata={repoMetadata as TypesRepository} {...pullReqMetadata} />

            <Container className={tabContainerCSS.tabsContainer}>
              <Tabs
                id="prTabs"
                defaultSelectedTabId={activeTab}
                selectedTabId={activeTab}
                large={false}
                onChange={tabId => {
                  history.push(
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
                        count={pullReqMetadata?.stats?.conversations || 0}
                      />
                    ),
                    panel: (
                      <Conversation
                        routingId={routingId}
                        standalone={standalone}
                        repoMetadata={repoMetadata as TypesRepository}
                        pullReqMetadata={pullReqMetadata as TypesPullReq}
                        prChecksDecisionResult={pullReqChecksDecision}
                        onDescriptionSaved={() => {
                          setShowEditDescription(false)
                        }}
                        prStats={pullReqStats}
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
                        count={pullReqStats?.commits || 0}
                        padding={{ left: 'medium' }}
                      />
                    ),
                    panel: (
                      <PullRequestCommits
                        repoMetadata={repoMetadata as TypesRepository}
                        pullReqMetadata={pullReqMetadata as TypesPullReq}
                        pullReqCommits={pullReqCommits}
                      />
                    )
                  },
                  {
                    id: PullRequestSection.FILES_CHANGED,
                    title: (
                      <TabTitleWithCount
                        icon={CodeIcon.File}
                        title={getString('filesChanged')}
                        count={pullReqChangesCount || pullReqStats?.files_changed || 0}
                        padding={{ left: 'medium' }}
                      />
                    ),
                    panel: (
                      <Container className={css.changes} data-page-section={PullRequestSection.FILES_CHANGED}>
                        {!!repoMetadata && !!pullReqMetadata && !!pullReqStats && (
                          <Changes
                            repoMetadata={repoMetadata}
                            pullRequestMetadata={pullReqMetadata}
                            pullReqCommits={pullReqCommits}
                            defaultCommitRange={compact(commitSHA?.split(/~1\.\.\.|\.\.\./g))}
                            targetRef={pullReqMetadata.merge_base_sha}
                            sourceRef={pullReqMetadata.source_sha}
                            emptyTitle={getString('noChanges')}
                            emptyMessage={getString('noChangesPR')}
                            refetchActivities={refetchActivities}
                            refetchCommits={refetchCommits}
                            setPullReqChangesCount={setPullReqChangesCount}
                            scrollElement={
                              (standalone
                                ? document.querySelector(`.${css.main}`)?.parentElement || window
                                : window) as HTMLElement
                            }
                          />
                        )}
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
                          pullReqChecksDecision?.overallStatus ? (
                            <Container className={css.checksCount}>
                              <Layout.Horizontal className={css.checksCountLayout}>
                                <ExecutionStatus
                                  status={pullReqChecksDecision?.overallStatus}
                                  noBackground
                                  iconOnly
                                  iconSize={15}
                                />

                                <Text
                                  color={pullReqChecksDecision?.color}
                                  padding={{ left: 'xsmall' }}
                                  tag="span"
                                  font={{ variation: FontVariation.FORM_MESSAGE_WARNING }}>
                                  {pullReqChecksDecision?.count[pullReqChecksDecision?.overallStatus]}
                                </Text>
                              </Layout.Horizontal>
                            </Container>
                          ) : null
                        }
                        count={pullReqChecksDecision?.count?.failure || 0}
                        padding={{ left: 'medium' }}
                      />
                    ),
                    panel: (
                      <Checks
                        repoMetadata={repoMetadata as TypesRepository}
                        pullReqMetadata={pullReqMetadata as TypesPullReq}
                        prChecksDecisionResult={pullReqChecksDecision}
                      />
                    )
                  },
                  {
                    id: PullRequestSection.SECURITY_ISSUES,
                    title: (
                      <TabTitleWithCount
                        icon="shield-gears"
                        iconSize={14}
                        title="New Security Issues"
                        countElement={
                          pullReqChecksDecision?.overallStatus ? (
                            <Container className={css.checksCount}>
                              <Layout.Horizontal className={css.checksCountLayout}>
                                {(pullReqChecksDecision?.overallStatus === ExecutionState.RUNNING
                                  || pullReqChecksDecision?.overallStatus === ExecutionState.ERROR)
                                  && (
                                  <ExecutionStatus
                                    status={pullReqChecksDecision?.overallStatus}
                                    noBackground
                                    iconOnly
                                    iconSize={20}
                                  />
                                )}
                              </Layout.Horizontal>
                            </Container>
                          ) : null
                        }
                        count={pullReqChecksDecision?.overallStatus !== ExecutionState.RUNNING ? issueCount : undefined}
                        padding={{ left: 'medium' }}
                      />
                    ),
                    panel: data?.executionId ? (
                      <Container width="100%" height="100%">
                        <ChildAppMounter<STOAppCustomProps>
                          ChildApp={RemoteSTOApp}
                          customComponents={{ /*UserLabel, UsefulOrNot*/ }}
                          customHooks={{ /*useGetSettingValue, useGetPipelineSummary*/ }}
                          parentContextObj={parentContextObj}
                        >
                          <RemotePipelineSecurityView pipelineExecutionDetail={{ pipelineExecutionSummary: { planExecutionId: data?.executionId } }} isPullRequest={true} />
                        </ChildAppMounter>
                      </Container>
                    ) : (
                      <PageSpinner />
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

// TODO:
// - Move pullReqChecksDecision to individual component to avoid weird
//   re-rendering everything in this page during its polling. Use an atom?
