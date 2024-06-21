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
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Container, Layout, SelectOption, useIsMounted, useToaster } from '@harnessio/uicore'
import cx from 'classnames'
import { useGet, useMutate } from 'restful-react'
import { Render } from 'react-jsx-match'
import type {
  TypesCodeOwnerEvaluation,
  TypesListCommitResponse,
  TypesPullReq,
  TypesPullReqActivity,
  TypesPullReqReviewer,
  TypesRepository,
  TypesRuleViolations
} from 'services/code'
import { PanelSectionOutletPosition } from 'pages/PullRequest/PullRequestUtils'
import { MergeCheckStatus, PRMergeOption } from 'utils/Utils'
import { PullRequestState, dryMerge } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import type { PRChecksDecisionResult } from 'hooks/usePRChecksDecision'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { PullRequestActionsBox } from '../PullRequestActionsBox/PullRequestActionsBox'
import PullRequestPanelSections from './PullRequestPanelSections'
import ChecksSection from './sections/ChecksSection'
import MergeSection from './sections/MergeSection'
import CommentsSection from './sections/CommentsSection'
import ChangesSection from './sections/ChangesSection'
import css from './PullRequestOverviewPanel.module.scss'
interface PullRequestOverviewPanelProps {
  repoMetadata: TypesRepository
  pullReqMetadata: TypesPullReq
  onPRStateChanged: () => void
  refetchReviewers: () => void
  prChecksDecisionResult: PRChecksDecisionResult
  codeOwners: TypesCodeOwnerEvaluation | null
  reviewers: TypesPullReqReviewer[] | null
  setActivityFilter: (val: SelectOption) => void
  loadingReviewers: boolean
  refetchCodeOwners: () => void
  activities: TypesPullReqActivity[] | undefined
  pullReqCommits: TypesListCommitResponse | undefined
}

