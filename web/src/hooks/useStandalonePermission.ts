import type { Unknown } from 'utils/Utils'

export function useStandalonePermission(_permissionsRequest?: Unknown, _deps: Array<Unknown> = []): Array<boolean> {
  return [true, true]
}
