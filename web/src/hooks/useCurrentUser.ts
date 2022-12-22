export interface CurrentUser {
  name: string
  email: string
}

export function useCurrentUser(): CurrentUser {
  // TODO: Implement this hook to get current user that works
  // in both standalone and embedded mode
  return {
    email: 'admin@harness.io',
    name: 'Admin'
  }
}
