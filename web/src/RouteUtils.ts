/**
 * Generate route paths to be used in RouteDefinitions.
 * @param path route path
 * @returns an array of proper route paths that works in both standalone and embedded modes across all levels of governance.
 */
export const routePath = (standalone: boolean, path: string): string | string[] =>
  standalone ? path : [`/account/:accountId/scm${path}`]
