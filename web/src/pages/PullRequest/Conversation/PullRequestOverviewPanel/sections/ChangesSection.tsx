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
import { Color, FontVariation } from '@harnessio/design-system'
import {
  Button,
  ButtonSize,
  Text,
  ButtonVariation,
  Container,
  Layout,
  useToggle,
  stringSubstitute
} from '@harnessio/uicore'
import React, { useEffect, useMemo, useState } from 'react'
import cx from 'classnames'
import { Render } from 'react-jsx-match'
import { isEmpty } from 'lodash-es'
import type { IconName } from '@blueprintjs/core'
import { Icon } from '@harnessio/icons'
import { CodeOwnerReqDecision, findChangeReqDecisions, findWaitingDecisions } from 'utils/Utils'
import { CodeOwnerSection } from 'pages/PullRequest/CodeOwners/CodeOwnersOverview'
import { useStrings } from 'framework/strings'
import type {
  TypesCodeOwnerEvaluation,
  TypesCodeOwnerEvaluationEntry,
  TypesPullReq,
  TypesPullReqReviewer,
  RepoRepositoryOutput
} from 'services/code'
import { capitalizeFirstLetter } from 'pages/PullRequest/Checks/ChecksUtils'
import greyCircle from '../../../../../icons/greyCircle.svg?url'
import emptyStatus from '../../../../../icons/emptyStatus.svg?url'
import Success from '../../../../../icons/code-success.svg?url'
import Fail from '../../../../../icons/code-fail.svg?url'
import FailGrey from '../../../../../icons/code-fail-grey.svg?url'
import Timeout from '../../../../../icons/code-timeout.svg?url'
import css from '../PullRequestOverviewPanel.module.scss'

interface ChangesSectionProps {
  repoMetadata: RepoRepositoryOutput
  pullReqMetadata: TypesPullReq
  codeOwners: TypesCodeOwnerEvaluation | null
  atLeastOneReviewerRule: boolean
  reqCodeOwnerApproval: boolean
  minApproval: number
  reviewers: TypesPullReqReviewer[] | null
  minReqLatestApproval: number
  reqCodeOwnerLatestApproval: boolean
  loadingReviewers: boolean
  refetchReviewers: () => void
  refetchCodeOwners: () => void
}

