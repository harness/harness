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

import { routeDefinitionWithMode } from '@ar/routes/utils'
import type { ARRouteDefinitionsReturn } from '@ar/routes/RouteDefinitions'

export default function getARRouteDefinitions(routeParams: Record<string, string>): ARRouteDefinitionsReturn {
  return {
    // anything random, as this route will not be used in gitness
    toAR: () => '/ar',
    toARRedirect: params => {
      if (!isEmpty(params)) {
        const queryParams = new URLSearchParams({
          packageType: defaultTo(params?.packageType, ''),
          registryId: defaultTo(params?.registryId, ''),
          artifactId: defaultTo(params?.artifactId, ''),
          versionId: defaultTo(params?.versionId, ''),
          versionDetailsTab: defaultTo(params?.versionDetailsTab, ''),
          artifactType: defaultTo(params?.artifactType, ''),
          tag: defaultTo(params?.tag, '')
        })
        return `/redirect?${queryParams.toString()}`
      }
      return '/redirect'
    },
    toARManageRegistries: () => '/manage-registries',
    toARManageRegistriesTab: params => `/manage-registries/${params?.tab}`,
    toARRepositories: routeDefinitionWithMode(() => '/'),
    toARRepositoryDetails: routeDefinitionWithMode(params => `/${params?.repositoryIdentifier}`),
    toARRepositoryDetailsTab: routeDefinitionWithMode(params => `/${params?.repositoryIdentifier}/${params?.tab}`),
    toARArtifacts: routeDefinitionWithMode(() => `/${routeParams?.repositoryIdentifier}`),
    toARArtifactDetails: routeDefinitionWithMode(
      params => `/${params?.repositoryIdentifier}/${params?.artifactType}/${params?.artifactIdentifier}`
    ),
    toARArtifactVersions: routeDefinitionWithMode(
      params => `/${params?.repositoryIdentifier}/${params?.artifactType}/${params?.artifactIdentifier}/versions`
    ),
    toARArtifactProperties: routeDefinitionWithMode(
      params => `/${params?.repositoryIdentifier}/${params?.artifactType}/${params?.artifactIdentifier}/properties`
    ),
    toARVersionDetails: routeDefinitionWithMode(params => {
      const queryParams = new URLSearchParams()
      if (params.tag) queryParams.append('tag', params.tag)
      if (params.digest) queryParams.append('digest', params.digest)
      const route = `/${params?.repositoryIdentifier}/${params?.artifactType}/${params?.artifactIdentifier}/versions/${params?.versionIdentifier}`
      if (queryParams.toString().length === 0) return route
      return `${route}?${queryParams.toString()}`
    }),
    // anything random, as this route will not be used in gitness
    toARVersionDetailsTab: routeDefinitionWithMode(params => {
      let route = `/${params.repositoryIdentifier}/${params?.artifactType}/${params?.artifactIdentifier}/versions/${params.versionIdentifier}`
      if (params.orgIdentifier) route += `/orgs/${params.orgIdentifier}`
      if (params.projectIdentifier) route += `/projects/${params.projectIdentifier}`
      if (params.sourceId && params.artifactId) {
        route += `/artifact-sources/${params.sourceId}/${params?.artifactType}/${params.artifactId}`
      }
      if (params.pipelineIdentifier && params.executionIdentifier) {
        route += `/pipelines/${params.pipelineIdentifier}/executions/${params.executionIdentifier}`
      }
      route += `/${params.versionTab}`
      return route
    }),
    toARRepositoryWebhookDetails: routeDefinitionWithMode(
      params => `/${params?.repositoryIdentifier}/webhooks/${params?.webhookIdentifier}`
    ),
    toARRepositoryWebhookDetailsTab: params =>
      `/${params?.repositoryIdentifier}/webhooks/${params?.webhookIdentifier}/${params?.tab}`
  }
}
