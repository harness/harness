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

import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { useGet, useMutate } from 'restful-react'
import { ButtonVariation, Container, SplitButton, useToaster, Text, Layout } from '@harnessio/uicore'
import { Icon, IconName } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { Menu, PopoverPosition } from '@blueprintjs/core'
import cx from 'classnames'
import { isEmpty } from 'lodash-es'
import { useStrings } from 'framework/strings'
import type { EnumPullReqReviewDecision, TypesPullReq } from 'services/code'
import type { GitInfoProps } from 'utils/GitUtils'
import { getErrorMessage } from 'utils/Utils'
import css from '../Changes.module.scss'

interface PrReviewOption {
  method: EnumPullReqReviewDecision | 'reject'
  title: string
  disabled?: boolean
  icon: IconName
  color: Color
}

enum ApproveState {
  APPROVED = 'approved',
  CHANGEREQ = 'changereq',
  APPROVE = 'approve',
  OUTDATED = 'outdated'
}
interface ReviewSplitButtonProps extends Pick<GitInfoProps, 'repoMetadata'> {
  shouldHide: boolean
  pullRequestMetadata?: TypesPullReq
  refreshPr: () => void
  disabled?: boolean
  refetchReviewers?: () => void
}

const ReviewSplitButton = (props: ReviewSplitButtonProps) => {
  const { refetchReviewers, pullRequestMetadata, repoMetadata, shouldHide, refreshPr, disabled } = props
  const { getString } = useStrings()
  const { showError, showSuccess } = useToaster()

  const { data: reviewers, refetch: updateReviewers } = useGet<Unknown[]>({
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/reviewers`
  })

  const determineOverallDecision = (data: any[] | null) => {
    let hasChangeReq = false
    let allApproved = true
    if (data === null || isEmpty(data)) {
      return ApproveState.APPROVE // Default case
    }
    for (const item of data) {
      if (item.review_decision === ApproveState.CHANGEREQ) {
        hasChangeReq = true
        break
      } else if (item.review_decision !== ApproveState.APPROVED) {
        allApproved = false
      }
    }

    if (hasChangeReq) {
      return ApproveState.CHANGEREQ
    } else if (allApproved) {
      return ApproveState.APPROVED
    } else {
      return ApproveState.APPROVE // Default case
    }
  }
  const [commitSha, setCommitSha] = useState('')
  useEffect(() => {
    if (reviewers) {
      if (reviewers[0] && reviewers[0].sha) {
        setCommitSha(reviewers[0].sha)
      }
      setApproveState(determineOverallDecision(reviewers))
    }
  }, [reviewers])
  const [approveState, setApproveState] = useState(determineOverallDecision(reviewers))

  const prDecisionOptions: PrReviewOption[] = useMemo(
    () => [
      {
        method: 'approved',
        title: getString('approve'),
        icon: 'tick-circle' as IconName,
        color: Color.GREEN_700
      },
      {
        method: 'changereq',
        title: getString('requestChanges'),
        icon: 'error' as IconName,
        color: Color.ORANGE_700
      }
    ],
    [getString]
  )

  const { mutate, loading } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullRequestMetadata?.number}/reviews`
  })
  const submitReview = useCallback(
    (decision: PrReviewOption) => {
      mutate({ decision: decision.method, commit_sha: pullRequestMetadata?.source_sha })
        .then(() => {
          showSuccess(getString(decision.method === 'approved' ? 'pr.reviewSubmitted' : 'pr.requestSubmitted'))
          refreshPr?.()
          refetchReviewers?.()
          updateReviewers()
        })
        .catch(exception => showError(getErrorMessage(exception)))
    },
    [mutate, showError, showSuccess, getString, refreshPr, pullRequestMetadata?.source_sha, refetchReviewers]
  )

  const processReviewDecision = (review_decision: ApproveState, reviewedSHA?: string, sourceSHA?: string) =>
    (review_decision === ApproveState.APPROVED && reviewedSHA !== sourceSHA) ||
    (review_decision === ApproveState.CHANGEREQ && reviewedSHA !== sourceSHA)
      ? ApproveState.OUTDATED
      : review_decision

  const getApprovalState = (state: string) => {
    const checkOutdated = processReviewDecision(approveState, commitSha, pullRequestMetadata?.source_sha)

    if (
      (state === ApproveState.APPROVED && checkOutdated === ApproveState.OUTDATED) ||
      (state === ApproveState.CHANGEREQ && checkOutdated === ApproveState.OUTDATED)
    ) {
      return getString('approve')
    } else if (state === ApproveState.APPROVED) {
      return getString('approved')
    } else if (state === ApproveState.CHANGEREQ) {
      return getString('requestChanges')
    } else {
      return getString('approve')
    }
  }

  const showMenuItem = (methodState: string) => {
    if (
      getApprovalState(approveState).toLocaleLowerCase() === ApproveState.APPROVE &&
      methodState === ApproveState.APPROVED
    ) {
      return false
    }
    if (
      methodState !== approveState &&
      processReviewDecision(approveState, commitSha, pullRequestMetadata?.source_sha) === ApproveState.OUTDATED
    ) {
      return true
    } else if (methodState !== approveState) {
      return true
    } else if (
      processReviewDecision(approveState, commitSha, pullRequestMetadata?.source_sha) === ApproveState.OUTDATED
    ) {
      return true
    }
    return false
  }

  return (
    <Container
      className={cx(css.reviewButton, {
        [css.hide]: shouldHide,
        [css.disabled]: disabled
      })}>
      <SplitButton
        className={cx(
          {
            [css.approvedBtn]:
              approveState === ApproveState.APPROVED &&
              processReviewDecision(approveState, commitSha, pullRequestMetadata?.source_sha) !== ApproveState.OUTDATED
          },
          {
            [css.changeReqBtn]:
              approveState === ApproveState.CHANGEREQ &&
              processReviewDecision(approveState, commitSha, pullRequestMetadata?.source_sha) !== ApproveState.OUTDATED
          }
        )}
        text={approveState === ApproveState.APPROVE ? prDecisionOptions[0].title : getApprovalState(approveState)}
        disabled={loading}
        variation={
          (approveState === ApproveState.APPROVED &&
            processReviewDecision(approveState, commitSha, pullRequestMetadata?.source_sha) ===
              ApproveState.OUTDATED) ||
          (ApproveState.CHANGEREQ &&
            processReviewDecision(approveState, commitSha, pullRequestMetadata?.source_sha) ===
              ApproveState.OUTDATED) ||
          approveState === ApproveState.APPROVE
            ? ButtonVariation.SECONDARY
            : ButtonVariation.PRIMARY
        }
        popoverProps={{
          interactionKind: 'click',
          usePortal: true,
          popoverClassName: css.popover,
          position: PopoverPosition.BOTTOM_RIGHT,
          transitionDuration: 1000
        }}
        onClick={() => {
          if (
            approveState === ApproveState.APPROVE ||
            processReviewDecision(approveState, commitSha, pullRequestMetadata?.source_sha) === ApproveState.OUTDATED
          ) {
            submitReview(prDecisionOptions[0])
          }
        }}>
        {showMenuItem(prDecisionOptions[0].method) && (
          <Menu.Item
            key={prDecisionOptions[0].method}
            className={cx(css.menuReviewItem, {
              [css.btnDisabled]: prDecisionOptions[0].method === getApprovalState(approveState)
            })}
            disabled={disabled || prDecisionOptions[0].disabled || !showMenuItem(prDecisionOptions[0].method)}
            text={
              <Layout.Horizontal>
                <Icon
                  className={css.reviewIcon}
                  {...(prDecisionOptions[0].icon === 'danger-icon' ? null : { color: prDecisionOptions[0].color })}
                  size={16}
                  name={prDecisionOptions[0].icon}
                />
                <Text flex width={'fit-content'} font={{ variation: FontVariation.BODY }}>
                  {prDecisionOptions[0].title}
                </Text>
              </Layout.Horizontal>
            }
            onClick={() => {
              submitReview(prDecisionOptions[0])
            }}
          />
        )}
        {showMenuItem(prDecisionOptions[1].method) && (
          <Menu.Item
            key={prDecisionOptions[1].method}
            className={cx(css.menuReviewItem, {
              [css.btnDisabled]: prDecisionOptions[1].method === getApprovalState(approveState)
            })}
            disabled={disabled || prDecisionOptions[1].disabled || !showMenuItem(prDecisionOptions[1].method)}
            text={
              <Layout.Horizontal>
                <Icon
                  className={css.reviewIcon}
                  {...(prDecisionOptions[1].icon === 'danger-icon' ? null : { color: prDecisionOptions[1].color })}
                  size={16}
                  name={prDecisionOptions[1].icon}
                />
                <Text flex width={'fit-content'} font={{ variation: FontVariation.BODY }}>
                  {getString('reqChanges')}
                </Text>
              </Layout.Horizontal>
            }
            onClick={() => {
              submitReview(prDecisionOptions[1])
            }}
          />
        )}
      </SplitButton>
    </Container>
  )
}

export default ReviewSplitButton
