// Code copied from https://github.com/vweevers/is-git-ref-name-valid and
// https://github.com/vweevers/is-git-branch-name-valid (MIT, © Vincent Weevers)
// Last updated for git 2.29.0.

import type { IconName } from '@harness/icons'
import type { OpenapiContentInfo, OpenapiDirContent, OpenapiGetContentOutput, TypesRepository } from 'services/scm'

// eslint-disable-next-line no-control-regex
const badGitRefRegrex = /(^|[/.])([/.]|$)|^@$|@{|[\x00-\x20\x7f~^:?*[\\]|\.lock(\/|$)/
const badGitBranchRegrex = /^(-|HEAD$)/

function isGitRefValid(name: string, onelevel: boolean): boolean {
  return !badGitRefRegrex.test(name) && (!!onelevel || name.includes('/'))
}

export function isGitBranchNameValid(name: string): boolean {
  return isGitRefValid(name, true) && !badGitBranchRegrex.test(name)
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

export const GitIcon: Readonly<Record<string, IconName>> = {
  FILE: 'file',
  REPOSITORY: 'code-repo',
  COMMIT: 'git-branch-existing',
  PULL_REQUEST: 'git-pull',
  SETTINGS: 'cog',
  FOLDER: 'code-folder',
  COPY: 'code-copy',
  DELETE: 'code-delete',
  EDIT: 'Edit',
  BRANCH: 'git-branch',
  HISTORY: 'code-history'
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

export interface GitInfoProps {
  repoMetadata: TypesRepository
  gitRef: string
  resourcePath: string
  resourceContent: OpenapiGetContentOutput
  commitRef: string
}
