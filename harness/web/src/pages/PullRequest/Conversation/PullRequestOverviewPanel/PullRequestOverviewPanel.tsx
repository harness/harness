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
  DeletePullReqSourceBranchQueryParams,
  TypesCodeOwnerEvaluation,
  TypesListCommitResponse,
  TypesPullReq,
  TypesPullReqActivity,
  RepoRepositoryOutput,
  TypesRuleViolations,
  TypesBranchExtended,
  TypesDefaultReviewerApprovalsResponse,
  PullreqCombinedListResponse
} from 'services/code'
import {
  PRMergeOption,
  PanelSectionOutletPosition,
  extractInfoFromRuleViolationArr,
  extractSpecificViolations,
  getMergeOptions
} from 'pages/PullRequest/PullRequestUtils'
import { MergeCheckStatus } from 'utils/Utils'
import { MergeStrategy, PullRequestState, dryMerge } from 'utils/GitUtils'
import { useStrings } from 'framework/strings'
import type { PRChecksDecisionResult } from 'hooks/usePRChecksDecision'
import { useGetRepositoryMetadata } from 'hooks/useGetRepositoryMetadata'
import { UserPreference, useUserPreference } from 'hooks/useUserPreference'
import { PullRequestActionsBox } from '../PullRequestActionsBox/PullRequestActionsBox'
import PullRequestPanelSections from './PullRequestPanelSections'
import ChecksSection from './sections/ChecksSection'
import MergeSection from './sections/MergeSection'
import CommentsSection from './sections/CommentsSection'
import ChangesSection from './sections/ChangesSection'
import BranchActionsSection from './sections/BranchActionsSection'
import RebaseSourceSection from './sections/RebaseSourceSection'
import css from './PullRequestOverviewPanel.module.scss'

interface PullRequestOverviewPanelProps {
  repoMetadata: RepoRepositoryOutput
  pullReqMetadata: TypesPullReq
  onPRStateChanged: () => void
  refetchReviewers: () => void
  prChecksDecisionResult: PRChecksDecisionResult
  codeOwners: TypesCodeOwnerEvaluation | null
  combinedReviewers: PullreqCombinedListResponse | null
  setActivityFilter: (val: SelectOption) => void
  loadingReviewers: boolean
  refetchActivities: () => void
  refetchCodeOwners: () => void
  refetchPullReq: () => void
  activities?: TypesPullReqActivity[]
  pullReqCommits?: TypesListCommitResponse
}

