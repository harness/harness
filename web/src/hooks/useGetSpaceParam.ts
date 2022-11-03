import { useParams } from 'react-router-dom'
import type { SCMPathProps } from 'RouteDefinitions'
import { useAppContext } from 'AppContext'

/**
 * Get space parameter.
 * Space is passed differently in standalone and embedded modes. In standalone
 * mode, space is available from routing url (aka being passed as `:space`
 * using react-router). In embedded mode, Harness UI's routing always works with
 * `:accountId/orgs/:orgIdentifier/projects/:projectIdentifier` so we can't get
 * `space` from URL. As a result, we have to pass `space` via AppContext.
 * @returns space parameter.
 */
export function useGetSpaceParam() {
  const { space: spaceFromParams = '' } = useParams<SCMPathProps>()
  const { space = spaceFromParams } = useAppContext()
  return space
}
