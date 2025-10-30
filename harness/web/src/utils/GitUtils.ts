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

// Code copied from https://github.com/vweevers/is-git-ref-name-valid and
// https://github.com/vweevers/is-git-branch-name-valid (MIT, © Vincent Weevers)
// Last updated for git 2.29.0.

import type { IconName } from '@harnessio/icons'
import type { MutateRequestOptions } from 'restful-react/dist/Mutate'
import type {
  EnumWebhookTrigger,
  OpenapiContentInfo,
  OpenapiDirContent,
  OpenapiGetContentOutput,
  TypesCommit,
  TypesPullReq,
  RepoRepositoryOutput,
  TypesRuleViolations,
  TypesDefaultReviewerApprovalsResponse
} from 'services/code'
import { getConfig } from 'services/config'
import { PullRequestSection, getErrorMessage } from './Utils'

export interface GitInfoProps {
  repoMetadata: RepoRepositoryOutput
  gitRef: string
  resourcePath: string
  resourceContent: OpenapiGetContentOutput
  commitRef: string
  commits: TypesCommit[]
  pullReqMetadata: TypesPullReq
}
export interface RepoFormData {
  name: string
  description: string
  license: string
  defaultBranch: string
  gitignore: string
  addReadme: boolean
  isPublic: RepoVisibility
}
export interface ImportFormData {
  gitProvider: GitProviders
  hostUrl: string
  org: string
  project: string
  repo: string
  username: string
  password: string
  name: string
  description: string
  importPipelineLabel: boolean
}

export interface ExportFormData {
  accountId: string
  token: string
  organization: string
  name: string
}

export interface ExportFormDataExtended extends ExportFormData {
  repoCount: number
}

export interface ImportSpaceFormData {
  gitProvider: GitProviders
  username: string
  password: string
  name: string
  description: string
  organization: string
  project: string
  host: string
  importPipelineLabel: boolean
}

export interface RepositorySummaryData {
  default_branch_commit_count: number
  branch_count: number
  tag_count: number
  pull_req_summary: {
    open_count: number
    closed_count: number
    merged_count: number
  }
}

export enum RepoVisibility {
  PUBLIC = 'public',
  PRIVATE = 'private'
}

export enum RepoState {
  ARCHIVED = 'archived',
  UNARCHIVED = 'unarchived'
}

export enum RepoCreationType {
  IMPORT = 'import',
  CREATE = 'create',
  IMPORT_MULTIPLE = 'import_multiple'
}

export enum SpaceCreationType {
  IMPORT = 'import',
  CREATE = 'create'
}

export enum GitContentType {
  FILE = 'file',
  DIR = 'dir',
  SYMLINK = 'symlink',
  SUBMODULE = 'submodule'
}
export enum SettingsTab {
  WEBHOOKS = 'webhook',
  GENERAL = '/',
  PROTECTION_RULES = 'rules',
  SECURITY = 'security',
  LABELS = 'labels'
}

export enum SpacePRTabs {
  CREATED = 'created',
  REVIEW_REQUESTED = 'review_requested'
}

export enum DashboardFilter {
  ALL = 'all',
  CREATED = 'created',
  REVIEW_REQUESTED = 'review_requested'
}

export enum WebhookTabs {
  DETAILS = 'details',
  EXECUTIONS = 'executions'
}

export enum ExecutionTabs {
  PAYLOAD = 'Payload',
  SERVER_RESPONSE = 'Server Response'
}

export enum VulnerabilityScanningType {
  DETECT = 'detect',
  BLOCK = 'block',
  DISABLED = 'disabled'
}

export enum GitBranchType {
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  YOURS = 'yours',
  ALL = 'all'
}

export enum GitRefType {
  BRANCH = 'branch',
  TAG = 'tag'
}

export enum SettingTypeMode {
  EDIT = 'edit',
  NEW = 'new'
}

export enum ProtectionRulesType {
  BRANCH = 'branch',
  TAG = 'tag',
  PUSH = 'push'
}

export enum RulesTargetType {
  INCLUDE = 'include',
  EXCLUDE = 'exclude'
}

export enum GitCommitAction {
  DELETE = 'DELETE',
  CREATE = 'CREATE',
  UPDATE = 'UPDATE',
  MOVE = 'MOVE'
}

