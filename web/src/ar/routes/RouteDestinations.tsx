/*
 * Copyright 2024 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react'
import { Redirect, Switch } from 'react-router-dom'

import { Parent } from '@ar/common/types'
import { useAppStore, useGetRepositoryListViewType, useRoutes } from '@ar/hooks'
import RedirectPage from '@ar/pages/redirect-page/RedirectPage'
import { RepositoryListViewTypeEnum } from '@ar/contexts/AppStoreContext'
import type { WebhookDetailsTab } from '@ar/pages/webhook-details/constants'
import type { LocalArtifactType, RepositoryDetailsTab } from '@ar/pages/repository-details/constants'
import type { ManageRegistriesDetailsTab } from '@ar/pages/manage-registries/constants'

import type {
  ArtifactDetailsPathParams,
  ManageRegistriesTabPathParams,
  RepositoryDetailsPathParams,
  RepositoryDetailsTabPathParams,
  RepositoryWebhookDetailsPathParams,
  RepositoryWebhookDetailsTabPathParams,
  VersionDetailsPathParams,
  VersionDetailsTabPathParams
} from './types'

const RepositoryListPage = React.lazy(() => import('@ar/pages/repository-list/RepositoryListPage'))
const ManageRegistriesPage = React.lazy(() => import('@ar/pages/manage-registries/ManageRegistriesPage'))
const RepositoryListTreeViewPage = React.lazy(() => import('@ar/pages/repository-list/RepositoryListTreeViewPage'))
const RepositoryDetailsPage = React.lazy(() => import('@ar/pages/repository-details/RepositoryDetailsPage'))
const ArtifactListPage = React.lazy(() => import('@ar/pages/artifact-list/ArtifactListPage'))
const ArtifactDetailsPage = React.lazy(() => import('@ar/pages/artifact-details/ArtifactDetailsPage'))
const VersionDetailsPage = React.lazy(() => import('@ar/pages/version-details/VersionDetailsPage'))
const OSSVersionDetailsPage = React.lazy(() => import('@ar/pages/version-details/OSSVersionDetailsPage'))
const RouteProvider = React.lazy(() => import('@ar/components/RouteProvider/RouteProvider'))
const WebhookDetailsPage = React.lazy(() => import('@ar/pages/webhook-details/WebhookDetailsPage'))

export const manageRegistriesTabPathProps: ManageRegistriesTabPathParams = {
  tab: ':tab' as ManageRegistriesDetailsTab
}

export const repositoryDetailsPathProps: RepositoryDetailsPathParams = {
  repositoryIdentifier: ':repositoryIdentifier'
}

export const repositoryDetailsTabPathProps: RepositoryDetailsTabPathParams = {
  ...repositoryDetailsPathProps,
  tab: ':tab' as RepositoryDetailsTab
}

export const artifactDetailsPathProps: ArtifactDetailsPathParams = {
  ...repositoryDetailsPathProps,
  artifactIdentifier: ':artifactIdentifier*',
  artifactType: ':artifactType' as LocalArtifactType
}

export const versionDetailsPathParams: VersionDetailsPathParams = {
  ...artifactDetailsPathProps,
  versionIdentifier: ':versionIdentifier'
}

export const versionDetailsTabPathParams: VersionDetailsTabPathParams = {
  ...versionDetailsPathParams,
  versionTab: ':versionTab'
}

export const versionDetailsTabWithProjectPathParams: VersionDetailsTabPathParams = {
  ...versionDetailsTabPathParams,
  projectIdentifier: ':projectIdentifier'
}

export const versionDetailsTabWithOrgAndProjectPathParams: VersionDetailsTabPathParams = {
  ...versionDetailsTabPathParams,
  orgIdentifier: ':orgIdentifier',
  projectIdentifier: ':projectIdentifier'
}

export const versionDetailsTabWithPipelineDetailsPathParams: VersionDetailsTabPathParams = {
  ...versionDetailsTabPathParams,
  pipelineIdentifier: ':pipelineIdentifier',
  executionIdentifier: ':executionIdentifier'
}

export const versionDetailsTabWithSSCADetailsPathParams: VersionDetailsTabPathParams = {
  ...versionDetailsTabPathParams,
  sourceId: ':sourceId',
  artifactId: ':artifactId'
}

export const repositoryWebhookDetailsPathParams: RepositoryWebhookDetailsPathParams = {
  ...repositoryDetailsPathProps,
  webhookIdentifier: ':webhookIdentifier'
}

export const repositoryWebhookDetailsTabPathParams: RepositoryWebhookDetailsTabPathParams = {
  ...repositoryWebhookDetailsPathParams,
  tab: ':tab' as WebhookDetailsTab
}

const RouteDestinations = (): JSX.Element => {
  const routes = useRoutes(true)
  const { parent } = useAppStore()
  const repositoryListViewType = useGetRepositoryListViewType()
  const shouldUseSeperateVersionDetailsRoute =
    parent === Parent.Enterprise || repositoryListViewType === RepositoryListViewTypeEnum.DIRECTORY
  return (
    <Switch>
      <RouteProvider exact path={routes.toAR()}>
        <Redirect to={routes.toARRepositories()} />
      </RouteProvider>
      <RouteProvider exact path={routes.toARRedirect()}>
        <RedirectPage />
      </RouteProvider>
      <RouteProvider exact path={routes.toARArtifacts()}>
        <ArtifactListPage />
      </RouteProvider>
      <RouteProvider path={routes.toARManageRegistries()}>
        <ManageRegistriesPage />
      </RouteProvider>
      {repositoryListViewType === RepositoryListViewTypeEnum.DIRECTORY && (
        <RouteProvider path={routes.toARRepositories()}>
          <RepositoryListTreeViewPage />
        </RouteProvider>
      )}
      {repositoryListViewType === RepositoryListViewTypeEnum.LIST && (
        <RouteProvider exact path={routes.toARRepositories()}>
          <RepositoryListPage />
        </RouteProvider>
      )}
      <RouteProvider path={routes.toARRepositoryWebhookDetails({ ...repositoryWebhookDetailsPathParams })}>
        <WebhookDetailsPage />
      </RouteProvider>
      <RouteProvider
        exact
        path={[
          routes.toARRepositoryDetails({ ...repositoryDetailsPathProps }),
          routes.toARRepositoryDetailsTab({ ...repositoryDetailsTabPathProps })
        ]}>
        <RepositoryDetailsPage />
      </RouteProvider>
      {shouldUseSeperateVersionDetailsRoute && (
        <RouteProvider
          exact
          path={[
            routes.toARVersionDetails({ ...versionDetailsPathParams }),
            routes.toARVersionDetailsTab({ ...versionDetailsTabPathParams }),
            // with project and org data
            routes.toARVersionDetailsTab({
              ...versionDetailsTabWithOrgAndProjectPathParams
            }),
            // with project data
            routes.toARVersionDetailsTab({
              ...versionDetailsTabWithProjectPathParams
            }),
            // ssca with pipeline data
            routes.toARVersionDetailsTab({ ...versionDetailsTabWithSSCADetailsPathParams }),
            // ssca with project and pipeline data
            routes.toARVersionDetailsTab({
              ...versionDetailsTabWithSSCADetailsPathParams,
              ...versionDetailsTabWithProjectPathParams
            }),
            // ssca with org, project and pipeline data
            routes.toARVersionDetailsTab({
              ...versionDetailsTabWithSSCADetailsPathParams,
              ...versionDetailsTabWithOrgAndProjectPathParams
            }),
            // sto with pipeline data
            routes.toARVersionDetailsTab({ ...versionDetailsTabWithPipelineDetailsPathParams }),
            // sto with project and pipeline data
            routes.toARVersionDetailsTab({
              ...versionDetailsTabWithPipelineDetailsPathParams,
              ...versionDetailsTabWithProjectPathParams
            }),
            // sto with org, project and pipeline data
            routes.toARVersionDetailsTab({
              ...versionDetailsTabWithPipelineDetailsPathParams,
              ...versionDetailsTabWithOrgAndProjectPathParams
            })
          ]}>
          <VersionDetailsPage />
        </RouteProvider>
      )}
      {/* IF Enterprise then will use different route for version details page
       * IF repositoryListViewType = DIRECTORY then will use different route for version details page
       * IF OSS then will use version details as sub route for artifact details page
       */}
      <RouteProvider
        exact={shouldUseSeperateVersionDetailsRoute}
        path={routes.toARArtifactDetails({ ...artifactDetailsPathProps })}>
        <>
          <ArtifactDetailsPage />
          {parent === Parent.OSS && (
            <Switch>
              <RouteProvider path={routes.toARVersionDetails({ ...versionDetailsPathParams })}>
                <OSSVersionDetailsPage />
              </RouteProvider>
            </Switch>
          )}
        </>
      </RouteProvider>
    </Switch>
  )
}

export default RouteDestinations
