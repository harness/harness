interface PathProps {
  accountId: string
  orgIdentifier: string
  projectIdentifier: string
  repoName?: string
  branchName?: string
  filePath?: string
  pullRequestId?: string
  commitId?: string
}

export const pathProps: Readonly<Required<PathProps>> = {
  accountId: ':accountId',
  orgIdentifier: ':orgIdentifier',
  projectIdentifier: ':projectIdentifier',
  repoName: ':repoName',
  branchName: ':branchName',
  filePath: ':filePath',
  pullRequestId: ':pullRequestId',
  commitId: ':commitId'
}

function withAccountId<T>(fn: (args: T) => string) {
  return (params: T & { accountId: string }): string => {
    const path = fn(params)
    return `/account/${params.accountId}/${path.replace(/^\//, '')}`
  }
}

export default {
  toSignIn: (): string => '/signin',
  toSignUp: (): string => '/signup',

  toSCM: withAccountId(() => `/scm`),
  toSCMHome: withAccountId(() => `/scm/home`),
  toSCMRepos: withAccountId(
    ({ orgIdentifier, projectIdentifier }: PathProps) =>
      `/scm/orgs/${orgIdentifier}/projects/${projectIdentifier}/repos`
  ),
  toSCMNewRepo: withAccountId(
    ({ orgIdentifier, projectIdentifier }: PathProps) =>
      `/scm/orgs/${orgIdentifier}/projects/${projectIdentifier}/repos/new`
  ),
  toSCMRepoSettings: withAccountId(
    ({ orgIdentifier, projectIdentifier, repoName }: RequireField<PathProps, 'repoName'>) =>
      `/scm/orgs/${orgIdentifier}/projects/${projectIdentifier}/repos/${repoName}/settings`
  ),
  toSCMFiles: withAccountId(
    ({ orgIdentifier, projectIdentifier, repoName, branchName }: RequireField<PathProps, 'repoName' | 'branchName'>) =>
      `/scm/orgs/${orgIdentifier}/projects/${projectIdentifier}/repos/${repoName}/branches/${branchName}`
  ),
  toSCMFileDetails: withAccountId(
    ({
      orgIdentifier,
      projectIdentifier,
      repoName,
      branchName,
      filePath
    }: RequireField<PathProps, 'repoName' | 'branchName' | 'filePath'>) =>
      `/scm/orgs/${orgIdentifier}/projects/${projectIdentifier}/repos/${repoName}/branches/${branchName}/files/${filePath}`
  ),
  toSCMPullRequests: withAccountId(
    ({ orgIdentifier, projectIdentifier, repoName, branchName }: RequireField<PathProps, 'repoName' | 'branchName'>) =>
      `/scm/orgs/${orgIdentifier}/projects/${projectIdentifier}/repos/${repoName}/branches/${branchName}/pull-requests`
  ),
  toSCMPullRequestDetails: withAccountId(
    ({
      orgIdentifier,
      projectIdentifier,
      repoName,
      branchName,
      pullRequestId
    }: RequireField<PathProps, 'repoName' | 'branchName' | 'pullRequestId'>) =>
      `/scm/orgs/${orgIdentifier}/projects/${projectIdentifier}/repos/${repoName}/branches/${branchName}/pull-requests/${pullRequestId}`
  ),
  toSCMCommits: withAccountId(
    ({ orgIdentifier, projectIdentifier, repoName, branchName }: RequireField<PathProps, 'repoName' | 'branchName'>) =>
      `/scm/orgs/${orgIdentifier}/projects/${projectIdentifier}/repos/${repoName}/branches/${branchName}/commits`
  ),
  toSCMCommitDetails: withAccountId(
    ({
      orgIdentifier,
      projectIdentifier,
      repoName,
      branchName,
      commitId
    }: RequireField<PathProps, 'repoName' | 'branchName' | 'commitId'>) =>
      `/scm/orgs/${orgIdentifier}/projects/${projectIdentifier}/repos/${repoName}/branches/${branchName}/commits/${commitId}`
  )
}