export enum PullRequestState {
  OPEN = 'open',
  MERGED = 'merged',
  CLOSED = 'closed'
}

export enum PullRequestFilterOption {
  OPEN = 'open',
  MERGED = 'merged',
  CLOSED = 'closed',
  DRAFT = 'draft',
  YOURS = 'yours',
  ALL = 'all'
}

export enum GitProviders {
  GITHUB = 'GitHub',
  GITHUB_ENTERPRISE = 'GitHub Enterprise',
  GITLAB = 'GitLab',
  GITLAB_SELF_HOSTED = 'GitLab Self-Hosted',
  BITBUCKET = 'Bitbucket',
  BITBUCKET_SERVER = 'Bitbucket Server',
  GITEA = 'Gitea',
  GOGS = 'Gogs',
  AZURE = 'Azure DevOps'
}

export enum ConvertPipelineLabel {
  CONVERT = 'convert',
  IGNORE = 'ignore'
}

export const PullRequestReviewFilterOption = {
  PENDING: 'pending',
  APPROVED: 'approved',
  CHANGES_REQUESTED: 'changereq'
}

export enum MergeStrategy {
  MERGE = 'merge',
  SQUASH = 'squash',
  REBASE = 'rebase',
  FAST_FORWARD = 'fast-forward'
}

export enum MergeMethodDisplay {
  MERGED = 'merged',
  SQUASHED = 'squashed',
  REBASED = 'rebased',
  FAST_FORWARDED = 'fast-forwarded'
}

export const CodeIcon: Record<string, IconName> = {
  Logo: 'code',
  PullRequest: 'git-pull',
  Merged: 'code-merged',
  Draft: 'code-draft',
  Rejected: 'code-rejected',
  PullRequestRejected: 'main-close',
  Add: 'plus',
  BranchSmall: 'code-branch-small',
  Branch: 'code-branch',
  Tag: 'main-tags',
  Clone: 'code-clone',
  Close: 'code-close',
  CommitLight: 'code-commit-light',
  CommitSmall: 'code-commit-small',
  Commit: 'code-commit',
  Copy: 'code-copy',
  Delete: 'code-delete',
  Edit: 'code-edit',
  FileLight: 'code-file-light',
  File: 'code-file',
  Folder: 'code-folder',
  History: 'code-history',
  Info: 'code-info',
  More: 'code-more',
  Repo: 'code-repo',
  Settings: 'code-settings',
  Webhook: 'code-webhook',
  InputSpinner: 'steps-spinner',
  InputSearch: 'search',
  Chat: 'code-chat',
  Checks: 'main-tick',
  ChecksSuccess: 'success-tick',
  CheckIcon: 'code-checks',
  Tick: 'tick',
  Blank: 'blank'
}

export const normalizeGitRef = (gitRef: string | undefined) => {
  if (isRefATag(gitRef)) {
    return gitRef
  } else if (isRefABranch(gitRef)) {
    return gitRef
  } else if (gitRef === '') {
    return ''
  } else if (gitRef && isGitRev(gitRef)) {
    return gitRef
  } else {
    return REFS_BRANCH_PREFIX + gitRef
  }
}

export const REFS_TAGS_PREFIX = 'refs/tags/'
export const REFS_BRANCH_PREFIX = 'refs/heads/'

export const FILE_VIEWED_OBSOLETE_SHA = 'ffffffffffffffffffffffffffffffffffffffff'

export function formatTriggers(triggers: EnumWebhookTrigger[]) {
  return triggers.map(trigger => {
    return trigger
      .split('_')
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ')
  })
}

export const handleUpload = (
  blob: File,
  setMarkdownContent: (data: string) => void,
  repoMetadata: RepoRepositoryOutput | undefined,
  showError: (message: React.ReactNode, timeout?: number | undefined, key?: string | undefined) => void,
  standalone: boolean,
  routingId?: string
) => {
  const reader = new FileReader()
  // Set up a function to be called when the load event is triggered
  reader.onload = async function () {
    const markdown = await uploadImage(reader.result, showError, repoMetadata, standalone, routingId)
    setMarkdownContent(markdown) // Set the markdown content
  }
  reader.readAsArrayBuffer(blob) // This will trigger the onload function when the reading is complete
}

