import React from 'react'
import { Route, Switch, BrowserRouter } from 'react-router-dom'
import { SignIn } from 'pages/SignIn/SignIn'
import { SignUp } from 'pages/SignUp/SignUp'
import Repository from 'pages/Repository/Repository'
import { routes, pathProps } from 'RouteDefinitions'
import RepositoriesListing from 'pages/RepositoriesListing/RepositoriesListing'
import Spaces from 'pages/Spaces/Spaces'

import { LayoutWithSideMenu, LayoutWithSideNav } from 'layouts/layout'
import Settings from 'pages/Settings/Settings'
import { GlobalSettingsMenu } from 'layouts/menu/GlobalSettingsMenu'
import { RepositoryMenu } from 'layouts/menu/RepositoryMenu'
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
import Home from 'pages/Home/Home'

export const RouteDestinations: React.FC = React.memo(function RouteDestinations() {
  const repoPath = `${pathProps.space}/${pathProps.repoName}`

  return (
    <BrowserRouter>
      <Switch>
        <Route path={routes.toSignIn()}>
          <SignIn />
        </Route>

        <Route path={routes.toRegister()}>
          <SignUp />
        </Route>

        <Route path={routes.toCODESpaces()} exact>
          <LayoutWithSideNav>
            <Spaces />
          </LayoutWithSideNav>
        </Route>

        <Route
          path={routes.toCODECompare({
            repoPath,
            diffRefs: pathProps.diffRefs
          })}>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <Compare />
          </LayoutWithSideMenu>
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
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <PullRequest />
          </LayoutWithSideMenu>
        </Route>

        <Route path={routes.toCODEPullRequests({ repoPath })} exact>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <PullRequests />
          </LayoutWithSideMenu>
        </Route>

        <Route path={routes.toCODEWebhookNew({ repoPath })} exact>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <WebhookNew />
          </LayoutWithSideMenu>
        </Route>

        <Route
          path={routes.toCODEWebhookDetails({
            repoPath,
            webhookId: pathProps.webhookId
          })}>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <WebhookDetails />
          </LayoutWithSideMenu>
        </Route>

        <Route path={routes.toCODEWebhooks({ repoPath })} exact>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <Webhooks />
          </LayoutWithSideMenu>
        </Route>

        <Route path={routes.toCODESettings({ repoPath })} exact>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <RepositorySettings />
          </LayoutWithSideMenu>
        </Route>

        <Route path={routes.toCODERepositories({ space: pathProps.space })} exact>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <RepositoriesListing />
          </LayoutWithSideMenu>
        </Route>

        <Route
          path={routes.toCODECommits({
            repoPath,
            commitRef: pathProps.commitRef
          })}>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <RepositoryCommits />
          </LayoutWithSideMenu>
        </Route>

        <Route path={routes.toCODEBranches({ repoPath })} exact>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <RepositoryBranches />
          </LayoutWithSideMenu>
        </Route>

        <Route path={routes.toCODETags({ repoPath })} exact>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <RepositoryTags />
          </LayoutWithSideMenu>
        </Route>

        <Route
          path={routes.toCODEFileEdit({
            repoPath,
            gitRef: pathProps.gitRef,
            resourcePath: pathProps.resourcePath
          })}>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <RepositoryFileEdit />
          </LayoutWithSideMenu>
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
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <Repository />
          </LayoutWithSideMenu>
        </Route>

        <Route
          path={routes.toCODEFileEdit({
            repoPath,
            gitRef: pathProps.gitRef,
            resourcePath: pathProps.resourcePath
          })}>
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <RepositoryFileEdit />
          </LayoutWithSideMenu>
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
          <LayoutWithSideMenu menu={<RepositoryMenu />}>
            <Repository />
          </LayoutWithSideMenu>
        </Route>

        <Route path={[routes.toCODEGlobalSettings()]} exact>
          <LayoutWithSideMenu menu={<GlobalSettingsMenu />}>
            <Settings />
          </LayoutWithSideMenu>
        </Route>

        <Route path={[routes.toCODEHome()]} exact>
          <LayoutWithSideNav>
            <Home />
          </LayoutWithSideNav>
        </Route>
      </Switch>
    </BrowserRouter>
  )
})
