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
import { useAppStore, useRoutes } from '@ar/hooks'
import type {
  ArtifactDetailsPathParams,
  RepositoryDetailsPathParams,
  VersionDetailsPathParams,
  VersionDetailsTabPathParams
} from './types'

import '@ar/pages/version-details/VersionFactory'
import '@ar/pages/repository-details/RepositoryFactory'

const RepositoryListPage = React.lazy(() => import('@ar/pages/repository-list/RepositoryListPage'))
const RepositoryDetailsPage = React.lazy(() => import('@ar/pages/repository-details/RepositoryDetailsPage'))
const ArtifactListPage = React.lazy(() => import('@ar/pages/artifact-list/ArtifactListPage'))
const ArtifactDetailsPage = React.lazy(() => import('@ar/pages/artifact-details/ArtifactDetailsPage'))
const VersionDetailsPage = React.lazy(() => import('@ar/pages/version-details/VersionDetailsPage'))
const OSSVersionDetailsPage = React.lazy(() => import('@ar/pages/version-details/OSSVersionDetailsPage'))
const RouteProvider = React.lazy(() => import('@ar/components/RouteProvider/RouteProvider'))

const repositoryDetailsPathProps: RepositoryDetailsPathParams = {
  repositoryIdentifier: ':repositoryIdentifier'
}

const artifactDetailsPathProps: ArtifactDetailsPathParams = {
  ...repositoryDetailsPathProps,
  artifactIdentifier: ':artifactIdentifier'
}

export const versionDetailsPathParams: VersionDetailsPathParams = {
  ...artifactDetailsPathProps,
  versionIdentifier: ':versionIdentifier'
}

export const versionDetailsTabPathParams: VersionDetailsTabPathParams = {
  ...versionDetailsPathParams,
  versionTab: ':versionTab'
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

const RouteDestinations = (): JSX.Element => {
  const routes = useRoutes(true)
  const { parent } = useAppStore()
  return (
    <Switch>
      <RouteProvider exact path={routes.toAR()}>
        <Redirect to={routes.toARRepositories()} />
      </RouteProvider>
      <RouteProvider exact path={routes.toARRepositories()}>
        <RepositoryListPage />
      </RouteProvider>
      <RouteProvider exact path={routes.toARRepositoryDetails({ ...repositoryDetailsPathProps })}>
        <RepositoryDetailsPage />
      </RouteProvider>
      <RouteProvider exact path={routes.toARArtifacts()}>
        <ArtifactListPage />
      </RouteProvider>
      <RouteProvider
        exact={parent === Parent.Enterprise}
        path={routes.toARArtifactDetails({ ...artifactDetailsPathProps })}>
        <>
          <ArtifactDetailsPage />
          {parent === Parent.OSS && (
            <Switch>
              <RouteProvider exact path={routes.toARVersionDetails({ ...versionDetailsPathParams })}>
                <OSSVersionDetailsPage />
              </RouteProvider>
            </Switch>
          )}
        </>
      </RouteProvider>
      {parent === Parent.Enterprise && (
        <RouteProvider path={routes.toARVersionDetails({ ...versionDetailsPathParams })}>
          <VersionDetailsPage />
        </RouteProvider>
      )}
    </Switch>
  )
}

export default RouteDestinations
