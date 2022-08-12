import { toRouteURL } from 'RouteUtils'
import type { AppPathProps } from 'AppProps'

export enum RoutePath {
  SIGNIN = '/signin',
  POLICY_DASHBOARD = '/dashboard',
  POLICY_LISTING = '/policies',
  POLICY_NEW = '/policies/new',
  POLICY_VIEW = '/policies/view/:policyIdentifier',
  //POLICY_EDIT = '/policies/edit/:policyIdentifier',
  POLICY_EDIT= '/policies/edit/:policyIdentifier/:repo?/:branch?',
  POLICY_SETS_LISTING = '/policy-sets',
  POLICY_SETS_DETAIL = '/policy-sets/:policySetIdentifier',
  POLICY_EVALUATIONS_LISTING = '/policy-evaluations',
  POLICY_EVALUATION_DETAIL = '/policy-evaluations/:evaluationId'
}

export default {
  toSignIn: (): string => toRouteURL(RoutePath.SIGNIN),
  toPolicyDashboard: (): string => toRouteURL(RoutePath.POLICY_DASHBOARD),
  toPolicyListing: (): string => toRouteURL(RoutePath.POLICY_LISTING),
  toPolicyNew: (): string => toRouteURL(RoutePath.POLICY_NEW),
  toPolicyView: ({ policyIdentifier }: Required<Pick<AppPathProps, 'policyIdentifier'>>): string => 
    toRouteURL(RoutePath.POLICY_VIEW, { policyIdentifier }),
  toPolicyEdit: ({ policyIdentifier }: Required<Pick<AppPathProps, 'policyIdentifier'>>): string =>
    toRouteURL(RoutePath.POLICY_EDIT, { policyIdentifier }),
  toPolicySets: (): string => toRouteURL(RoutePath.POLICY_SETS_LISTING),
  toPolicyEvaluations: (): string => toRouteURL(RoutePath.POLICY_EVALUATIONS_LISTING),
  toGovernancePolicyDashboard: ({ orgIdentifier, projectIdentifier, module }: AppPathProps) =>
    toRouteURL(RoutePath.POLICY_DASHBOARD, {
      orgIdentifier,
      projectIdentifier,
      module
    }),
  toGovernancePolicyListing: ({ orgIdentifier, projectIdentifier, module }: AppPathProps) =>
    toRouteURL(RoutePath.POLICY_LISTING, {
      orgIdentifier,
      projectIdentifier,
      module
    }),
  toGovernanceNewPolicy: ({ orgIdentifier, projectIdentifier, module }: AppPathProps) =>
    toRouteURL(RoutePath.POLICY_NEW, {
      orgIdentifier,
      projectIdentifier,
      module
    }),
  toGovernanceEditPolicy: ({
    orgIdentifier,
    projectIdentifier,
    policyIdentifier,
    module,
    repo,
    branch
  }: RequireField<AppPathProps, 'policyIdentifier'>) =>
    toRouteURL(RoutePath.POLICY_EDIT, {
      orgIdentifier,
      projectIdentifier,
      policyIdentifier,
      module,
      repo,
      branch
    }),
  toGovernanceViewPolicy: ({
    orgIdentifier,
    projectIdentifier,
    policyIdentifier,
    module
  }: RequireField<AppPathProps, 'policyIdentifier'>) =>
    toRouteURL(RoutePath.POLICY_VIEW, {
      orgIdentifier,
      projectIdentifier,
      policyIdentifier,
      module
    }),
  toGovernancePolicySetsListing: ({ orgIdentifier, projectIdentifier, module }: AppPathProps) =>
    toRouteURL(RoutePath.POLICY_SETS_LISTING, {
      orgIdentifier,
      projectIdentifier,
      module
    }),
  toGovernancePolicySetDetail: ({ orgIdentifier, projectIdentifier, policySetIdentifier, module }: AppPathProps) =>
    toRouteURL(RoutePath.POLICY_SETS_DETAIL, {
      orgIdentifier,
      projectIdentifier,
      module,
      policySetIdentifier
    }),
  toGovernanceEvaluationsListing: ({ orgIdentifier, projectIdentifier, module }: AppPathProps) =>
    toRouteURL(RoutePath.POLICY_EVALUATIONS_LISTING, {
      orgIdentifier,
      projectIdentifier,
      module
    }),
  toGovernanceEvaluationDetail: ({ orgIdentifier, projectIdentifier, evaluationId, module }: AppPathProps) =>
    toRouteURL(RoutePath.POLICY_EVALUATION_DETAIL, {
      orgIdentifier,
      projectIdentifier,
      module,
      evaluationId
    })
}
