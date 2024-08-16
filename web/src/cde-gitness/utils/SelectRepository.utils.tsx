import React from 'react'
import { Bitbucket as BitbucketIcon, Code, GitLabFull, GithubCircle } from 'iconoir-react'
import type { EnumCodeRepoType } from 'services/cde'
import type { EnumGitspaceCodeRepoType } from 'cde-gitness/services'

export const isValidUrl = (url: string) => {
  const urlPattern = new RegExp(
    '^(https?:\\/\\/)?' + // validate protocol
      '((([a-z\\d]([a-z\\d-]*[a-z\\d])*)\\.)+[a-z]{2,}|' + // validate domain name
      '((\\d{1,3}\\.){3}\\d{1,3}))' + // validate OR ip (v4) address
      '(\\:\\d+)?(\\/[-a-z\\d%_.~+]*)*' + // validate port and path
      '(\\?[;&a-z\\d%_.~+=-]*)?' + // validate query string
      '(\\#[-a-z\\d_]*)?$',
    'i'
  ) // validate fragment locator
  return !!urlPattern.test(url)
}

export const getRepoIdFromURL = (repoURL?: string) => {
  const repoURLSplit = repoURL?.split('/')
  return repoURLSplit?.[repoURLSplit?.length - 1]
    ?.replace(/-/g, '')
    ?.replace(/_/g, '')
    .replace(/\./g, '')
    ?.toLowerCase()
}

export const getRepoNameFromURL = (repoURL?: string) => {
  const repoURLSplit = repoURL?.split('/')
  return repoURLSplit?.[repoURLSplit?.length - 1]
}

export enum CodeRepoType {
  Github = 'github',
  Gitlab = 'gitlab',
  HarnessCode = 'harnessCode',
  Bitbucket = 'bitbucket',
  Unknown = 'unknown'
}

export const getIconByRepoType = ({
  repoType,
  height = 40
}: {
  repoType?: EnumCodeRepoType | EnumGitspaceCodeRepoType
  height?: number
}): React.ReactNode => {
  switch (repoType) {
    case CodeRepoType.Github:
      return <GithubCircle height={height} />
    case CodeRepoType.Gitlab:
      return <GitLabFull height={height} />
    case CodeRepoType.Bitbucket:
      return <BitbucketIcon height={height} />
    default:
    case CodeRepoType.Unknown:
    case CodeRepoType.HarnessCode:
    case 'harness_code':
      return <Code height={height} />
  }
}
