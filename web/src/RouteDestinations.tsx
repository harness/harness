import React from 'react'
import { Route, Switch, BrowserRouter } from 'react-router-dom'
import { SignIn } from 'pages/SignIn/SignIn'
import { SignUp } from 'pages/SignUp/SignUp'
import Repository from 'pages/Repository/Repository'
import { routes, pathProps } from 'RouteDefinitions'
import RepositoriesListing from 'pages/RepositoriesListing/RepositoriesListing'
import { LayoutWithSideNav, LayoutWithoutSideNav } from 'layouts/layout'
import RepositoryFileEdit from 'pages/RepositoryFileEdit/RepositoryFileEdit'
import RepositoryCommits from 'pages/RepositoryCommits/RepositoryCommits'
import RepositoryBranches from 'pages/RepositoryBranches/RepositoryBranches'
import RepositoryTags from 'pages/RepositoryTags/RepositoryTags'
import Compare from 'pages/Compare/Compare'
import PullRequest from 'pages/PullRequest/PullRequest'
import PullRequests from 'pages/PullRequests/PullRequests'
import WebhookNew from 'pages/WebhookNew/WebhookNew'
import WebhookDetails from 'pages/WebhookDetails/WebhookDetails'
import Webhooks from 'pages/Webhooks/Webhooks'
import RepositorySettings from 'pages/RepositorySettings/RepositorySettings'
import UsersListing from 'pages/UsersListing/UsersListing'
import Home from 'pages/Home/Home'
import UserProfile from 'pages/UserProfile/UserProfile'
import ChangePassword from 'pages/ChangePassword/ChangePassword'
import SpaceAccessControl from 'pages/SpaceAccessControl/SpaceAccessControl'
import SpaceSettings from 'pages/SpaceSettings/SpaceSettings'
import { useStrings } from 'framework/strings'

export const RouteDestinations: React.FC = React.memo(function RouteDestinations() {
  const { getString } = useStrings()
  const repoPath = `${pathProps.space}/${pathProps.repoName}`

  return (
    <BrowserRouter>
      <Switch>
        <Route path={routes.toSignIn()}>
          <LayoutWithoutSideNav title={getString('pageTitle.signin')}>
            <SignIn />
          </LayoutWithoutSideNav>
        </Route>

        <Route path={routes.toRegister()}>
          <LayoutWithoutSideNav title={getString('pageTitle.register')}>
            <SignUp />
          </LayoutWithoutSideNav>
        </Route>

        <Route path={routes.toCODESpaceAccessControl({ space: pathProps.space })} exact>
          <LayoutWithSideNav title={getString('pageTitle.accessControl')}>
            <SpaceAccessControl />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODESpaceSettings({ space: pathProps.space })} exact>
          <LayoutWithSideNav title={getString('pageTitle.spaceSettings')}>
            <SpaceSettings />
          </LayoutWithSideNav>
        </Route>

        <Route
          path={routes.toCODECompare({
            repoPath,
            diffRefs: pathProps.diffRefs
          })}>
          <LayoutWithSideNav title={getString('pageTitle.compare')}>
            <Compare />
          </LayoutWithSideNav>
        </Route>

        <Route path={[routes.toCODEHome()]} exact>
          <LayoutWithSideNav title={getString('pageTitle.home')}>
            <Home />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODEUsers()}>
          <LayoutWithSideNav title={getString('pageTitle.users')}>
            <UsersListing />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODEUserProfile()} exact>
          <LayoutWithSideNav title={getString('pageTitle.userProfile')}>
            <UserProfile />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODEUserChangePassword()} exact>
          <LayoutWithSideNav title={getString('pageTitle.changePassword')}>
            <ChangePassword />
          </LayoutWithSideNav>
        </Route>

        <Route
          path={[
            routes.toCODEPullRequest({
              repoPath,
              pullRequestId: pathProps.pullRequestId,
              pullRequestSection: pathProps.pullRequestSection
            }),
            routes.toCODEPullRequest({
              repoPath,
              pullRequestId: pathProps.pullRequestId
            })
          ]}
          exact>
          <LayoutWithSideNav title={getString('pageTitle.pullRequest')}>
            <PullRequest />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODEPullRequests({ repoPath })} exact>
          <LayoutWithSideNav title={getString('pageTitle.pullRequests')}>
            <PullRequests />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODEWebhookNew({ repoPath })} exact>
          <LayoutWithSideNav title={getString('pageTitle.createWebhook')}>
            <WebhookNew />
          </LayoutWithSideNav>
        </Route>

        <Route
          path={routes.toCODEWebhookDetails({
            repoPath,
            webhookId: pathProps.webhookId
          })}>
          <LayoutWithSideNav title={getString('pageTitle.webhookDetail')}>
            <WebhookDetails />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODEWebhooks({ repoPath })} exact>
          <LayoutWithSideNav title={getString('pageTitle.webhooks')}>
            <Webhooks />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODESettings({ repoPath })} exact>
          <LayoutWithSideNav title={getString('pageTitle.repositorySettings')}>
            <RepositorySettings />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODERepositories({ space: pathProps.space })} exact>
          <LayoutWithSideNav title={getString('pageTitle.repositories')}>
            <RepositoriesListing />
          </LayoutWithSideNav>
        </Route>

        <Route
          path={routes.toCODECommits({
            repoPath,
            commitRef: pathProps.commitRef
          })}>
          <LayoutWithSideNav title={getString('pageTitle.commits')}>
            <RepositoryCommits />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODEBranches({ repoPath })} exact>
          <LayoutWithSideNav title={getString('pageTitle.branches')}>
            <RepositoryBranches />
          </LayoutWithSideNav>
        </Route>

        <Route path={routes.toCODETags({ repoPath })} exact>
          <LayoutWithSideNav title={getString('pageTitle.tags')}>
            <RepositoryTags />
          </LayoutWithSideNav>
        </Route>

        <Route
          path={routes.toCODEFileEdit({
            repoPath,
            gitRef: pathProps.gitRef,
            resourcePath: pathProps.resourcePath
          })}>
          <LayoutWithSideNav title={getString('pageTitle.editFile')}>
            <RepositoryFileEdit />
          </LayoutWithSideNav>
        </Route>

        <Route
          path={[
            routes.toCODERepository({
              repoPath,
              gitRef: pathProps.gitRef,
              resourcePath: pathProps.resourcePath
            }),
            routes.toCODERepository({
              repoPath,
              gitRef: pathProps.gitRef
            }),
            routes.toCODERepository({ repoPath })
          ]}>
          <LayoutWithSideNav title={getString('pageTitle.repository')}>
            <Repository />
          </LayoutWithSideNav>
        </Route>

        <Route
          path={[
            routes.toCODERepository({
              repoPath,
              gitRef: pathProps.gitRef,
              resourcePath: pathProps.resourcePath
            }),
            routes.toCODERepository({
              repoPath,
              gitRef: pathProps.gitRef
            }),
            routes.toCODERepository({ repoPath })
          ]}>
          <LayoutWithSideNav title={getString('pageTitle.repository')}>
            <Repository />
          </LayoutWithSideNav>
        </Route>
      </Switch>
    </BrowserRouter>
  )
})
