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
        <Route path={routes.toCODERepositoriesListing({ space: pathProps.space })} exact>
          <RepositoriesListing />
        </Route>
        <Route
          path={[
            routes.toCODERepository({
              repoPath: `${pathProps.space}/${pathProps.repoName}`,
              gitRef: pathProps.gitRef,
              resourcePath: pathProps.resourcePath
            }),
            routes.toCODERepository({
              repoPath: `${pathProps.space}/${pathProps.repoName}`,
              gitRef: pathProps.gitRef
            })
          ]}>
          <Repository />
        </Route>
      </Switch>
    </HashRouter>
  )
})
