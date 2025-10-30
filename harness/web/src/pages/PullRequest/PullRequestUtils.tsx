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

import { stringSubstitute, type SelectOption } from '@harnessio/uicore'
import type { IconName } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { isEmpty } from 'lodash-es'
import type { UseStringsReturn } from 'framework/strings'
import type { CommentItem } from 'components/CommentBox/CommentBox'
import type { ColorName, PullRequestSection } from 'utils/Utils'
import { MergeStrategy } from 'utils/GitUtils'
import type {
  EnumMergeMethod,
  EnumPullReqReviewDecision,
  TypesCodeOwnerEvaluationEntry,
  TypesDefaultReviewerApprovalsResponse,
  TypesOwnerEvaluation,
  TypesPrincipalInfo,
  TypesPullReq,
  TypesPullReqActivity,
  TypesPullReqReviewer,
  TypesRuleViolations,
  TypesViolation
} from 'services/code'
import { ExecutionState } from 'components/ExecutionStatus/ExecutionStatus'

export interface PRMergeOption extends SelectOption {
  method: EnumMergeMethod | 'close'
  title: string
  desc: string
  disabled?: boolean
}

export interface PRDraftOption {
  method: 'close' | 'open'
  title: string
  desc: string
  disabled?: boolean
}

export enum CommentType {
  COMMENT = 'comment',
  CODE_COMMENT = 'code-comment',
  TITLE_CHANGE = 'title-change',
  REVIEW_SUBMIT = 'review-submit',
  MERGE = 'merge',
  BRANCH_UPDATE = 'branch-update',
  BRANCH_DELETE = 'branch-delete',
  BRANCH_RESTORE = 'branch-restore',
  STATE_CHANGE = 'state-change',
  LABEL_MODIFY = 'label-modify',
  REVIEWER_ADD = 'reviewer-add',
  USER_GROUP_REVIEWER_ADD = 'user-group-reviewer-add',
  REVIEWER_DELETE = 'reviewer-delete',
  USER_GROUP_REVIEWER_DELETE = 'user-group-reviewer-delete',
  TARGET_BRANCH_CHANGE = 'target-branch-change'
}

export function isCodeComment(commentItems: CommentItem<TypesPullReqActivity>[]) {
  return commentItems[0]?.payload?.type === CommentType.CODE_COMMENT
}

export function isComment(commentItems: CommentItem<TypesPullReqActivity>[]) {
  return commentItems[0]?.payload?.type === CommentType.COMMENT
}

export function isSystemComment(commentItems: CommentItem<TypesPullReqActivity>[]) {
  return commentItems[0].payload?.kind === 'system'
}

export enum PullReqReviewDecision {
  APPROVED = 'approved',
  CHANGEREQ = 'changereq',
  PENDING = 'pending',
  OUTDATED = 'outdated'
}

export type ReviewDecisionInfo = {
  name: IconName
  color?: Color
  size?: number
  icon: IconName
  iconProps?: { color?: Color }
  message: string
}

export const generateReviewDecisionInfo = (
  reviewDecision: EnumPullReqReviewDecision | PullReqReviewDecision.OUTDATED
): ReviewDecisionInfo => {
  let info: ReviewDecisionInfo

  switch (reviewDecision) {
    case PullReqReviewDecision.CHANGEREQ:
      info = {
        name: 'error-transparent-no-outline',
        color: Color.RED_700,
        size: 18,
        icon: 'error-transparent-no-outline',
        iconProps: { color: Color.RED_700 },
        message: 'requested changes'
      }
      break
    case PullReqReviewDecision.APPROVED:
      info = {
        name: 'tick-circle',
        color: Color.GREEN_700,
        size: 16,
        icon: 'tick-circle',
        iconProps: { color: Color.GREEN_700 },
        message: 'approved changes'
      }
      break
    case PullReqReviewDecision.PENDING:
      info = {
        name: 'waiting',
        color: Color.GREY_700,
        size: 16,
        icon: 'waiting',
        iconProps: { color: Color.GREY_700 },
        message: 'pending review'
      }
      break
    case PullReqReviewDecision.OUTDATED:
      info = {
        name: 'dot',
        color: Color.GREY_100,
        size: 16,
        icon: 'dot',
        message: 'outdated approval'
      }
      break
    default:
      info = {
        name: 'dot',
        color: Color.GREY_100,
        size: 16,
        icon: 'dot',
        message: 'status'
      }
  }

  return info
}