const PullRequestOverviewPanel = (props: PullRequestOverviewPanelProps) => {
  const {
    setActivityFilter,
    codeOwners,
    repoMetadata,
    pullReqMetadata,
    onPRStateChanged,
    refetchReviewers,
    combinedReviewers,
    loadingReviewers,
    refetchActivities,
    refetchCodeOwners,
    activities,
    pullReqCommits,
    refetchPullReq
  } = props
  const { getString } = useStrings()
  const { showError } = useToaster()

  const isMounted = useIsMounted()
  const isMerged = pullReqMetadata.state === PullRequestState.MERGED
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
  const [defaultReviewersInfoSet, setDefaultReviewersInfoSet] = useState<TypesDefaultReviewerApprovalsResponse[]>([])
  const [minReqLatestApproval, setMinReqLatestApproval] = useState(0)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [resolvedCommentArr, setResolvedCommentArr] = useState<any>()
  const [mergeBlockedRule, setMergeBlockedRule] = useState<boolean>(false)
  const [PRStateLoading, setPRStateLoading] = useState(isClosed ? false : true)
  const { pullRequestSection } = useGetRepositoryMetadata()
  const mergeable = useMemo(() => pullReqMetadata.merge_check_status === MergeCheckStatus.MERGEABLE, [pullReqMetadata])
  const mergeOptions = useMemo(() => getMergeOptions(getString, mergeable), [mergeable])
  const [allowedStrats, setAllowedStrats] = useState<string[]>([
    mergeOptions[0].method,
    mergeOptions[1].method,
    mergeOptions[2].method,
    mergeOptions[3].method
  ])
  const [showDeleteBranchButton, setShowDeleteBranchButton] = useState(false)
  const [showRestoreBranchButton, setShowRestoreBranchButton] = useState(false)
  const [isSourceBranchDeleted, setIsSourceBranchDeleted] = useState(false)

  const {
    data: sourceBranch,
    error,
    refetch: refetchBranch
  } = useGet<TypesBranchExtended>({
    path: `/api/v1/repos/${repoMetadata?.path}/+/branches/${pullReqMetadata?.source_branch}`,
    queryParams: {
      repo_ref: repoMetadata.path || '',
      branch_name: pullReqMetadata.source_branch || ''
    },
    lazy: true
  })
  const { mutate: deleteBranch } = useMutate({
    verb: 'DELETE',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}/branch`,
    queryParams: { bypass_rules: true, dry_run_rules: true } as DeletePullReqSourceBranchQueryParams
  })
  const { mutate: restoreBranch } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}/branch`
  })
  const { mutate: mergePR, loading: mergeLoading } = useMutate({
    verb: 'POST',
    path: `/api/v1/repos/${repoMetadata.path}/+/pullreq/${pullReqMetadata.number}/merge`
  })

  // Flags to optimize rendering
  const internalFlags = useRef({ dryRun: false })

  useEffect(() => {
    if (isMerged || isClosed) {
      refetchBranch()
    }
  }, [isMerged, isClosed])

  useEffect(() => {
    if (error && error.status === 404) {
      setIsSourceBranchDeleted(true)
      restoreBranch({
        bypass_rules: true,
        dry_run_rules: true
      })
        .then(res => {
          if (res?.rule_violations) {
            const { checkIfBypassNotAllowed } = extractInfoFromRuleViolationArr(res.rule_violations)
            if (!checkIfBypassNotAllowed) {
              setShowRestoreBranchButton(true)
            } else {
              setShowRestoreBranchButton(false)
            }
          } else {
            setShowRestoreBranchButton(true)
          }
        })
        .catch(err => console.error('Dry run failed while trying to restore branch', err)) // eslint-disable-line no-console
    }
  }, [error])

  useEffect(() => {
    if (sourceBranch?.sha === pullReqMetadata?.source_sha) {
      deleteBranch({})
        .then(res => {
          if (res?.rule_violations) {
            const { checkIfBypassNotAllowed } = extractInfoFromRuleViolationArr(res.rule_violations)
            if (!checkIfBypassNotAllowed) {
              setShowDeleteBranchButton(true)
            } else {
              setShowDeleteBranchButton(false)
            }
          } else {
            setShowDeleteBranchButton(true)
          }
        })
        .catch(err => console.error('Dry run failed while trying to delete branch', err)) // eslint-disable-line no-console
    }
  }, [sourceBranch, pullReqMetadata?.source_sha])

  useEffect(() => {
    if (ruleViolationArr) {
      const requireResCommentRule = extractSpecificViolations(ruleViolationArr, 'pullreq.comments.require_resolve_all')
      const mergeBlockedViaRule = extractSpecificViolations(ruleViolationArr, 'pullreq.merge.blocked')
      if (requireResCommentRule) {
        setResolvedCommentArr(requireResCommentRule[0])
      }
      setMergeBlockedRule(mergeBlockedViaRule.length > 0)
    } else {
      setMergeBlockedRule(false)
    }
  }, [ruleViolationArr, pullReqMetadata, repoMetadata, ruleViolation])

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
      refetchPullReq,
      setRequiresCommentApproval,
      setAtLeastOneReviewerRule,
      setReqCodeOwnerApproval,
      setMinApproval,
      setReqCodeOwnerLatestApproval,
      setMinReqLatestApproval,
      setPRStateLoading,
      setDefaultReviewersInfoSet
    ) // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [unchecked, pullReqMetadata?.source_sha, activities])

  const rebasePossible = useMemo(
    () => pullReqMetadata.merge_target_sha !== pullReqMetadata.merge_base_sha && !pullReqMetadata.merged,
    [pullReqMetadata]
  )

  const [mergeOption, setMergeOption] = useUserPreference<PRMergeOption>(
    UserPreference.PULL_REQUEST_MERGE_STRATEGY,
    mergeOptions[0],
    option => option.method !== 'close'
  )

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
          refetchPullReq={refetchPullReq}
          refetchActivities={refetchActivities}
          restoreBranch={restoreBranch}
          refetchBranch={refetchBranch}
          deleteBranch={deleteBranch}
          showRestoreBranchButton={showRestoreBranchButton}
          showDeleteBranchButton={showDeleteBranchButton}
          setShowDeleteBranchButton={setShowDeleteBranchButton}
          setShowRestoreBranchButton={setShowRestoreBranchButton}
          isSourceBranchDeleted={isSourceBranchDeleted}
          mergeOption={mergeOption}
          setMergeOption={setMergeOption}
          rebasePossible={rebasePossible}
        />
        {!isClosed ? (
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
                    combinedReviewers={combinedReviewers}
                    minReqLatestApproval={minReqLatestApproval}
                    reqCodeOwnerLatestApproval={reqCodeOwnerLatestApproval}
                    refetchCodeOwners={refetchCodeOwners}
                    mergeBlockedRule={mergeBlockedRule}
                    defaultReviewersInfoSet={defaultReviewersInfoSet}
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
              ),
              [PanelSectionOutletPosition.REBASE_SOURCE_BRANCH]: rebasePossible &&
                !mergeLoading &&
                !conflictingFiles?.length &&
                mergeOption.method === MergeStrategy.FAST_FORWARD && (
                  <RebaseSourceSection
                    pullReqMetadata={pullReqMetadata}
                    repoMetadata={repoMetadata}
                    refetchActivities={refetchActivities}
                  />
                )
            }}
          />
        ) : (
          <PullRequestPanelSections
            outlets={{
              [PanelSectionOutletPosition.BRANCH_ACTIONS]: (showDeleteBranchButton || showRestoreBranchButton) && (
                <BranchActionsSection
                  sourceBranch={sourceBranch?.name || pullReqMetadata.source_branch || ''}
                  restoreBranch={restoreBranch}
                  refetchBranch={refetchBranch}
                  refetchActivities={refetchActivities}
                  deleteBranch={deleteBranch}
                  showDeleteBranchButton={showDeleteBranchButton}
                  setShowRestoreBranchButton={setShowRestoreBranchButton}
                  setShowDeleteBranchButton={setShowDeleteBranchButton}
                  setIsSourceBranchDeleted={setIsSourceBranchDeleted}
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
