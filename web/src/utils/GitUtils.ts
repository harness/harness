// Code copied from https://github.com/vweevers/is-git-ref-name-valid and
// https://github.com/vweevers/is-git-branch-name-valid (MIT, © Vincent Weevers)
// Last updated for git 2.29.0.

import type { IconName } from '@harness/icons'
import type {
  OpenapiContentInfo,
  OpenapiDirContent,
  OpenapiGetContentOutput,
  TypesCommit,
  TypesPullReq,
  TypesRepository
} from 'services/code'

export interface GitInfoProps {
  repoMetadata: TypesRepository
  gitRef: string
  resourcePath: string
  resourceContent: OpenapiGetContentOutput
  commitRef: string
  commits: TypesCommit[]
  pullRequestMetadata: TypesPullReq
}

export enum GitContentType {
  FILE = 'file',
  DIR = 'dir',
  SYMLINK = 'symlink',
  SUBMODULE = 'submodule'
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

export const PullRequestFilterOption = {
  ...PullRequestState,
  // REJECTED: 'rejected',
  DRAFT: 'draft',
  YOURS: 'yours',
  ALL: 'all'
}

export const CodeIcon = {
  Logo: 'code' as IconName,
  PullRequest: 'git-pull' as IconName,
  Merged: 'code-merged' as IconName,
  Draft: 'code-draft' as IconName,
  PullRequestRejected: 'main-close' as IconName,
  Add: 'plus' as IconName,
  BranchSmall: 'code-branch-small' as IconName,
  Branch: 'code-branch' as IconName,
  Tag: 'main-tags' as IconName,
  Clone: 'code-clone' as IconName,
  Close: 'code-close' as IconName,
  CommitLight: 'code-commit-light' as IconName,
  CommitSmall: 'code-commit-small' as IconName,
  Commit: 'code-commit' as IconName,
  Copy: 'code-copy' as IconName,
  Delete: 'code-delete' as IconName,
  Edit: 'code-edit' as IconName,
  FileLight: 'code-file-light' as IconName,
  File: 'code-file' as IconName,
  Folder: 'code-folder' as IconName,
  History: 'code-history' as IconName,
  Info: 'code-info' as IconName,
  More: 'code-more' as IconName,
  Repo: 'code-repo' as IconName,
  Settings: 'code-settings' as IconName,
  Webhook: 'code-webhook' as IconName,
  InputSpinner: 'spinner' as IconName,
  InputSearch: 'search' as IconName,
  Chat: 'code-chat' as IconName,
  Checks: 'main-tick' as IconName,
  ChecksSuccess: 'success-tick' as IconName
}

export const REFS_TAGS_PREFIX = 'refs/tags/'

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

export const decodeGitContent = (content = '') => decodeURIComponent(escape(window.atob(content)))