export const checkEntries = (
  getString: UseStringsReturn['getString'],
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  changeReqArr: any[], // eslint-disable-next-line @typescript-eslint/no-explicit-any
  waitingEntriesArr: any[], // eslint-disable-next-line @typescript-eslint/no-explicit-any
  approvalEntriesArr: any[],
  totalCodeOwners: number
): { borderColor: string; message: string; overallStatus: ExecutionState } => {
  if (!isEmpty(changeReqArr)) {
    return {
      borderColor: 'red800',
      overallStatus: ExecutionState.FAILURE,
      message: stringSubstitute(getString('codeOwner.changesRequested'), { count: changeReqArr.length }) as string
    }
  } else if (!isEmpty(waitingEntriesArr)) {
    return {
      borderColor: 'orange800',
      message: stringSubstitute(getString('codeOwner.waitToApprove'), { count: waitingEntriesArr.length }) as string,
      overallStatus: ExecutionState.PENDING
    }
  }
  return {
    borderColor: 'green800',
    message: stringSubstitute(getString('codeOwner.approvalCompleted'), {
      count: approvalEntriesArr.length || '0',
      total: totalCodeOwners
    }) as string,
    overallStatus: ExecutionState.SUCCESS
  }
}

export const findWaitingDecisions = (
  pullReqMetadata: TypesPullReq,
  reqCodeOwnerLatestApproval: boolean,
  entries?: TypesCodeOwnerEvaluationEntry[] | null
) => {
  if (!entries) {
    return []
  }

  return entries.filter((entry: TypesCodeOwnerEvaluationEntry) => {
    const uniqueEvaluations = getCombinedEvaluations(entry)

    const hasNoReview = uniqueEvaluations?.every((evaluation: TypesOwnerEvaluation) =>
      isEmpty(evaluation.review_decision)
    )
    // add entry if no review found from codeowners
    if (hasNoReview) return true

    // add entry to waiting decision array if approved changes are outdated or no approvals are found for the given entry
    const hasApprovedDecision = uniqueEvaluations?.some(
      evaluation =>
        evaluation.review_decision === PullReqReviewDecision.APPROVED &&
        (reqCodeOwnerLatestApproval ? evaluation.review_sha === pullReqMetadata?.source_sha : true)
    )
    return !hasApprovedDecision
  })
}

// find code owner request decision from given entries
export function getCombinedEvaluations(entry: TypesCodeOwnerEvaluationEntry): TypesOwnerEvaluation[] {
  const ugUserEvaluations = (entry?.user_group_owner_evaluations || []).flatMap(ug => ug.evaluations || [])
  const evaluations = [...(entry?.owner_evaluations || []), ...ugUserEvaluations]

  const seen = new Set<number>()
  return evaluations.filter(ev => {
    const uniqueKey = ev?.owner?.id ?? -1
    if (seen.has(uniqueKey)) {
      return false
    }
    seen.add(uniqueKey)
    return true
  })
}

export const findReviewDecisions = (entries: TypesCodeOwnerEvaluationEntry[] | null | undefined, decision: string) => {
  if (!entries) {
    return []
  }

  return (
    entries
      ?.map((entry: TypesCodeOwnerEvaluationEntry) => {
        const evaluations =
          getCombinedEvaluations(entry).filter((evaluation: any) => evaluation?.review_decision === decision) || []

        if (!isEmpty(evaluations)) {
          return { ...entry, owner_evaluations: evaluations }
        }
      }) // eslint-disable-next-line @typescript-eslint/no-explicit-any
      .filter((entry: any) => entry !== null && entry !== undefined) || []
  ) // Filter out the null entries
}

export const processReviewDecision = (
  reviewDecision: EnumPullReqReviewDecision,
  reviewedSHA?: string,
  sourceSHA?: string
) =>
  reviewDecision === PullReqReviewDecision.APPROVED && reviewedSHA !== sourceSHA
    ? PullReqReviewDecision.OUTDATED
    : reviewDecision

export const getUnifiedDefaultReviewersState = (info: TypesDefaultReviewerApprovalsResponse[]) => {
  const defaultReviewState = {
    defReviewerApprovalRequiredByRule: false,
    defReviewerLatestApprovalRequiredByRule: false,
    defReviewerApprovedLatestChanges: true,
    defReviewerApprovedChanges: true
  }

  info?.forEach(item => {
    if (item?.minimum_required_count !== undefined && item.minimum_required_count > 0) {
      defaultReviewState.defReviewerApprovalRequiredByRule = true
      if (item.current_count !== undefined && item.current_count < item.minimum_required_count) {
        defaultReviewState.defReviewerApprovedChanges = false
      }
    }
    if (item?.minimum_required_count_latest !== undefined && item.minimum_required_count_latest > 0) {
      defaultReviewState.defReviewerLatestApprovalRequiredByRule = true
      if (item.current_count !== undefined && item.current_count < item.minimum_required_count_latest) {
        defaultReviewState.defReviewerApprovedLatestChanges = false
      }
    }
  })

  return defaultReviewState
}

export function getActivePullReqPageSection(): PullRequestSection | undefined {
  return (document.querySelector('[data-page-section]') as HTMLElement)?.dataset?.pageSection as PullRequestSection
}

