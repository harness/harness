export interface CODEProps {
  space?: string
  repoName?: string
  repoPath?: string
  gitRef?: string
  resourcePath?: string
  commitRef?: string
  branch?: string
  tags?: string
  diffRefs?: string
  pullRequestId?: string
  pullRequestSection?: string
  webhookId?: string
  pipeline?: string
  pipelinePath?: string
  execution?: string
  executionPath?: string
}

export interface CODEQueryProps {
  query?: string
}

export const pathProps: Readonly<
  Omit<Required<CODEProps>, 'repoPath' | 'branch' | 'tags' | 'pipelinePath' | 'executionPath'>
> = {
  space: ':space*',
  repoName: ':repoName',
  gitRef: ':gitRef*',
  resourcePath: ':resourcePath*',
  commitRef: ':commitRef*',
  diffRefs: ':diffRefs*',
  pullRequestId: ':pullRequestId',
  pullRequestSection: ':pullRequestSection*',
  webhookId: ':webhookId',
  pipeline: ':pipeline',
  execution: ':execution'
}

export interface CODERoutes {
  toSignIn: () => string
  toRegister: () => string

  toCODEHome: () => string

  toCODESpaceAccessControl: (args: Required<Pick<CODEProps, 'space'>>) => string
  toCODESpaceSettings: (args: Required<Pick<CODEProps, 'space'>>) => string
  toCODEPipelines: (args: Required<Pick<CODEProps, 'space'>>) => string
  toCODESecrets: (args: Required<Pick<CODEProps, 'space'>>) => string

  toCODEGlobalSettings: () => string
  toCODEUsers: () => string
  toCODEUserProfile: () => string
  toCODEUserChangePassword: () => string

  toCODERepositories: (args: Required<Pick<CODEProps, 'space'>>) => string
  toCODERepository: (args: RequiredField<Pick<CODEProps, 'repoPath' | 'gitRef' | 'resourcePath'>, 'repoPath'>) => string
  toCODEFileEdit: (args: Required<Pick<CODEProps, 'repoPath' | 'gitRef' | 'resourcePath'>>) => string
  toCODECommits: (args: Required<Pick<CODEProps, 'repoPath' | 'commitRef'>>) => string
  toCODECommit: (args: Required<Pick<CODEProps, 'repoPath' | 'commitRef'>>) => string
  toCODEPullRequests: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODEPullRequest: (
    args: RequiredField<
      Pick<CODEProps, 'repoPath' | 'pullRequestId' | 'pullRequestSection'>,
      'repoPath' | 'pullRequestId'
    >
  ) => string
  toCODECompare: (args: Required<Pick<CODEProps, 'repoPath' | 'diffRefs'>>) => string
  toCODEBranches: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODETags: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODEWebhooks: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODEWebhookNew: (args: Required<Pick<CODEProps, 'repoPath'>>) => string
  toCODEWebhookDetails: (args: Required<Pick<CODEProps, 'repoPath' | 'webhookId'>>) => string
  toCODESettings: (args: Required<Pick<CODEProps, 'repoPath'>>) => string

  toCODEExecutions: (args: Required<Pick<CODEProps, 'pipelinePath'>>) => string
  toCODEExecution: (args: Required<Pick<CODEProps, 'executionPath'>>) => string
}

export const routes: CODERoutes = {
  toSignIn: (): string => '/signin',
  toRegister: (): string => '/register',

  toCODEHome: () => `/`,

  toCODESpaceAccessControl: ({ space }) => `/access-control/${space}`,
  toCODESpaceSettings: ({ space }) => `/settings/${space}`,
  toCODEPipelines: ({ space }) => `/pipelines/${space}`,
  toCODESecrets: ({ space }) => `/secrets/${space}`,

  toCODEGlobalSettings: () => '/settings',
  toCODEUsers: () => '/users',
  toCODEUserProfile: () => '/profile',
  toCODEUserChangePassword: () => '/change-password',

  toCODERepositories: ({ space }) => `/spaces/${space}`,
  toCODERepository: ({ repoPath, gitRef, resourcePath }) =>
    `/${repoPath}${gitRef ? '/files/' + gitRef : ''}${resourcePath ? '/~/' + resourcePath : ''}`,
  toCODEFileEdit: ({
    repoPath,
    gitRef,
    resourcePath
  }: RequiredField<Pick<CODEProps, 'repoPath' | 'gitRef' | 'resourcePath'>, 'repoPath' | 'gitRef'>) =>
    `/${repoPath}/edit/${gitRef}/~/${resourcePath || ''}`,

  toCODECommits: ({ repoPath, commitRef }) => `/${repoPath}/commits/${commitRef}`,
  toCODECommit: ({ repoPath, commitRef }) => `/${repoPath}/commit/${commitRef}`,
  toCODEPullRequests: ({ repoPath }) => `/${repoPath}/pulls`,
  toCODEPullRequest: ({ repoPath, pullRequestId, pullRequestSection }) =>
    `/${repoPath}/pulls/${pullRequestId}${pullRequestSection ? '/' + pullRequestSection : ''}`,
  toCODECompare: ({ repoPath, diffRefs }) => `/${repoPath}/pulls/compare/${diffRefs}`,
  toCODEBranches: ({ repoPath }) => `/${repoPath}/branches`,
  toCODETags: ({ repoPath }) => `/${repoPath}/tags`,
  toCODESettings: ({ repoPath }) => `/${repoPath}/settings`,
  toCODEWebhooks: ({ repoPath }) => `/${repoPath}/webhooks`,
  toCODEWebhookNew: ({ repoPath }) => `/${repoPath}/webhooks/new`,
  toCODEWebhookDetails: ({ repoPath, webhookId }) => `/${repoPath}/webhook/${webhookId}`,

  toCODEExecutions: ({ pipelinePath }) => `/pipelines/${pipelinePath}`,
  toCODEExecution: ({ executionPath }) => `/pipelines/${executionPath}`
}
