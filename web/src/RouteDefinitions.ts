export interface CODEPathProps {
  space?: string
  repoName?: string
  gitRef?: string
  resourcePath?: string
  commitRef?: string
}

export interface CODEQueryProps {
  query?: string
}

export const pathProps: Readonly<Required<CODEPathProps>> = {
  space: ':space',
  repoName: ':repoName',
  gitRef: ':gitRef*',
  resourcePath: ':resourcePath*',
  commitRef: ':commitRef*'
}

export interface CODERoutes {
  toSignIn: () => string
  toSignUp: () => string
  toCODERepositoriesListing: ({ space }: { space: string }) => string
  toCODERepository: ({
    repoPath,
    gitRef,
    resourcePath
  }: {
    repoPath: string
    gitRef?: string
    resourcePath?: string
  }) => string
  toCODERepositoryFileEdit: ({
    repoPath,
    gitRef,
    resourcePath
  }: {
    repoPath: string
    gitRef: string
    resourcePath: string
  }) => string

  toCODERepositoryCommits: ({ repoPath, commitRef }: { repoPath: string; commitRef: string }) => string
  toCODERepositoryBranches: ({ repoPath, branch }: { repoPath: string; branch?: string }) => string
  toCODERepositorySettings: ({ repoPath }: { repoPath: string }) => string
  toCODECreateWebhook: ({ repoPath }: { repoPath: string }) => string
}

export const routes: CODERoutes = {
  toSignIn: (): string => '/signin',
  toSignUp: (): string => '/signup',
  toCODERepositoriesListing: ({ space }) => `/${space}`,
  toCODERepository: ({ repoPath, gitRef, resourcePath }) =>
    `/${repoPath}/${gitRef ? '/' + gitRef : ''}${resourcePath ? '/~/' + resourcePath : ''}`,
  toCODERepositoryFileEdit: ({ repoPath, gitRef, resourcePath }) => `/${repoPath}/edit/${gitRef}/~/${resourcePath}`,
  toCODERepositoryCommits: ({ repoPath, commitRef }) => `/${repoPath}/commits/${commitRef}`,
  toCODERepositoryBranches: ({ repoPath, branch }) => `/${repoPath}/branches/${branch ? '/' + branch : ''}`,
  toCODERepositorySettings: ({ repoPath }) => `/${repoPath}/settings`,
  toCODECreateWebhook: ({ repoPath }) => `/${repoPath}/settings/webhook/new`
}
