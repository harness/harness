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
// @see https://github.com/wings-software/nextgenui/blob/master/src/modules/10-common/utils/routeUtils.ts#L171
//
const getScopeBasedRouteURL = ({ path, scope = {} }: { path: string; scope?: Scope }): string => {
  const { orgIdentifier, projectIdentifier, module } = scope

  // The Governance app is mounted in three places in Harness Platform
  //  1. Account Settings (account level governance)
  //  2. Org Details (org level governance)
  //  3. Project Settings (project level governance)
  if (module && orgIdentifier && projectIdentifier) {
    return `/account/${accountId}/${module}/orgs/${orgIdentifier}/projects/${projectIdentifier}/setup/governance${path}`
  } else if (orgIdentifier && projectIdentifier) {
    return `/account/${accountId}/home/orgs/${orgIdentifier}/projects/${projectIdentifier}/setup/governance${path}`
  } else if (orgIdentifier) {
    return `/account/${accountId}/settings/organizations/${orgIdentifier}/setup/governance${path}`
  }

  return `/account/${accountId}/settings/governance${path}`
}

/**
 * Generate route paths to be used in RouteDefinitions.
 * @param path route path
 * @returns an array of proper route paths that works in both standalone and embedded modes across all levels of governance.
 */
export const routePath = (path: string): string[] => [
  `/account/:accountId/settings/governance${path}`,
  `/account/:accountId/settings/organizations/:orgIdentifier/setup/governance${path}`,
  `/account/:accountId/:module(cd)/orgs/:orgIdentifier/projects/:projectIdentifier/setup/governance${path}`,
  `/account/:accountId/:module(ci)/orgs/:orgIdentifier/projects/:projectIdentifier/setup/governance${path}`,
  `/account/:accountId/:module(cf)/orgs/:orgIdentifier/projects/:projectIdentifier/setup/governance${path}`,
  `/account/:accountId/:module(sto)/orgs/:orgIdentifier/projects/:projectIdentifier/setup/governance${path}`,
  `/account/:accountId/:module(cv)/orgs/:orgIdentifier/projects/:projectIdentifier/setup/governance${path}`,
]

export const standaloneRoutePath = (path: string): string => `${baseRoutePath || ''}${path}`

/**
 * Generate route URL to be used RouteDefinitions' default export (aka actual react-router link href)
 * @param path route path
 * @param params URL parameters
 * @returns a proper URL that works in both standalone and embedded modes.
 */
export const toRouteURL = (path: string, params?: AppPathProps): string =>
  generatePath(getScopeBasedRouteURL({ path, scope: params }), { ...params, accountId })