export const uploadImage = async (
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  fileBlob: any,
  showError: (message: React.ReactNode, timeout?: number | undefined, key?: string | undefined) => void,
  repoMetadata: RepoRepositoryOutput | undefined,
  standalone: boolean,
  routingId?: string
) => {
  try {
    const response = await fetch(
      `${window.location.origin}${getConfig(
        `code/api/v1/repos/${repoMetadata?.path}/+/uploads${standalone || !routingId ? `` : `?routingId=${routingId}`}`
      )}`,
      {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'content-type': 'application/octet-stream'
        },
        body: fileBlob,
        redirect: 'follow'
      }
    )
    const result = await response.json()
    if (!response.ok && result) {
      showError(getErrorMessage(result))
      return ''
    }
    const filePath = result.file_path
    return `${window.location.origin}${getConfig(
      `code/api/v1/repos/${repoMetadata?.path}/+/uploads/${filePath}${
        standalone || !routingId ? `` : `?routingId=${routingId}`
      }`
    )}`
  } catch (exception) {
    showError(getErrorMessage(exception))
    return ''
  }
}

// eslint-disable-next-line no-control-regex
const BAD_GIT_REF_REGREX = /(^|[/.])([/.]|$)|^@$|@{|[\x00-\x20\x7f~^:?*[\\]|\.lock(\/|$)/
const BAD_GIT_BRANCH_REGREX = /^(-|HEAD$)/

function isGitRefValid(name: string, onelevel: boolean): boolean {
  return !BAD_GIT_REF_REGREX.test(name) && (!!onelevel || name.includes('/'))
}

export function isGitBranchNameValid(name: string): boolean {
  return isGitRefValid(name, true) && !BAD_GIT_BRANCH_REGREX.test(name)
}

export const isDir = (content: Nullable<OpenapiGetContentOutput>): boolean => content?.type === GitContentType.DIR
export const isFile = (content: Nullable<OpenapiGetContentOutput>): boolean => content?.type === GitContentType.FILE
export const isSymlink = (content: Nullable<OpenapiGetContentOutput>): boolean =>
  content?.type === GitContentType.SYMLINK
export const isSubmodule = (content: Nullable<OpenapiGetContentOutput>): boolean =>
  content?.type === GitContentType.SUBMODULE

export const findReadmeInfo = (content: Nullable<OpenapiGetContentOutput>): OpenapiContentInfo | undefined =>
  (content?.content as OpenapiDirContent)?.entries?.find(
    entry => entry.type === GitContentType.FILE && /^readme(.md)?$/.test(entry?.name?.toLowerCase() || '')
  )

export const findMarkdownInfo = (content: Nullable<OpenapiGetContentOutput>): OpenapiContentInfo | undefined =>
  content?.type === GitContentType.FILE && /.md$/.test(content?.name?.toLowerCase() || '') ? content : undefined

export const isRefATag = (gitRef: string | undefined) => gitRef?.includes(REFS_TAGS_PREFIX) || false
export const isRefABranch = (gitRef: string | undefined) => gitRef?.includes(REFS_BRANCH_PREFIX) || false

/**
 * Make a diff refs string to use in URL.
 * @param targetGitRef target git ref (base ref).
 * @param sourceGitRef source git ref (compare ref).
 * @returns A concatenation string of `targetGitRef...sourceGitRef`.
 */
export const makeDiffRefs = (targetGitRef: string, sourceGitRef: string) => `${targetGitRef}...${sourceGitRef}`

/**
 * Split a diff refs string into targetRef and sourceRef.
 * @param diffRefs diff refs string.
 * @returns An object of { targetGitRef, sourceGitRef }
 */
export const diffRefsToRefs = (diffRefs: string) => {
  const parts = diffRefs.split('...')

  return {
    targetGitRef: parts[0] || '',
    sourceGitRef: parts[1] || ''
  }
}

export const decodeGitContent = (content = '') => {
  try {
    // Decode base64 content for text file
    return decodeURIComponent(escape(window.atob(content)))
  } catch (_exception) {
    try {
      // Return original base64 content for binary file
      return content
    } catch (exception) {
      console.error(exception) // eslint-disable-line no-console
    }
  }
  return ''
}

// Check if gitRef is a git commit hash (https://github.com/diegohaz/is-git-rev, MIT © Diego Haz)
export const isGitRev = (gitRef = ''): boolean => /^[0-9a-f]{7,40}$/i.test(gitRef)

export const getProviderTypeMapping = (provider: GitProviders): string => {
  switch (provider) {
    case GitProviders.BITBUCKET_SERVER:
      return 'stash'
    case GitProviders.GITHUB_ENTERPRISE:
      return 'github'
    case GitProviders.GITLAB_SELF_HOSTED:
      return 'gitlab'
    case GitProviders.AZURE:
      return 'azure'
    default:
      return provider.toLowerCase()
  }
}

export const getOrgLabel = (gitProvider: string) => {
  switch (gitProvider) {
    case GitProviders.BITBUCKET:
      return 'importRepo.workspace'
    case GitProviders.BITBUCKET_SERVER:
      return 'importRepo.project'
    case GitProviders.GITLAB:
    case GitProviders.GITLAB_SELF_HOSTED:
      return 'importRepo.group'
    default:
      return 'importRepo.org'
  }
}

export const getOrgPlaceholder = (gitProvider: string) => {
  switch (gitProvider) {
    case GitProviders.BITBUCKET:
      return 'importRepo.workspacePlaceholder'
    case GitProviders.BITBUCKET_SERVER:
      return 'importRepo.projectPlaceholder'
    case GitProviders.GITLAB:
    case GitProviders.GITLAB_SELF_HOSTED:
      return 'importRepo.groupPlaceholder'
    default:
      return 'importRepo.orgPlaceholder'
  }
}

export const getProviders = () =>
  Object.values(GitProviders).map(value => {
    return { value, label: value }
  })

export const codeOwnersNotFoundMessage = 'CODEOWNERS file not found'
export const codeOwnersNotFoundMessage2 = `path "CODEOWNERS" not found`
export const codeOwnersNotFoundMessage3 = `failed to find node 'CODEOWNERS' in 'main': failed to get tree node: failed to ls file: path "CODEOWNERS" not found`
export const oldCommitRefetchRequired = 'A newer commit is available. Only the latest commit can be merged.'
export const prMergedRefetchRequired = 'Pull request already merged'

export const dryMerge = (
  isMounted: React.MutableRefObject<boolean>,
  isClosed: boolean,
  pullReqMetadata: TypesPullReq,
  internalFlags: React.MutableRefObject<{
    dryRun: boolean
  }>,
  mergePR: (
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    data: any,
    mutateRequestOptions?:
      | MutateRequestOptions<
          {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            [key: string]: any
          },
          unknown
        >
      | undefined // eslint-disable-next-line @typescript-eslint/no-explicit-any
  ) => Promise<any>,
  setRuleViolation: (value: React.SetStateAction<boolean>) => void,
  setRuleViolationArr: (
    value: React.SetStateAction<
      | {
          data: {
            rule_violations: TypesRuleViolations[]
          }
        }
      | undefined
    >
  ) => void,
  setAllowedStrats: (value: React.SetStateAction<string[]>) => void,
  pullRequestSection: string | undefined,
  showError: (message: React.ReactNode, timeout?: number | undefined, key?: string | undefined) => void,
  setConflictingFiles: React.Dispatch<React.SetStateAction<string[] | undefined>>,
  refetchPullReq: () => void,
  setRequiresCommentApproval?: (value: React.SetStateAction<boolean>) => void,
  setAtLeastOneReviewerRule?: (value: React.SetStateAction<boolean>) => void,
  setReqCodeOwnerApproval?: (value: React.SetStateAction<boolean>) => void,
  setMinApproval?: (value: React.SetStateAction<number>) => void,
  setReqCodeOwnerLatestApproval?: (value: React.SetStateAction<boolean>) => void,
  setMinReqLatestApproval?: (value: React.SetStateAction<number>) => void,
  setPRStateLoading?: (value: React.SetStateAction<boolean>) => void,
  setDefaultReviewersInfoSet?: React.Dispatch<React.SetStateAction<TypesDefaultReviewerApprovalsResponse[]>>
) => {
  if (isMounted.current && !isClosed && pullReqMetadata?.state !== PullRequestState.MERGED) {
    // Use an internal flag to prevent flickering during the loading state of buttons
    internalFlags.current.dryRun = true
    mergePR({ bypass_rules: true, dry_run: true, source_sha: pullReqMetadata?.source_sha })
      .then(res => {
        if (res?.rule_violations?.length > 0) {
          setRuleViolation(true)
          setRuleViolationArr({ data: { rule_violations: res?.rule_violations } })
          setAllowedStrats(res.allowed_methods)
          setRequiresCommentApproval?.(res.requires_comment_resolution)
          setAtLeastOneReviewerRule?.(res.requires_no_change_requests)
          setReqCodeOwnerApproval?.(res.requires_code_owners_approval)
          setMinApproval?.(res.minimum_required_approvals_count)
          setReqCodeOwnerLatestApproval?.(res.requires_code_owners_approval_latest)
          setMinReqLatestApproval?.(res.minimum_required_approvals_count_latest)
          setConflictingFiles?.(res.conflict_files)
          setDefaultReviewersInfoSet?.(res.default_reviewer_aprovals)
        } else {
          setRuleViolation(false)
          setAllowedStrats(res.allowed_methods)
          setRequiresCommentApproval?.(res.requires_comment_resolution)
          setAtLeastOneReviewerRule?.(res.requires_no_change_requests)
          setReqCodeOwnerApproval?.(res.requires_code_owners_approval)
          setMinApproval?.(res.minimum_required_approvals_count)
          setReqCodeOwnerLatestApproval?.(res.requires_code_owners_approval_latest)
          setMinReqLatestApproval?.(res.minimum_required_approvals_count_latest)
          setConflictingFiles?.(res.conflict_files)
          setDefaultReviewersInfoSet?.(res.default_reviewer_aprovals)
        }
      })
      .catch(err => {
        if (err.status === 422) {
          setRuleViolation(true)
          setRuleViolationArr(err)
          setAllowedStrats(err.allowed_methods)
          setRequiresCommentApproval?.(err.requires_comment_resolution)
          setAtLeastOneReviewerRule?.(err.requires_no_change_requests)
          setReqCodeOwnerApproval?.(err.requires_code_owners_approval)
          setMinApproval?.(err.minimum_required_approvals_count)
          setReqCodeOwnerLatestApproval?.(err.requires_code_owners_approval_latest)
          setMinReqLatestApproval?.(err.minimum_required_approvals_count_latest)
          setConflictingFiles?.(err.conflict_files)
          setDefaultReviewersInfoSet?.(err.default_reviewer_aprovals)
        } else if (
          err.status === 400 &&
          [oldCommitRefetchRequired, prMergedRefetchRequired].includes(getErrorMessage(err) || '')
        ) {
          refetchPullReq()
        } else if (
          [codeOwnersNotFoundMessage, codeOwnersNotFoundMessage2, codeOwnersNotFoundMessage3].includes(
            getErrorMessage(err) || ''
          ) ||
          err.status === 423 // resource locked (merge / dry-run already ongoing)
        ) {
          return
        } else if (pullRequestSection !== PullRequestSection.CONVERSATION) {
          return
        } else {
          showError(getErrorMessage(err))
        }
      })
      .finally(() => {
        internalFlags.current.dryRun = false
        setPRStateLoading?.(false)
      })
  }
}

export enum WebhookEventType {
  PUSH = 'push',
  ALL = 'all',
  INDIVIDUAL = 'individual'
}

export enum WebhookIndividualEvent {
  BRANCH_CREATED = 'branch_created',
  BRANCH_UPDATED = 'branch_updated',
  BRANCH_DELETED = 'branch_deleted',
  TAG_CREATED = 'tag_created',
  TAG_UPDATED = 'tag_updated',
  TAG_DELETED = 'tag_deleted',
  PR_CREATED = 'pullreq_created',
  PR_UPDATED = 'pullreq_updated',
  PR_REOPENED = 'pullreq_reopened',
  PR_BRANCH_UPDATED = 'pullreq_branch_updated',
  PR_CLOSED = 'pullreq_closed',
  PR_COMMENT_CREATED = 'pullreq_comment_created',
  PR_COMMENT_STATUS_UPDATED = 'pullreq_comment_status_updated',
  PR_COMMENT_UPDATED = 'pullreq_comment_updated',
  PR_MERGED = 'pullreq_merged',
  PR_LABEL_ASSIGNED = 'pullreq_label_assigned',
  PR_REVIEW_SUBMITTED = 'pullreq_review_submitted'
}

export enum WebhookEventMap {
  BRANCH_CREATED = 'Branch created',
  BRANCH_UPDATED = 'Branch updated',
  BRANCH_DELETED = 'Branch deleted',
  TAG_CREATED = 'Tag created',
  TAG_UPDATED = 'Tag updated',
  TAG_DELETED = 'Tag deleted',
  PR_CREATED = 'PR created',
  PR_UPDATED = 'PR updated',
  PR_REOPENED = 'PR reopened',
  PR_BRANCH_UPDATED = 'PR branch updated',
  PR_CLOSED = 'PR closed',
  PR_COMMENT_CREATED = 'PR comment created',
  PR_COMMENT_STATUS_UPDATED = 'PR comment status updated',
  PR_COMMENT_UPDATED = 'PR comment updated',
  PR_MERGED = 'PR merged',
  PR_LABEL_ASSIGNED = 'PR label assigned',
  PR_REVIEW_SUBMITTED = 'PR review submitted'
}

export const eventMapping: Record<WebhookIndividualEvent, WebhookEventMap> = {
  [WebhookIndividualEvent.BRANCH_CREATED]: WebhookEventMap.BRANCH_CREATED,
  [WebhookIndividualEvent.BRANCH_UPDATED]: WebhookEventMap.BRANCH_UPDATED,
  [WebhookIndividualEvent.BRANCH_DELETED]: WebhookEventMap.BRANCH_DELETED,
  [WebhookIndividualEvent.TAG_CREATED]: WebhookEventMap.TAG_CREATED,
  [WebhookIndividualEvent.TAG_UPDATED]: WebhookEventMap.TAG_UPDATED,
  [WebhookIndividualEvent.TAG_DELETED]: WebhookEventMap.TAG_DELETED,
  [WebhookIndividualEvent.PR_CREATED]: WebhookEventMap.PR_CREATED,
  [WebhookIndividualEvent.PR_UPDATED]: WebhookEventMap.PR_UPDATED,
  [WebhookIndividualEvent.PR_REOPENED]: WebhookEventMap.PR_REOPENED,
  [WebhookIndividualEvent.PR_BRANCH_UPDATED]: WebhookEventMap.PR_BRANCH_UPDATED,
  [WebhookIndividualEvent.PR_CLOSED]: WebhookEventMap.PR_CLOSED,
  [WebhookIndividualEvent.PR_COMMENT_CREATED]: WebhookEventMap.PR_COMMENT_CREATED,
  [WebhookIndividualEvent.PR_COMMENT_STATUS_UPDATED]: WebhookEventMap.PR_COMMENT_STATUS_UPDATED,
  [WebhookIndividualEvent.PR_COMMENT_UPDATED]: WebhookEventMap.PR_COMMENT_UPDATED,
  [WebhookIndividualEvent.PR_MERGED]: WebhookEventMap.PR_MERGED,
  [WebhookIndividualEvent.PR_LABEL_ASSIGNED]: WebhookEventMap.PR_LABEL_ASSIGNED,
  [WebhookIndividualEvent.PR_REVIEW_SUBMITTED]: WebhookEventMap.PR_REVIEW_SUBMITTED
}

export function getEventDescription(event: WebhookIndividualEvent): string {
  return eventMapping[event]
}

export const mergeMethodMapping: Record<MergeStrategy, MergeMethodDisplay> = {
  [MergeStrategy.MERGE]: MergeMethodDisplay.MERGED,
  [MergeStrategy.SQUASH]: MergeMethodDisplay.SQUASHED,
  [MergeStrategy.REBASE]: MergeMethodDisplay.REBASED,
  [MergeStrategy.FAST_FORWARD]: MergeMethodDisplay.FAST_FORWARDED
}

export function getMergeMethodDisplay(mergeMethodType: MergeStrategy): string {
  return mergeMethodMapping[mergeMethodType]
}
