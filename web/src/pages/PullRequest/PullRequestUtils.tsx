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

import type { SelectOption } from '@harnessio/uicore'
import type { UseStringsReturn } from 'framework/strings'
import type { CommentItem } from 'components/CommentBox/CommentBox'
import type { PullRequestSection } from 'utils/Utils'
import { MergeStrategy } from 'utils/GitUtils'
import type {
  EnumMergeMethod,
  EnumPullReqReviewDecision,
  TypesCodeOwnerEvaluationEntry,
  TypesOwnerEvaluation,
  TypesPullReq,
  TypesPullReqActivity
} from 'services/code'

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
  REVIEWER_DELETE = 'reviewer-delete'
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

export const findWaitingDecisions = (
  pullReqMetadata: TypesPullReq,
  reqCodeOwnerLatestApproval: boolean,
  entries?: TypesCodeOwnerEvaluationEntry[] | null
) => {
  if (entries === null || entries === undefined) {
    return []
  } else {
    return entries.filter((entry: TypesCodeOwnerEvaluationEntry) => {
      const hasNoReview = entry?.owner_evaluations?.every(
        (evaluation: TypesOwnerEvaluation | { review_decision: string }) => evaluation.review_decision === ''
      )

      // add entry if no review found from codeowners
      if (hasNoReview) return true
      // add entry to waiting decision array if approved changes are outdated or no approvals are found for the given entry
      const hasApprovedDecision = entry?.owner_evaluations?.some(
        evaluation =>
          evaluation.review_decision === PullReqReviewDecision.APPROVED &&
          (reqCodeOwnerLatestApproval ? evaluation.review_sha === pullReqMetadata.source_sha : true)
      )
      return !hasApprovedDecision
    })
  }
}

export const processReviewDecision = (
  review_decision: EnumPullReqReviewDecision,
  reviewedSHA?: string,
  sourceSHA?: string
) =>
  review_decision === PullReqReviewDecision.APPROVED && reviewedSHA !== sourceSHA
    ? PullReqReviewDecision.OUTDATED
    : review_decision

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
