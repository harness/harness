import React, { useCallback } from 'react'
import { HashRouter, Route, Switch, Redirect } from 'react-router-dom'
import { NotFoundPage } from 'pages/404/NotFoundPage'
import { SignIn } from 'pages/SignIn/SignIn'
import { SignUp } from 'pages/SignUp/SignUp'
import { routePath } from './RouteUtils'
import { RoutePath } from './RouteDefinitions'

export const RouteDestinations: React.FC<{ standalone: boolean }> = React.memo(function RouteDestinations({
  standalone
}) {
  const Destinations: React.FC = useCallback(
    () => (
      <Switch>
        {standalone && (
          <Route path={routePath(standalone, RoutePath.SIGNIN)}>
            <SignIn />
          </Route>
        )}

        {standalone && (
          <Route path={routePath(standalone, RoutePath.SIGNUP)}>
            <SignUp />
          </Route>
        )}

        <Route path={routePath(standalone, RoutePath.DASHBOARD)}>
          <h1>DASHBOARD</h1>
        </Route>

        <Route path={routePath(standalone, RoutePath.TEST_PAGE1)}>
          <h1>TEST_PAGE1</h1>
        </Route>

        <Route path={routePath(standalone, RoutePath.TEST_PAGE2)}>
          <h1>TEST_PAGE2</h1>
        </Route>

        <Route path="/">
          {standalone ? <Redirect to={routePath(standalone, RoutePath.DASHBOARD) as string} /> : <NotFoundPage />}
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