const ChangesSection = (props: ChangesSectionProps) => {
  const {
    reviewers: currReviewers,
    minApproval,
    reqCodeOwnerApproval,
    repoMetadata,
    pullReqMetadata,
    codeOwners: currCodeOwners,
    atLeastOneReviewerRule,
    reqCodeOwnerLatestApproval,
    minReqLatestApproval,
    loadingReviewers,
    refetchReviewers,
    refetchCodeOwners
  } = props
  const reqNoChangeReq = atLeastOneReviewerRule
  const { getString } = useStrings()
  const [headerText, setHeaderText] = useState('')
  const [contentText, setContentText] = useState('')
  const [color, setColor] = useState('')
  const [loading, setLoading] = useState(true)
  const [status, setStatus] = useState('tick-circle')
  const [isExpanded, toggleExpanded] = useToggle(false)
  const [isNotRequiredFlag, setIsNotRequired] = useState(false)
  const reviewers = useMemo(() => {
    refetchCodeOwners()
    return currReviewers // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currReviewers, refetchReviewers])

  const codeOwners = useMemo(() => {
    return currCodeOwners // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currCodeOwners, refetchCodeOwners, refetchReviewers])

  const checkIfOutdatedSha = (reviewedSHA?: string, sourceSHA?: string) =>
    reviewedSHA !== sourceSHA || reviewedSHA !== sourceSHA ? true : false
  const codeOwnerChangeReqEntries = findChangeReqDecisions(
    codeOwners?.evaluation_entries,
    CodeOwnerReqDecision.CHANGEREQ
  )

  const codeOwnerApprovalEntries = findChangeReqDecisions(codeOwners?.evaluation_entries, CodeOwnerReqDecision.APPROVED)

  const latestCodeOwnerApprovalArr = codeOwnerApprovalEntries
    ?.map(entry => {
      // Filter the owner_evaluations for 'changereq' decisions
      const entryEvaluation = entry?.owner_evaluations.filter(
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (evaluation: any) => !checkIfOutdatedSha(evaluation?.review_sha, pullReqMetadata?.source_sha as string)
      )
      // If there are any 'changereq' decisions, return the entry along with them
      if (entryEvaluation && entryEvaluation?.length > 0) {
        return { entryEvaluation }
      }
    }) // eslint-disable-next-line @typescript-eslint/no-explicit-any
    .filter((entry: any) => entry !== null && entry !== undefined) // Filter out the null entries

  const codeOwnerPendingEntries = findWaitingDecisions(codeOwners?.evaluation_entries)
  const approvedEvaluations = reviewers?.filter(evaluation => evaluation.review_decision === 'approved')
  const latestApprovalArr = approvedEvaluations?.filter(
    approved => !checkIfOutdatedSha(approved.sha, pullReqMetadata?.source_sha as string)
  )

  const changeReqEvaluations = reviewers?.filter(evaluation => evaluation.review_decision === 'changereq')
  const changeReqReviewer =
    changeReqEvaluations && !isEmpty(changeReqEvaluations)
      ? capitalizeFirstLetter(
          changeReqEvaluations[0].reviewer?.display_name || changeReqEvaluations[0].reviewer?.uid || ''
        )
      : 'Reviewer'
  const extractInfoForCodeOwnerContent = () => {
    let statusMessage = ''
    let statusColor = 'grey' // Default color for no rules required
    let title = ''
    let statusIcon = ''
    let isNotRequired = false
    if (
      reqNoChangeReq ||
      reqCodeOwnerApproval ||
      minApproval > 0 ||
      reqCodeOwnerLatestApproval ||
      minReqLatestApproval > 0
    ) {
      if (codeOwnerChangeReqEntries.length > 0 && (reqCodeOwnerApproval || reqCodeOwnerLatestApproval)) {
        title = getString('changesSection.reqChangeFromCodeOwners')
        statusMessage = getString('changesSection.codeOwnerReqChanges')
        statusColor = Color.RED_700
        statusIcon = 'warning-icon'
      } else if (changeReqEvaluations && changeReqEvaluations?.length > 0 && reqNoChangeReq) {
        title = getString('requestChanges')
        statusMessage = getString('pr.requestedChanges', { user: changeReqReviewer })
        statusColor = Color.RED_700
        statusIcon = 'warning-icon'
      } else if (
        (codeOwnerPendingEntries && codeOwnerPendingEntries?.length > 0 && reqCodeOwnerLatestApproval) ||
        (!isEmpty(latestCodeOwnerApprovalArr) &&
          latestCodeOwnerApprovalArr?.length < minReqLatestApproval &&
          reqCodeOwnerLatestApproval)
      ) {
        title = getString('changesSection.pendingAppFromCodeOwners')
        statusMessage = getString('changesSection.pendingLatestApprovalCodeOwners')
        statusColor = Color.ORANGE_500
        statusIcon = 'execution-waiting'
      } else if (codeOwnerPendingEntries && codeOwnerPendingEntries?.length > 0 && reqCodeOwnerApproval) {
        title = getString('changesSection.pendingAppFromCodeOwners')
        statusMessage = getString('changesSection.waitingOnCodeOwner')
        statusColor = Color.ORANGE_500
        statusIcon = 'execution-waiting'
      } else if (minReqLatestApproval > 0 && latestApprovalArr && latestApprovalArr?.length < minReqLatestApproval) {
        title = getString('changesSection.approvalPending')
        statusMessage = getString('changesSection.latestChangesPendingReqRev')
        statusColor = Color.ORANGE_500
        statusIcon = 'execution-waiting'
      } else if (approvedEvaluations && approvedEvaluations?.length < minApproval && minApproval > 0) {
        title = getString('changesSection.approvalPending')
        statusMessage = stringSubstitute(getString('changesSection.waitingOnReviewers'), {
          count: approvedEvaluations?.length || '0',
          total: minApproval
        }) as string

        statusColor = Color.ORANGE_500
        statusIcon = 'execution-waiting'
      } else if (reqCodeOwnerLatestApproval && latestCodeOwnerApprovalArr?.length > 0) {
        title = getString('changesSection.changesApproved')
        statusMessage = getString('changesSection.latestChangesWereAppByCodeOwner')
        statusColor = Color.GREEN_700
        statusIcon = 'tick-circle'
      } else if (reqCodeOwnerApproval && codeOwnerApprovalEntries?.length > 0) {
        title = getString('changesSection.changesApproved')
        statusMessage = getString('changesSection.changesWereAppByCodeOwner')
        statusColor = Color.GREEN_700
        statusIcon = 'tick-circle'
      } else if (minReqLatestApproval > 0 && latestApprovalArr && latestApprovalArr?.length >= minReqLatestApproval) {
        title = getString('changesSection.changesApproved')
        statusMessage = getString('changesSection.latestChangesWereApprovedByReq')
        statusColor = Color.GREEN_700
        statusIcon = 'tick-circle'
      } else if (minApproval > 0 && approvedEvaluations && approvedEvaluations?.length >= minApproval) {
        title = getString('changesSection.changesApproved')
        statusMessage = getString('changesSection.changesWereAppByLatestReqRev')
        statusColor = Color.GREEN_700
        statusIcon = 'tick-circle'
      } else if (approvedEvaluations && approvedEvaluations?.length > 0) {
        title = getString('changesSection.changesApproved')
        statusMessage = stringSubstitute(getString('changesSection.changesAppByRev')) as string
        statusColor = Color.GREEN_700
        statusIcon = 'tick-circle'
      } else {
        title = getString('changesSection.noReviewsReq')
        statusMessage = getString('changesSection.pullReqWithoutAnyReviews')
        statusColor = Color.GREY_500
        statusIcon = 'tick-circle'
      }
    } else {
      // When no rules are required
      if (codeOwnerChangeReqEntries && codeOwnerChangeReqEntries?.length > 0) {
        title = getString('changesSection.reqChangeFromCodeOwners')
        statusMessage = getString('changesSection.codeOwnerReqChanges')
        statusIcon = 'warning-icon'
        isNotRequired = true
      } else if (changeReqEvaluations && changeReqEvaluations?.length > 0) {
        title = getString('requestChanges')
        statusMessage = getString('pr.requestedChanges', { user: changeReqReviewer })
        statusIcon = 'warning-icon'
        isNotRequired = true
      } else if (approvedEvaluations?.length && approvedEvaluations?.length > 0) {
        title = getString('changesSection.changesApproved')
        statusMessage = stringSubstitute(getString('changesSection.changesAppByRev')) as string
        statusIcon = 'tick-circle'
      } else {
        title = getString('changesSection.noReviewsReq')
        statusMessage = getString('changesSection.pullReqWithoutAnyReviews')
        statusIcon = 'tick-circle'
      }
    }
    return { title, statusMessage, statusColor, statusIcon, isNotRequired }
  }
  useEffect(() => {
    setLoading(true)
    const { title, statusColor, statusMessage, statusIcon: curStatus, isNotRequired } = extractInfoForCodeOwnerContent()
    setHeaderText(title)
    setContentText(statusMessage)
    setColor(statusColor)
    setLoading(false)
    setStatus(curStatus)
    setIsNotRequired(isNotRequired) // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    currReviewers,
    currCodeOwners,
    reqNoChangeReq,
    reqCodeOwnerApproval,
    minApproval,
    reqCodeOwnerLatestApproval,
    minReqLatestApproval,
    refetchReviewers,
    refetchCodeOwners
  ])

  function renderCodeOwnerStatus() {
    if (codeOwnerPendingEntries?.length > 0 && reqCodeOwnerLatestApproval) {
      return (
        <Layout.Horizontal>
          <Container padding={{ left: 'large' }}>
            <img alt="emptyStatus" width={16} height={16} src={emptyStatus} />
          </Container>

          <Text padding={{ left: 'medium' }} className={css.sectionSubheader}>
            {getString('changesSection.waitingOnLatestCodeOwner')}
          </Text>
        </Layout.Horizontal>
      )
    }

    if (codeOwnerPendingEntries?.length > 0 && reqCodeOwnerApproval) {
      return (
        <Layout.Horizontal>
          <Container padding={{ left: 'large' }}>
            <img alt="emptyStatus" width={16} height={16} src={emptyStatus} />
          </Container>

          <Text padding={{ left: 'medium' }} className={css.sectionSubheader}>
            {getString('changesSection.waitingOnCodeOwner')}
          </Text>
        </Layout.Horizontal>
      )
    }

    if (
      (codeOwnerApprovalEntries as TypesCodeOwnerEvaluationEntry[])?.length > 0 &&
      codeOwnerPendingEntries?.length > 0
    ) {
      return (
        <Layout.Horizontal>
          <Container padding={{ left: 'large' }}>
            <img alt="greyCircle" width={16} height={16} src={greyCircle} />
          </Container>
          <Text padding={{ left: 'medium' }} className={css.sectionSubheader}>
            {getString('changesSection.someChangesWereAppByCodeOwner')}
          </Text>
        </Layout.Horizontal>
      )
    }
    if (latestCodeOwnerApprovalArr?.length > 0 && reqCodeOwnerLatestApproval) {
      return (
        <Text
          icon={'tick-circle'}
          iconProps={{
            size: 16,
            color: Color.GREEN_700,
            padding: { right: 'medium' }
          }}
          padding={{ left: 'large' }}
          className={css.sectionSubheader}>
          {getString('changesSection.latestChangesWereAppByCodeOwner')}
        </Text>
      )
    }
    if (codeOwnerApprovalEntries?.length > 0 && reqCodeOwnerApproval) {
      return (
        <Text
          icon={'tick-circle'}
          iconProps={{
            size: 16,
            color: Color.GREEN_700,
            padding: { right: 'medium' }
          }}
          padding={{ left: 'large' }}
          className={css.sectionSubheader}>
          {getString('changesSection.changesWereAppByCodeOwner')}
        </Text>
      )
    }
    if (codeOwnerApprovalEntries?.length > 0) {
      if (reqCodeOwnerLatestApproval && latestCodeOwnerApprovalArr.length < minReqLatestApproval) {
        return (
          <Layout.Horizontal>
            <Container padding={{ left: 'large' }}>
              <img alt="emptyStatus" width={16} height={16} src={emptyStatus} />
            </Container>

            <Text padding={{ left: 'medium' }} className={css.sectionSubheader}>
              {getString('changesSection.latestChangesPendingReqRev')}
            </Text>
          </Layout.Horizontal>
        )
      }
      return (
        <Text
          icon={'tick-circle'}
          iconProps={{
            size: 16,
            color: Color.GREEN_700,
            padding: { right: 'medium' }
          }}
          padding={{ left: 'large' }}
          className={css.sectionSubheader}>
          {getString('changesSection.changesWereAppByCodeOwner')}
        </Text>
      )
    }

    return (
      <Layout.Horizontal>
        <Container padding={{ left: 'large' }}>
          <img alt="greyCircle" width={16} height={16} src={greyCircle} />
        </Container>
        <Text padding={{ left: 'medium' }} className={css.sectionSubheader}>
          {getString('changesSection.noCodeOwnerReviewsReq')}
        </Text>
      </Layout.Horizontal>
    )
  }
  const viewBtn =
    minApproval > minReqLatestApproval ||
    (!isEmpty(approvedEvaluations) && minReqLatestApproval === 0) ||
    (minApproval > 0 && minReqLatestApproval === undefined) ||
    minReqLatestApproval > 0 ||
    !isEmpty(changeReqEvaluations) ||
    !isEmpty(codeOwners) ||
    false
  return (
    <Render when={!loading && !loadingReviewers && status}>
      <Container className={cx(css.sectionContainer, css.borderContainer)}>
        <Layout.Horizontal flex={{ justifyContent: 'space-between' }}>
          <Layout.Horizontal flex={{ alignItems: 'center' }}>
            {status === 'tick-circle' ? (
              <img alt={getString('success')} width={26} height={26} src={Success} />
            ) : status === 'warning-icon' ? (
              <img alt={getString('failed')} width={26} height={26} src={isNotRequiredFlag ? FailGrey : Fail} />
            ) : status === 'execution-waiting' ? (
              <img alt={getString('waiting')} width={27} height={27} src={Timeout} className={css.timeoutIcon} />
            ) : (
              <Icon className={css.statusIcon} name={status as IconName} size={25} color={color} />
            )}
            <Layout.Vertical padding={{ left: 'medium' }}>
              <Text
                padding={{ bottom: 'xsmall' }}
                className={css.sectionTitle}
                color={
                  headerText === getString('changesSection.noReviewsReq')
                    ? color
                    : headerText === getString('changesSection.changesApproved')
                    ? Color.GREEN_800
                    : color
                }>
                {headerText}
              </Text>
              <Text className={css.sectionSubheader} color={Color.GREY_450} font={{ variation: FontVariation.BODY }}>
                {contentText}
              </Text>
            </Layout.Vertical>
          </Layout.Horizontal>
          {viewBtn && (
            <Button
              padding={{ right: 'unset' }}
              className={cx(css.blueText, css.buttonPadding)}
              variation={ButtonVariation.LINK}
              size={ButtonSize.SMALL}
              text={getString(isExpanded ? 'showLessMatches' : 'showMoreText')}
              onClick={toggleExpanded}
              rightIcon={isExpanded ? 'main-chevron-up' : 'main-chevron-down'}
              iconProps={{ size: 10, margin: { left: 'xsmall' } }}
            />
          )}
        </Layout.Horizontal>
      </Container>

      <Render when={isExpanded}>
        <Container className={css.greyContainer}>
          {minApproval > minReqLatestApproval ||
            (!isEmpty(approvedEvaluations) && minReqLatestApproval === 0) ||
            (minApproval > 0 && minReqLatestApproval === undefined && (
              <Container padding={{ left: 'xlarge', right: 'small' }} className={css.borderContainer}>
                <Layout.Horizontal
                  className={css.paddingContainer}
                  flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
                  {approvedEvaluations && minApproval <= approvedEvaluations?.length ? (
                    <Text
                      icon="tick-circle"
                      iconProps={{ size: 16, color: Color.GREEN_700 }}
                      padding={{
                        left: 'large'
                      }}>
                      <Text className={cx(css.sectionSubheader, css.sectionPadding)}>
                        {
                          stringSubstitute(getString('changesSection.changesApprovedByXReviewers'), {
                            length: approvedEvaluations?.length || '0'
                          }) as string
                        }
                      </Text>
                    </Text>
                  ) : (
                    <Layout.Horizontal>
                      <Container padding={{ left: 'large' }}>
                        <img alt="emptyStatus" width={16} height={16} src={emptyStatus} />
                      </Container>

                      <Text
                        className={css.sectionSubheader}
                        padding={{
                          left: 'medium'
                        }}>
                        {
                          stringSubstitute(getString('codeOwner.approvalCompleted'), {
                            count: approvedEvaluations?.length || '0',
                            total: minApproval
                          }) as string
                        }
                      </Text>
                    </Layout.Horizontal>
                  )}
                  <Container margin={{ bottom: `2px` }} className={css.changeContainerPadding}>
                    <Container className={css.requiredContainer}>
                      <Text className={css.requiredText}>{getString('required')}</Text>
                    </Container>
                  </Container>
                </Layout.Horizontal>
              </Container>
            ))}
          {minReqLatestApproval > 0 && (
            <Container padding={{ left: 'xlarge', right: 'small' }} className={css.borderContainer}>
              <Layout.Horizontal
                className={css.paddingContainer}
                flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
                {latestApprovalArr && minReqLatestApproval <= latestApprovalArr?.length ? (
                  <Text
                    icon="tick-circle"
                    className={css.successIcon}
                    iconProps={{ size: 16, color: Color.GREEN_700 }}
                    padding={{
                      left: 'large'
                    }}>
                    <Text
                      className={css.sectionSubheader}
                      padding={{
                        left: 'medium'
                      }}>
                      {
                        stringSubstitute(getString('changesSection.latestChangesApprovedByXReviewers'), {
                          length: latestApprovalArr?.length || minReqLatestApproval || 0
                        }) as string
                      }
                    </Text>
                  </Text>
                ) : (
                  <Layout.Horizontal>
                    <Container padding={{ left: 'large' }}>
                      <img alt="emptyStatus" width={16} height={16} src={emptyStatus} />
                    </Container>

                    <Text
                      className={css.sectionSubheader}
                      padding={{
                        left: 'medium'
                      }}>
                      {
                        stringSubstitute(getString('codeOwner.pendingLatestApprovals'), {
                          count: latestApprovalArr?.length || minReqLatestApproval || 0
                        }) as string
                      }
                    </Text>
                  </Layout.Horizontal>
                )}
                <Container margin={{ bottom: `2px` }} className={css.changeContainerPadding}>
                  <Container className={css.requiredContainer}>
                    <Text className={css.requiredText}>{getString('required')}</Text>
                  </Container>
                </Container>
              </Layout.Horizontal>
            </Container>
          )}
          {!isEmpty(changeReqEvaluations) && (
            <Container className={css.borderContainer} padding={{ left: 'xlarge', right: 'small' }}>
              <Layout.Horizontal className={css.paddingContainer} flex={{ justifyContent: 'space-between' }}>
                <Text
                  className={cx(
                    css.sectionSubheader,

                    {
                      [css.greyIcon]: !reqNoChangeReq
                    },
                    {
                      [css.redIcon]: reqNoChangeReq
                    }
                  )}
                  icon={'error-transparent-no-outline'}
                  iconProps={{ size: 17, color: reqNoChangeReq ? Color.RED_600 : '', padding: { right: 'medium' } }}
                  padding={{ left: 'large' }}>
                  {getString('pr.requestedChanges', { user: changeReqReviewer })}
                </Text>
                {reqNoChangeReq && (
                  <Container className={css.changeContainerPadding}>
                    <Container className={css.requiredContainer}>
                      <Text className={css.requiredText}>{getString('required')}</Text>
                    </Container>
                  </Container>
                )}
              </Layout.Horizontal>
            </Container>
          )}
          {!isEmpty(codeOwners) && !isEmpty(codeOwners.evaluation_entries) && (
            <Container className={css.borderContainer} padding={{ left: 'xlarge', right: 'small' }}>
              <Layout.Horizontal className={css.paddingContainer} flex={{ justifyContent: 'space-between' }}>
                {codeOwnerChangeReqEntries && codeOwnerChangeReqEntries?.length > 0 ? (
                  <Text
                    className={cx(
                      css.sectionSubheader,
                      reqCodeOwnerApproval || reqCodeOwnerLatestApproval ? css.redIcon : css.greyIcon
                    )}
                    icon={'error-transparent-no-outline'}
                    iconProps={{
                      size: 17,
                      color: reqCodeOwnerApproval || reqCodeOwnerLatestApproval ? Color.RED_600 : '',
                      padding: { right: 'medium' }
                    }}
                    padding={{ left: 'large' }}>
                    {getString('changesSection.codeOwnerReqChangesToPr')}
                  </Text>
                ) : (
                  renderCodeOwnerStatus()
                )}
                {(reqCodeOwnerApproval || reqCodeOwnerLatestApproval) && (
                  <Container className={css.changeContainerPadding}>
                    <Container className={css.requiredContainer}>
                      <Text className={css.requiredText}>{getString('required')}</Text>
                    </Container>
                  </Container>
                )}
              </Layout.Horizontal>
            </Container>
          )}
        </Container>
        {codeOwners && !isEmpty(codeOwners?.evaluation_entries) && (
          <Container
            className={css.codeOwnerContainer}
            padding={{ top: 'small', bottom: 'small', left: 'xxxlarge', right: 'small' }}>
            <CodeOwnerSection data={codeOwners} pullReqMetadata={pullReqMetadata} repoMetadata={repoMetadata} />
          </Container>
        )}
      </Render>
    </Render>
  )
}

export default ChangesSection
