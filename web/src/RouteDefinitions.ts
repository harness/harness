export interface SCMPathProps {
  space?: string
  repoName?: string
  gitRef?: string
  resourcePath?: string
}

export interface SCMQueryProps {
  branch?: string
  filePath?: string
}

export const pathProps: Readonly<Required<SCMPathProps>> = {
  space: ':space',
  repoName: ':repoName',
  gitRef: ':gitRef*',
  resourcePath: ':resourcePath'
}

// function withAccountId<T>(fn: (args: T) => string) {
//   return (params: T & { accountId: string }): string => {
//     const path = fn(params)
//     return `/account/${params.accountId}/${path.replace(/^\//, '')}`
//   }
// }

// function withQueryParams(path: string, params: Record<string, string>): string {
//   return Object.entries(params).every(([key, value]) => ':' + key === value)
//     ? path
//     : [
//         path,
//         Object.entries(params)
//           .reduce((value, entry) => {
//             value.push(entry.join('='))
//             return value
//           }, [] as string[])
//           .join('&')
//       ].join('?')
// }

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
}

export const routes: SCMRoutes = {
  toSignIn: (): string => '/signin',
  toSignUp: (): string => '/signup',
  toSCMRepositoriesListing: ({ space }: { space: string }) => {
    const [accountId, orgIdentifier, projectIdentifier] = space.split('/')
    return `/account/${accountId}/code/${orgIdentifier}/${projectIdentifier}`
  },
  toSCMRepository: ({
    repoPath,
    gitRef,
    resourcePath
  }: {
    repoPath: string
    gitRef?: string
    resourcePath?: string
  }) => {
    const [accountId, orgIdentifier, projectIdentifier, repoName] = repoPath.split('/')
    return `/account/${accountId}/code/${orgIdentifier}/${projectIdentifier}/${repoName}${gitRef ? '/' + gitRef : ''}${
      resourcePath ? '/~/' + resourcePath : ''
    }`
  }
}
