export interface SCMPathProps {
  space?: string
  repoName?: string
  gitRef?: string
  resourcePath?: string
  commitRef?: string
}

export interface SCMQueryProps {
  query?: string
}

export const pathProps: Readonly<Required<SCMPathProps>> = {
  space: ':space',
  repoName: ':repoName',
  gitRef: ':gitRef*',
  resourcePath: ':resourcePath*',
  commitRef: ':commitRef*'
}

export interface SCMRoutes {
  toSignIn: () => string
  toSignUp: () => string
  toSCMRepositoriesListing: ({ space }: { space: string }) => string
  toSCMRepository: ({
    repoPath,
    gitRef,
    resourcePath
  }: {
    repoPath: string
    gitRef?: string
    resourcePath?: string
  }) => string
  toSCMRepositoryFileEdit: ({
    repoPath,
    gitRef,
    resourcePath
  }: {
    repoPath: string
    gitRef: string
    resourcePath: string
  }) => string
  toSCMRepositoryCommits: ({ repoPath, commitRef }: { repoPath: string; commitRef: string }) => string
  toSCMRepositoryBranches: ({ repoPath, branch }: { repoPath: string; branch?: string }) => string
}

export const routes: SCMRoutes = {
  toSignIn: (): string => '/signin',
  toSignUp: (): string => '/signup',
  toSCMRepositoriesListing: ({ space }) => `/${space}`,
  toSCMRepository: ({ repoPath, gitRef, resourcePath }) =>
    `/${repoPath}/${gitRef ? '/' + gitRef : ''}${resourcePath ? '/~/' + resourcePath : ''}`,
  toSCMRepositoryFileEdit: ({ repoPath, gitRef, resourcePath }) => `/${repoPath}/edit/${gitRef}/~/${resourcePath}`,
  toSCMRepositoryCommits: ({ repoPath, commitRef }) => `/${repoPath}/commits/${commitRef}`,
  toSCMRepositoryBranches: ({ repoPath, branch }) => `/${repoPath}/branches/${branch ? '/' + branch : ''}`
}
