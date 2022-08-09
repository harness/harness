import { generatePath } from 'react-router-dom'
import type { AppPathProps } from 'AppProps'

let baseRoutePath: string
let accountId: string

export const setBaseRouteInfo = (_accountId: string, _baseRoutePath: string): void => {
  accountId = _accountId
  baseRoutePath = _baseRoutePath
}

type Scope = Pick<AppPathProps, 'orgIdentifier' | 'projectIdentifier' | 'module'>

//
// Note: This function needs to be in sync with NextGen UI's routeUtils' getScopeBasedRoute. When
// it's out of sync, the URL routing scheme could be broken.
// @see https://github.com/harness/harness-core-ui/blob/master/src/modules/10-common/utils/routeUtils.ts#L171
//
const getScopeBasedRouteURL = ({ path, scope = {} }: { path: string; scope?: Scope }): string => {
  if (window.APP_RUN_IN_STANDALONE_MODE) {
    return path
  }

  const { orgIdentifier, projectIdentifier, module } = scope

  //
  // TODO: Change this scheme below to reflect your application when it's embedded into Harness NextGen UI
  //

  // The Sample Module UI app is mounted in three places in Harness Platform
  //  1. Account Settings (account level)
  //  2. Org Details (org level)
  //  3. Project Settings (project level)
  if (module && orgIdentifier && projectIdentifier) {
    return `/account/${accountId}/${module}/orgs/${orgIdentifier}/projects/${projectIdentifier}/setup/sample-module${path}`
  } else if (orgIdentifier && projectIdentifier) {
    return `/account/${accountId}/home/orgs/${orgIdentifier}/projects/${projectIdentifier}/setup/sample-module${path}`
  } else if (orgIdentifier) {
    return `/account/${accountId}/settings/organizations/${orgIdentifier}/setup/sample-module${path}`
  }

  return `/account/${accountId}/settings/sample-module${path}`
}

/**
 * Generate route path to be used in RouteDefinitions.
 * @param path route path
 * @returns a proper route path that works in both standalone and embedded modes.
 */
export const routePath = (path: string): string => `${baseRoutePath || ''}${path}`

/**
 * Generate route URL to be used RouteDefinitions' default export (aka actual react-router link href)
 * @param path route path
 * @param params URL parameters
 * @returns a proper URL that works in both standalone and embedded modes.
 */
export const toRouteURL = (path: string, params?: AppPathProps): string =>
  generatePath(getScopeBasedRouteURL({ path, scope: params }), { ...params, accountId })
