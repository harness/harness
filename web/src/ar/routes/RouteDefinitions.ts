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

import { defaultTo, isEmpty } from 'lodash-es'

import type {
  ArtifactDetailsPathParams,
  ManageRegistriesTabPathParams,
  RedirectPageQueryParams,
  RepositoryDetailsPathParams,
  RepositoryDetailsTabPathParams,
  RepositoryWebhookDetailsPathParams,
  RepositoryWebhookDetailsTabPathParams,
  VersionDetailsPathParams,
  VersionDetailsTabPathParams
} from './types'
import { IRouteOptions, routeDefinitionWithMode } from './utils'

export interface ARRouteDefinitionsReturn {
  toAR: () => string
  toARRedirect: (params?: RedirectPageQueryParams, options?: IRouteOptions) => string
  toARManageRegistries: (_?: unknown, options?: IRouteOptions) => string
  toARManageRegistriesTab: (params: ManageRegistriesTabPathParams, options?: IRouteOptions) => string
  toARRepositories: (_?: unknown, options?: IRouteOptions) => string
  toARRepositoryDetails: (params: RepositoryDetailsPathParams, options?: IRouteOptions) => string
  toARRepositoryDetailsTab: (params: RepositoryDetailsTabPathParams, options?: IRouteOptions) => string
  toARArtifacts: (_?: unknown, options?: IRouteOptions) => string
  toARArtifactDetails: (params: ArtifactDetailsPathParams, options?: IRouteOptions) => string
  toARVersionDetails: (params: VersionDetailsPathParams, options?: IRouteOptions) => string
  toARVersionDetailsTab: (params: VersionDetailsTabPathParams, options?: IRouteOptions) => string
  toARRepositoryWebhookDetails: (params: RepositoryWebhookDetailsPathParams, options?: IRouteOptions) => string
  toARRepositoryWebhookDetailsTab: (params: RepositoryWebhookDetailsTabPathParams, options?: IRouteOptions) => string
}

export const routeDefinitions: ARRouteDefinitionsReturn = {
  toAR: () => '/',
  toARRedirect: params => {
    if (!isEmpty(params)) {
      const queryParams = new URLSearchParams({
        packageType: defaultTo(params?.packageType, ''),
        registryId: defaultTo(params?.registryId, ''),
        artifactId: defaultTo(params?.artifactId, ''),
        versionId: defaultTo(params?.versionId, ''),
        versionDetailsTab: defaultTo(params?.versionDetailsTab, ''),
        artifactType: defaultTo(params?.artifactType, '')
      })
      return `/redirect?${queryParams.toString()}`
    }
    return '/redirect'
  },
  toARManageRegistries: routeDefinitionWithMode(() => '/manage'),
  toARManageRegistriesTab: routeDefinitionWithMode(params => `/manage/${params.tab}`),
  toARRepositories: routeDefinitionWithMode(() => '/registries'),
  toARRepositoryDetails: routeDefinitionWithMode(params => `/registries/${params?.repositoryIdentifier}`),
  toARRepositoryDetailsTab: routeDefinitionWithMode(
    params => `/registries/${params?.repositoryIdentifier}/${params.tab}`
  ),
  toARArtifacts: routeDefinitionWithMode(() => '/artifacts'),
  toARArtifactDetails: routeDefinitionWithMode(
    params => `/registries/${params?.repositoryIdentifier}/${params?.artifactType}/${params?.artifactIdentifier}`
  ),
  toARVersionDetails: routeDefinitionWithMode(
    params =>
      `/registries/${params?.repositoryIdentifier}/${params?.artifactType}/${params?.artifactIdentifier}/versions/${params?.versionIdentifier}`
  ),
  toARVersionDetailsTab: routeDefinitionWithMode(params => {
    let route = `/registries/${params?.repositoryIdentifier}/${params?.artifactType}/${params?.artifactIdentifier}/versions/${params?.versionIdentifier}`
    if (params.orgIdentifier) route += `/orgs/${params.orgIdentifier}`
    if (params.projectIdentifier) route += `/projects/${params.projectIdentifier}`
    if (params.sourceId && params.artifactId) {
      route += `/artifact-sources/${params.sourceId}/artifacts/${params.artifactId}`
    }
    if (params.pipelineIdentifier && params.executionIdentifier) {
      route += `/pipelines/${params.pipelineIdentifier}/executions/${params.executionIdentifier}`
    }
    route += `/${params.versionTab}`
    return route
  }),
  toARRepositoryWebhookDetails: routeDefinitionWithMode(
    params => `/registries/${params?.repositoryIdentifier}/webhooks/${params?.webhookIdentifier}`
  ),
  toARRepositoryWebhookDetailsTab: routeDefinitionWithMode(
    params => `/registries/${params?.repositoryIdentifier}/webhooks/${params?.webhookIdentifier}/${params.tab}`
  )
}
