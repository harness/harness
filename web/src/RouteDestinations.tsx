import React from 'react'
import { HashRouter, Route, Switch } from 'react-router-dom'
import { SignIn } from 'pages/SignIn/SignIn'
import { SignUp } from 'pages/SignUp/SignUp'
import Repository from 'pages/Repository/Repository'
import { routes, pathProps } from 'RouteDefinitions'
import RepositoriesListing from 'pages/RepositoriesListing/RepositoriesListing'

export const RouteDestinations: React.FC = React.memo(function RouteDestinations() {
  return (
    <HashRouter>
      <Switch>
        <Route path={routes.toSignIn()}>
          <SignIn />
        </Route>
        <Route path={routes.toSignUp()}>
          <SignUp />
        </Route>
        <Route path={routes.toSCMRepositoriesListing(pathProps)} exact>
          <RepositoriesListing />
        </Route>
        <Route path={routes.toSCMRepository(pathProps)}>
          <Repository />
        </Route>
      </Switch>
    </HashRouter>
  )
})
