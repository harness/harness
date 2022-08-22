export enum RoutePath {
  // Standalone-only paths
  SIGNIN = '/signin',
  SIGNUP = '/signup',

  // Shared paths
  DASHBOARD = '/dashboard',
  TEST_PAGE1 = '/test1',
  TEST_PAGE2 = '/test2'
}

export default {
  toSignIn: (): string => RoutePath.SIGNIN,
  toSignUp: (): string => RoutePath.SIGNUP,

  toDashboard: (): string => RoutePath.DASHBOARD,
  toTestPage1: (): string => RoutePath.TEST_PAGE1,
  toTestPage2: (): string => RoutePath.TEST_PAGE2
}
