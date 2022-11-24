// Code copied from https://github.com/vweevers/is-git-ref-name-valid and
// https://github.com/vweevers/is-git-branch-name-valid (MIT, © Vincent Weevers)
// Last updated for git 2.29.0.

import type { IconName } from '@harness/icons'
import type { OpenapiContentInfo, OpenapiDirContent, OpenapiGetContentOutput, TypesRepository } from 'services/code'

export interface GitInfoProps {
  repoMetadata: TypesRepository
  gitRef: string
  resourcePath: string
  resourceContent: OpenapiGetContentOutput
  commitRef: string
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

export const CodeIcon = {
  PullRequest: 'git-pull' as IconName,
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
  InputSpinner: 'steps-spinner' as IconName,
  InputSearch: 'search' as IconName
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
