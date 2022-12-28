import type { UserProfile } from 'utils/types'

export function useCurrentUser(): UserProfile {
  // TODO: Implement this hook to get current user that works
  // in both standalone and embedded mode
  return {
    id: '0',
    email: 'admin@harness.io',
    name: 'Admin'
  }
}
