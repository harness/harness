import React from 'react'
import { HashRouter, Route, Switch } from 'react-router-dom'
import { SignIn } from 'pages/SignIn/SignIn'
import { SignUp } from 'pages/SignUp/SignUp'
import routes, { pathProps } from 'RouteDefinitions'
import Welcome from 'views/Welcome/Welcome'
import Repos from 'views/Repos/Repos'
import RepoSettings from 'views/RepoSettings/RepoSettings'
import RepoFiles from 'views/RepoFiles/RepoFiles'
import RepoFileDetails from 'views/RepoFileDetails/RepoFileDetails'
import RepoCommits from 'views/RepoCommits/RepoCommits'
import RepoCommitDetails from 'views/RepoCommitDetails/RepoCommitDetails'
import RepoPullRequests from 'views/RepoPullRequests/RepoPullRequests'
import RepoPullRequestDetails from 'views/RepoPullRequestDetails/RepoPullRequestDetails'

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
        <Route path={routes.toSCMHome(pathProps)}>
          <Welcome />
        </Route>
        <Route path={routes.toSCMRepos(pathProps)} exact>
          <Repos />
        </Route>
        <Route path={routes.toSCMRepoSettings(pathProps)}>
          <RepoSettings />
        </Route>
        <Route path={routes.toSCMFiles(pathProps)}>
          <RepoFiles />
        </Route>
        <Route path={routes.toSCMFileDetails(pathProps)}>
          <RepoFileDetails />
        </Route>
        <Route path={routes.toSCMCommits(pathProps)}>
          <RepoCommits />
        </Route>
        <Route path={routes.toSCMCommitDetails(pathProps)}>
          <RepoCommitDetails />
        </Route>
        <Route path={routes.toSCMPullRequests(pathProps)}>
          <RepoPullRequests />
        </Route>
        <Route path={routes.toSCMPullRequestDetails(pathProps)}>
          <RepoPullRequestDetails />
        </Route>
      </Switch>
    </HashRouter>
  )
})
