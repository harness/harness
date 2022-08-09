import { toRouteURL } from 'RouteUtils'
import type { AppPathProps } from 'AppProps'

export enum RoutePath {
  SIGNIN = '/signin',
  TEST_PAGE1 = '/test-page1',
  TEST_PAGE2 = '/test-page2',

  REGISTER = '/register',
  LOGIN = '/login',
  USERS = '/users',
  ACCOUNT = '/account',
  PIPELINES = '/pipelines',
  PIPELINE = '/pipelines/:pipeline',
  PIPELINE_SETTINGS = '/pipelines/:pipeline/settings',
  PIPELINE_EXECUTIONS = '/pipelines/:pipeline/executions',
  PIPELINE_EXECUTION = '/pipelines/:pipeline/executions/:execution',
  PIPELINE_EXECUTION_SETTINGS = '/pipelines/:pipeline/executions/:execution/settings'
}

export default {
  toLogin: (): string => toRouteURL(RoutePath.LOGIN),
  toRegister: (): string => toRouteURL(RoutePath.REGISTER),
  toAccount: (): string => toRouteURL(RoutePath.ACCOUNT),
  toPipelines: (): string => toRouteURL(RoutePath.PIPELINES),
  toPipeline: ({ pipeline }: Required<Pick<AppPathProps, 'pipeline'>>): string =>
    toRouteURL(RoutePath.PIPELINE, { pipeline }),
  toPipelineExecutions: ({ pipeline }: Required<Pick<AppPathProps, 'pipeline'>>): string =>
    toRouteURL(RoutePath.PIPELINE_EXECUTIONS, { pipeline }),
  toPipelineSettings: ({ pipeline }: Required<Pick<AppPathProps, 'pipeline'>>): string =>
    toRouteURL(RoutePath.PIPELINE_SETTINGS, { pipeline }),
  toPipelineExecution: ({ pipeline, execution }: AppPathProps): string =>
    toRouteURL(RoutePath.PIPELINE_EXECUTION, { pipeline, execution }),
  toPipelineExecutionSettings: ({ pipeline, execution }: AppPathProps): string =>
    toRouteURL(RoutePath.PIPELINE_EXECUTION_SETTINGS, { pipeline, execution })

  // @see https://github.com/drone/policy-mgmt/blob/main/web/src/RouteDefinitions.ts
  // for more examples regarding to passing parameters to generate URLs
}