const PullRequestOverviewPanel = (props: PullRequestOverviewPanelProps) => {
  const {
    setActivityFilter,
    codeOwners,
    repoMetadata,
    pullReqMetadata,
    onPRStateChanged,
    refetchReviewers,
    reviewers,
    loadingReviewers,
    refetchCodeOwners,
    activities,
    pullReqCommits
  } = props
  const { getString } = useStrings()
  const { showError } = useToaster()

  const isMounted = useIsMounted()
  const isClosed = pullReqMetadata.state === PullRequestState.CLOSED

  const unchecked = useMemo(
    () => pullReqMetadata.merge_check_status === MergeCheckStatus.UNCHECKED && !isClosed,
    [pullReqMetadata, isClosed]
  )
  const [conflictingFiles, setConflictingFiles] = useState<string[]>()
  const [ruleViolation, setRuleViolation] = useState(false)
  const [ruleViolationArr, setRuleViolationArr] = useState<{ data: { rule_violations: TypesRuleViolations[] } }>()
  const [requiresCommentApproval, setRequiresCommentApproval] = useState(false)
  const [atLeastOneReviewerRule, setAtLeastOneReviewerRule] = useState(false)
  const [reqCodeOwnerApproval, setReqCodeOwnerApproval] = useState(false)
  const [minApproval, setMinApproval] = useState(0)
  const [reqCodeOwnerLatestApproval, setReqCodeOwnerLatestApproval] = useState(false)
  const [minReqLatestApproval, setMinReqLatestApproval] = useState(0)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [resolvedCommentArr, setResolvedCommentArr] = useState<any>()
  const [PRStateLoading, setPRStateLoading] = useState(isClosed ? false : true)
  const { pullRequestSection } = useGetRepositoryMetadata()
  const mergeable = useMemo(() => pullReqMetadata.merge_check_status === MergeCheckStatus.MERGEABLE, [pullReqMetadata])
  const mergeOptions: PRMergeOption[] = [
    {
      method: 'squash',
      title: getString('pr.mergeOptions.squashAndMerge'),
      desc: getString('pr.mergeOptions.squashAndMergeDesc'),
      disabled: mergeable === false,
      label: getString('pr.mergeOptions.squashAndMerge'),
      value: 'squash'
    },
    {
      method: 'merge',
      title: getString('pr.mergeOptions.createMergeCommit'),
      desc: getString('pr.mergeOptions.createMergeCommitDesc'),
      disabled: mergeable === false,
      label: getString('pr.mergeOptions.createMergeCommit'),
      value: 'merge'
    },
    {
      method: 'rebase',
      title: getString('pr.mergeOptions.rebaseAndMerge'),
      desc: getString('pr.mergeOptions.rebaseAndMergeDesc'),
      disabled: mergeable === false,
      label: getString('pr.mergeOptions.rebaseAndMerge'),
      value: 'rebase'
    },

    {
      method: 'close',
      title: getString('pr.mergeOptions.close'),
      desc: getString('pr.mergeOptions.closeDesc'),
      label: getString('pr.mergeOptions.close'),
      value: 'close'
    }
  ]
  const [allowedStrats, setAllowedStrats] = useState<string[]>([
    mergeOptions[0].method,
    mergeOptions[1].method,
    mergeOptions[2].method,
    mergeOptions[3].method
  ])
  const { mutate: mergePR } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}/merge`
  })
  const { data: data } = useGet({
    path: `/api/v1/repos/${repoMetadata.path}/+/rules`
  })
  // Flags to optimize rendering
  const internalFlags = useRef({ dryRun: false })
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  function extractSpecificViolations(violationsData: any, rule: string) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const specificViolations = violationsData?.data?.rule_violations.flatMap((violation: { violations: any[] }) =>
      violation.violations.filter(v => v.code === rule)
    )
    return specificViolations
  }

  useEffect(() => {
    if (ruleViolationArr) {
      const requireResCommentRule = extractSpecificViolations(ruleViolationArr, 'pullreq.comments.require_resolve_all')
      if (requireResCommentRule) {
        setResolvedCommentArr(requireResCommentRule[0])
      }
    }
  }, [ruleViolationArr, pullReqMetadata, repoMetadata, data, ruleViolation])
  useEffect(() => {
    // recheck PR in case source SHA changed or PR was marked as unchecked
    // TODO: optimize call to handle all causes and avoid double calls by keeping track of SHA
    dryMerge(
      isMounted,
      isClosed,
      pullReqMetadata,
      internalFlags,
      mergePR,
      setRuleViolation,
      setRuleViolationArr,
      setAllowedStrats,
      pullRequestSection,
      showError,
      setConflictingFiles,
      setRequiresCommentApproval,
      setAtLeastOneReviewerRule,
      setReqCodeOwnerApproval,
      setMinApproval,
      setReqCodeOwnerLatestApproval,
      setMinReqLatestApproval,
      setPRStateLoading
    ) // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [unchecked, pullReqMetadata?.source_sha, activities])

  return (
    <Container margin={{ bottom: 'medium' }} className={css.mainContainer}>
      <Layout.Vertical>
        <PullRequestActionsBox
          conflictingFiles={conflictingFiles}
          setConflictingFiles={setConflictingFiles}
          repoMetadata={repoMetadata}
          pullReqMetadata={pullReqMetadata}
          onPRStateChanged={onPRStateChanged}
          refetchReviewers={refetchReviewers}
          allowedStrategy={allowedStrats}
          pullReqCommits={pullReqCommits}
          PRStateLoading={PRStateLoading || loadingReviewers}
        />
        {pullReqMetadata.state !== PullRequestState.CLOSED && (
          <PullRequestPanelSections
            outlets={{
              [PanelSectionOutletPosition.CHANGES]: !pullReqMetadata.merged && (
                <Render when={!loadingReviewers}>
                  <ChangesSection
                    loadingReviewers={loadingReviewers}
                    pullReqMetadata={pullReqMetadata}
                    repoMetadata={repoMetadata}
                    refetchReviewers={refetchReviewers}
                    codeOwners={codeOwners}
                    atLeastOneReviewerRule={atLeastOneReviewerRule}
                    reqCodeOwnerApproval={reqCodeOwnerApproval}
                    minApproval={minApproval}
                    reviewers={reviewers}
                    minReqLatestApproval={minReqLatestApproval}
                    reqCodeOwnerLatestApproval={reqCodeOwnerLatestApproval}
                    refetchCodeOwners={refetchCodeOwners}
                  />
                </Render>
              ),
              [PanelSectionOutletPosition.COMMENTS]: (!!resolvedCommentArr || requiresCommentApproval) &&
                !pullReqMetadata.merged && (
                  <Container className={cx(css.sectionContainer, css.borderContainer)}>
                    <CommentsSection
                      pullReqMetadata={pullReqMetadata}
                      repoMetadata={repoMetadata}
                      resolvedCommentArr={resolvedCommentArr}
                      requiresCommentApproval={requiresCommentApproval}
                      setActivityFilter={setActivityFilter}
                    />
                  </Container>
                ),
              [PanelSectionOutletPosition.CHECKS]: (
                <ChecksSection pullReqMetadata={pullReqMetadata} repoMetadata={repoMetadata} />
              ),
              [PanelSectionOutletPosition.MERGEABILITY]: !pullReqMetadata.merged && (
                <MergeSection
                  pullReqMetadata={pullReqMetadata}
                  unchecked={unchecked}
                  mergeable={mergeable}
                  conflictingFiles={conflictingFiles}
                />
              )
            }}
          />
        )}
      </Layout.Vertical>
    </Container>
  )
}

export default PullRequestOverviewPanel
