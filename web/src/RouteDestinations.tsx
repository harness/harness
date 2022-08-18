/* eslint-disable react/display-name */
import React, { useCallback } from 'react'
import { HashRouter, Route, Switch, Redirect } from 'react-router-dom'
// import { SignInPage } from 'pages/signin/SignInPage'
import { NotFoundPage } from 'pages/404/NotFoundPage'
import { SignIn } from 'pages/SignIn/SignIn'
import { Register } from 'pages/Register/Register'
import { routePath, standaloneRoutePath } from './RouteUtils'
import { RoutePath } from './RouteDefinitions'

export const RouteDestinations: React.FC<{ standalone: boolean }> = React.memo(({ standalone }) => {
  const Destinations: React.FC = useCallback(
    () => (
      <Switch>
        {standalone && (
          <>
            <Route path={routePath(RoutePath.SIGNIN)}>
              <SignIn />
            </Route>
            <Route path={routePath(RoutePath.SIGNUP)}>
              <SignIn />
            </Route>
            <Route path={routePath(RoutePath.REGISTER)}>
              <Register />
            </Route>
          </>
        )}

        <Route path={routePath(RoutePath.POLICY_DASHBOARD)}>
          <h1>Overview</h1>
        </Route>

        <Route path={routePath(RoutePath.POLICY_NEW)}>
          <h1>New</h1>
        </Route>

        <Route path={routePath(RoutePath.POLICY_VIEW)}>
          <h1>View</h1>
        </Route>

        <Route exact path={routePath(RoutePath.POLICY_EDIT)}>
          <h1>Edit</h1>
        </Route>

        <Route path={routePath(RoutePath.POLICY_LISTING)}>
          <h1>Listing</h1>
        </Route>

        <Route exact path={routePath(RoutePath.POLICY_SETS_LISTING)}>
          <h1>Listing 2</h1>
        </Route>

        <Route path={routePath(RoutePath.POLICY_SETS_DETAIL)}>
          <h1>Detail 1</h1>
        </Route>

        <Route path={routePath(RoutePath.POLICY_EVALUATION_DETAIL)}>
          <h1>Detail 2</h1>
        </Route>

        <Route path="/">
          {standalone ? <Redirect to={standaloneRoutePath(RoutePath.POLICY_DASHBOARD)} /> : <NotFoundPage />}
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
})
