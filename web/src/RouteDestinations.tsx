/* eslint-disable react/display-name */
import React, { useCallback } from 'react'
import { HashRouter, Route, Switch, Redirect } from 'react-router-dom'
import { SignInPage } from 'pages/signin/SignInPage'
import { NotFoundPage } from 'pages/404/NotFoundPage'
import { routePath, standaloneRoutePath } from './RouteUtils'
import { RoutePath } from './RouteDefinitions'
import PolicyControlPage from './pages/PolicyControl/PolicyControlPage'
import Policies from './pages/Policies/Policies'
import PolicyDashboard from './pages/PolicyDashboard/PolicyDashboard'
import PolicySets from './pages/PolicySets/PolicySets'
import PolicyEvaluations from './pages/PolicyEvaluations/PolicyEvaluations'
import { EditPolicy } from './pages/EditPolicy/EditPolicy'
import { ViewPolicy } from './pages/ViewPolicy/ViewPolicy'
import { PolicySetDetail } from './pages/PolicySetDetail/PolicySetDetail'
import { EvaluationDetail } from './pages/EvaluationDetail/EvaluationDetail'

export const RouteDestinations: React.FC<{ standalone: boolean }> = React.memo(
  ({ standalone }) => {
    // TODO: Add Auth wrapper

    const Destinations: React.FC = useCallback(
      () => (
        <Switch>
          {standalone && (
            <Route path={routePath(RoutePath.SIGNIN)}>
              <SignInPage />
            </Route>
          )}

          <Route path={routePath(RoutePath.POLICY_DASHBOARD)}>
            <PolicyControlPage titleKey="overview">
              <PolicyDashboard />
            </PolicyControlPage>
          </Route>

          <Route path={routePath(RoutePath.POLICY_NEW)}>
            <PolicyControlPage titleKey="common.policy.newPolicy">
              <EditPolicy />
            </PolicyControlPage>
          </Route>

          <Route path={routePath(RoutePath.POLICY_VIEW)}>
            <PolicyControlPage titleKey="governance.viewPolicy">
              <ViewPolicy />
            </PolicyControlPage>
          </Route>

          <Route exact path={routePath(RoutePath.POLICY_EDIT)}>
            <PolicyControlPage titleKey="governance.editPolicy">
              <EditPolicy />
            </PolicyControlPage>
          </Route>

          <Route path={routePath(RoutePath.POLICY_LISTING)}>
            <PolicyControlPage titleKey="common.policies">
              <Policies />
            </PolicyControlPage>
          </Route>

          <Route exact path={routePath(RoutePath.POLICY_SETS_LISTING)}>
            <PolicyControlPage titleKey="common.policy.policysets">
              <PolicySets />
            </PolicyControlPage>
          </Route>

          <Route path={routePath(RoutePath.POLICY_SETS_DETAIL)}>
            <PolicyControlPage titleKey="common.policy.policysets">
              <PolicySetDetail />
            </PolicyControlPage>
          </Route>

          <Route path={routePath(RoutePath.POLICY_EVALUATION_DETAIL)}>
            <PolicyControlPage titleKey="governance.evaluations">
              <EvaluationDetail />
            </PolicyControlPage>
          </Route>

          <Route path={routePath(RoutePath.POLICY_EVALUATIONS_LISTING)}>
            <PolicyControlPage titleKey="governance.evaluations">
              <PolicyEvaluations />
            </PolicyControlPage>
          </Route>

          <Route path="/">
            {standalone ? (
              <Redirect to={standaloneRoutePath(RoutePath.POLICY_DASHBOARD)} />
            ) : (
              <NotFoundPage />
            )}
          </Route>
        </Switch>
      ),
      [standalone]
    )

    return standalone ? (
      <HashRouter>
        <Destinations />
      </HashRouter>
    ) : (
      <Destinations />
    )
  }
)
