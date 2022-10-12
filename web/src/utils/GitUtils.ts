// Code copied from https://github.com/vweevers/is-git-ref-name-valid and
// https://github.com/vweevers/is-git-branch-name-valid (MIT, © Vincent Weevers)
// Last updated for git 2.29.0.

import type { IconName } from '@harness/icons'

// eslint-disable-next-line no-control-regex
const badGitRefRegrex = /(^|[/.])([/.]|$)|^@$|@{|[\x00-\x20\x7f~^:?*[\\]|\.lock(\/|$)/
const badGitBranchRegrex = /^(-|HEAD$)/

function isGitRefValid(name: string, onelevel: boolean): boolean {
  return !badGitRefRegrex.test(name) && (!!onelevel || name.includes('/'))
}

export function isGitBranchNameValid(name: string): boolean {
  return isGitRefValid(name, true) && !badGitBranchRegrex.test(name)
}

export const GitIcon: Readonly<Record<string, IconName>> = {
  FILE: 'file',
  REPOSITORY: 'git-repo',
  COMMIT: 'git-branch-existing',
  PULL_REQUEST: 'git-pull',
  SETTINGS: 'cog'
}
