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
import { CommentType } from 'components/DiffViewer/DiffViewerUtils'
import type { PullRequestSection } from 'utils/Utils'
import { MergeStrategy } from 'utils/GitUtils'
import type { EnumMergeMethod, EnumPullReqReviewDecision, TypesPullReqActivity } from 'services/code'

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
  approved = 'approved',
  changeReq = 'changereq',
  pending = 'pending',
  outdated = 'outdated'
}

export const processReviewDecision = (
  review_decision: EnumPullReqReviewDecision,
  reviewedSHA?: string,
  sourceSHA?: string
) =>
  review_decision === PullReqReviewDecision.approved && reviewedSHA !== sourceSHA
    ? PullReqReviewDecision.outdated
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
  BRANCH_ACTIONS = 'branchActions'
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
    method: 'close',
    title: getString('pr.mergeOptions.close'),
    desc: getString('pr.mergeOptions.closeDesc'),
    label: getString('pr.mergeOptions.close'),
    value: 'close'
  }
]