export function extractSpecificViolations(violationsData: any, rule: string) {
  const specificViolations = violationsData?.data?.rule_violations.flatMap((violation: { violations: any[] }) =>
    violation.violations.filter(v => v.code === rule)
  )
  return specificViolations
}

export enum PullReqCustomEvent {
  REFETCH_DIFF = 'PullReqCustomEvent_REFETCH',
  REFETCH_ACTIVITIES = 'PullReqCustomEvent_REFETCH_ACTIVITIES'
}
export enum PanelSectionOutletPosition {
  CHANGES = 'changes',
  COMMENTS = 'comments',
  CHECKS = 'checks',
  MERGEABILITY = 'mergeability',
  BRANCH_ACTIONS = 'branchActions',
  REBASE_SOURCE_BRANCH = 'rebaseSourceBranch'
}

export const getMergeOptions = (getString: UseStringsReturn['getString'], mergeable: boolean): PRMergeOption[] => [
  {
    method: MergeStrategy.SQUASH,
    title: getString('pr.mergeOptions.squashAndMerge'),
    desc: getString('pr.mergeOptions.squashAndMergeDesc'),
    disabled: mergeable === false,
    label: getString('pr.mergeOptions.squashAndMerge'),
    value: MergeStrategy.SQUASH
  },
  {
    method: MergeStrategy.MERGE,
    title: getString('pr.mergeOptions.createMergeCommit'),
    desc: getString('pr.mergeOptions.createMergeCommitDesc'),
    disabled: mergeable === false,
    label: getString('pr.mergeOptions.createMergeCommit'),
    value: MergeStrategy.MERGE
  },
  {
    method: MergeStrategy.REBASE,
    title: getString('pr.mergeOptions.rebaseAndMerge'),
    desc: getString('pr.mergeOptions.rebaseAndMergeDesc'),
    disabled: mergeable === false,
    label: getString('pr.mergeOptions.rebaseAndMerge'),
    value: MergeStrategy.REBASE
  },
  {
    method: MergeStrategy.FAST_FORWARD,
    title: getString('pr.mergeOptions.fastForwardMerge'),
    desc: getString('pr.mergeOptions.fastForwardMergeDesc'),
    disabled: mergeable === false,
    label: getString('pr.mergeOptions.fastForwardMerge'),
    value: MergeStrategy.FAST_FORWARD
  },
  {
    method: 'close',
    title: getString('pr.mergeOptions.close'),
    desc: getString('pr.mergeOptions.closeDesc'),
    label: getString('pr.mergeOptions.close'),
    value: 'close'
  }
]

export const updateReviewDecisionPrincipal = (reviewers: TypesPullReqReviewer[], principals: TypesPrincipalInfo[]) => {
  const reviewDecisionMap: {
    [x: number]: { sha: string; review_decision: EnumPullReqReviewDecision } | null
  } = reviewers?.reduce((acc, rev) => {
    if (rev.reviewer?.id) {
      acc[rev.reviewer.id] = {
        sha: rev.sha ?? '',
        review_decision: rev.review_decision ?? 'pending'
      }
    }
    return acc
  }, {} as { [x: number]: { sha: string; review_decision: EnumPullReqReviewDecision } | null })

  return principals?.map(principal => {
    if (principal?.id) {
      return {
        ...principal,
        review_decision: reviewDecisionMap[principal.id] ? reviewDecisionMap[principal.id]?.review_decision : 'pending',
        review_sha: reviewDecisionMap[principal.id]?.sha
      }
    }
    return principal
  })
}

export const defaultReviewerResponseWithDecision = (
  defaultRevApprovalResponse: TypesDefaultReviewerApprovalsResponse[],
  reviewers: TypesPullReqReviewer[]
) => {
  return defaultRevApprovalResponse?.map(res => {
    return {
      ...res,
      principals:
        reviewers && res.principals ? updateReviewDecisionPrincipal(reviewers, res.principals) : res.principals
    }
  })
}

export type ActivityLabel = {
  label: string
  label_color: ColorName
  label_scope: number
  value: string
  value_color: ColorName
}

export const extractInfoFromRuleViolationArr = (ruleViolationArr: TypesRuleViolations[]) => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const tempArray: any[] = ruleViolationArr?.flatMap(
    (item: { violations?: TypesViolation[] | null }) => item?.violations?.map(violation => violation.message) ?? []
  )
  const uniqueViolations = new Set(tempArray)
  const violationArr = [...uniqueViolations].map(violation => ({ violation: violation }))

  const checkIfBypassNotAllowed = ruleViolationArr.some(ruleViolation => ruleViolation.bypassed === false)

  return {
    uniqueViolations,
    checkIfBypassNotAllowed,
    violationArr
  }
}
