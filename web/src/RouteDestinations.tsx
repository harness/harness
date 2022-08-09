import React from 'react'
import { HashRouter, Route, Switch } from 'react-router-dom'
import type { AppProps } from 'AppProps'
import { NotFoundPage } from 'pages/404/NotFoundPage'
import { routePath } from 'RouteUtils'
import { RoutePath } from 'RouteDefinitions'

import { Login } from './pages/Login/Login'
import { Home } from './pages/Pipelines/Pipelines'
import { Executions } from './pages/Executions/Executions'
import { ExecutionSettings } from './pages/Execution/Settings'
import { PipelineSettings } from './pages/Pipeline/Settings'
import { Account } from './pages/Account/Account'
import { SideNav } from './components/SideNav/SideNav'

export const RouteDestinations: React.FC<Pick<AppProps, 'standalone'>> = ({ standalone }) => {
  // TODO: Add a generic Auth Wrapper

  const Destinations: React.FC = () => (
    <Switch>
      {standalone && (
        <Route path={routePath(RoutePath.REGISTER)}>
          <Login />
        </Route>
      )}
      {standalone && (
        <Route path={routePath(RoutePath.LOGIN)}>
          <Login />
        </Route>
      )}

      <Route exact path={routePath(RoutePath.PIPELINES)}>
        <SideNav>
          <Home />
        </SideNav>
      </Route>

      <Route exact path={routePath(RoutePath.PIPELINE)}>
        <SideNav>
          <Executions />
        </SideNav>
      </Route>

      <Route exact path={routePath(RoutePath.PIPELINE_SETTINGS)}>
        <SideNav>
          <PipelineSettings />
        </SideNav>
      </Route>

      <Route exact path={routePath(RoutePath.PIPELINE_EXECUTION_SETTINGS)}>
        <SideNav>
          <ExecutionSettings />
        </SideNav>
      </Route>

      <Route exact path={routePath(RoutePath.ACCOUNT)}>
        <SideNav>
          <Account />
        </SideNav>
      </Route>

      <Route path="/">
        <NotFoundPage />
      </Route>
    </Switch>
  )

  return standalone ? (
    <HashRouter>
      <Destinations />
    </HashRouter>
  ) : (
    <Destinations />
  )
}
