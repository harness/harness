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
  RedirectPageQueryParams,
  RepositoryDetailsPathParams,
  RepositoryDetailsTabPathParams,
  RepositoryWebhookDetailsPathParams,
  RepositoryWebhookDetailsTabPathParams,
  VersionDetailsPathParams,
  VersionDetailsTabPathParams
} from './types'

export interface ARRouteDefinitionsReturn {
  toAR: () => string
  toARRedirect: (params?: RedirectPageQueryParams) => string
  toARRepositories: () => string
  toARRepositoryDetails: (params: RepositoryDetailsPathParams) => string
  toARRepositoryDetailsTab: (params: RepositoryDetailsTabPathParams) => string
  toARArtifacts: () => string
  toARArtifactDetails: (params: ArtifactDetailsPathParams) => string
  toARVersionDetails: (params: VersionDetailsPathParams) => string
  toARVersionDetailsTab: (params: VersionDetailsTabPathParams) => string
  toARRepositoryWebhookDetails: (params: RepositoryWebhookDetailsPathParams) => string
  toARRepositoryWebhookDetailsTab: (params: RepositoryWebhookDetailsTabPathParams) => string
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
        versionDetailsTab: defaultTo(params?.versionDetailsTab, '')
      })
      return `/redirect?${queryParams.toString()}`
    }
    return '/redirect'
  },
  toARRepositories: () => '/registries',
  toARRepositoryDetails: params => `/registries/${params?.repositoryIdentifier}`,
  toARRepositoryDetailsTab: params => `/registries/${params?.repositoryIdentifier}/${params?.tab}`,
  toARArtifacts: () => '/artifacts',
  toARArtifactDetails: params => `/registries/${params?.repositoryIdentifier}/artifacts/${params?.artifactIdentifier}`,
  toARVersionDetails: params =>
    `/registries/${params?.repositoryIdentifier}/artifacts/${params?.artifactIdentifier}/versions/${params?.versionIdentifier}`,
  toARVersionDetailsTab: params => {
    let route = `/registries/${params?.repositoryIdentifier}/artifacts/${params?.artifactIdentifier}/versions/${params?.versionIdentifier}`
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
  },
  toARRepositoryWebhookDetails: params =>
    `/registries/${params?.repositoryIdentifier}/webhooks/${params?.webhookIdentifier}`,
  toARRepositoryWebhookDetailsTab: params =>
    `/registries/${params?.repositoryIdentifier}/webhooks/${params?.webhookIdentifier}/${params.tab}`
}
