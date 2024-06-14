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

import React, { useCallback, useMemo, useRef, useState } from 'react'
import { Container, Layout, PageBody, Tabs, Text } from '@harnessio/uicore'
import { FontVariation } from '@harnessio/design-system'
import { Render } from 'react-jsx-match'
import { useHistory } from 'react-router-dom'
import { compact } from 'lodash-es'
import { useAppContext } from 'AppContext'
import { useStrings } from 'framework/strings'
import { RepositoryPageHeader } from 'components/RepositoryPageHeader/RepositoryPageHeader'
import { getErrorMessage, PullRequestSection } from 'utils/Utils'
import { CodeIcon } from 'utils/GitUtils'
import type { TypesPullReq, TypesRepository } from 'services/code'
import { LoadingSpinner } from 'components/LoadingSpinner/LoadingSpinner'
import { TabTitleWithCount, tabContainerCSS } from 'components/TabTitleWithCount/TabTitleWithCount'
import { ExecutionStatus } from 'components/ExecutionStatus/ExecutionStatus'
import { useSetPageContainerWidthVar } from 'hooks/useSetPageContainerWidthVar'
import { useScrollTop } from 'hooks/useScrollTop'
import { PullRequestMetaLine } from './PullRequestMetaLine'
import { Conversation } from './Conversation/Conversation'
import { Checks } from './Checks/Checks'
import { Changes } from '../../components/Changes/Changes'
import { PullRequestCommits } from './PullRequestCommits/PullRequestCommits'
import { PullRequestTitle } from './PullRequestTitle'
import { useGetPullRequestInfo } from './useGetPullRequestInfo'
import css from './PullRequest.module.scss'

export default function PullRequest() {
  const history = useHistory()
  const { getString } = useStrings()
  const { routes, standalone, routingId } = useAppContext()
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
                        pullReqCommits={pullReqCommits}
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
                        icon={CodeIcon.CheckIcon}
                        iconSize={16}
                        title={getString('checks')}
                        countElement={
                          pullReqChecksDecision?.overallStatus ? (
                            <Container className={css.checksCount}>
                              <Layout.Horizontal className={css.checksCountLayout}>
                                <ExecutionStatus
                                  status={pullReqChecksDecision?.overallStatus}
                                  noBackground
                                  iconOnly
                                  inPr
                                  iconSize={pullReqChecksDecision?.overallStatus === 'failure' ? 17 : 15}
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
