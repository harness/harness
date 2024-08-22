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

import type { ARRouteDefinitionsReturn } from '@ar/routes/RouteDefinitions'

export default function getARRouteDefinitions(routeParams: Record<string, string>): ARRouteDefinitionsReturn {
  return {
    // anything random, as this route will not be used in gitness
    toAR: () => '/ar',
    toARRepositories: () => '/',
    toARRepositoryDetails: params => {
      let url = `/${params?.repositoryIdentifier}?`
      if (params.tab) {
        url += `tab=${params.tab}`
      }
      return url
    },
    toARArtifacts: () => `/${routeParams?.repositoryIdentifier}?tab=packages`,
    toARArtifactDetails: params => `/${params?.repositoryIdentifier}/artifacts/${params?.artifactIdentifier}`,
    toARVersionDetails: params =>
      `/${params?.repositoryIdentifier}/artifacts/${params?.artifactIdentifier}/versions/${params?.versionIdentifier}`,
    // anything random, as this route will not be used in gitness
    toARVersionDetailsTab: params =>
      `/${params?.repositoryIdentifier}/artifacts/${params?.artifactIdentifier}/versions/${params?.versionIdentifier}`
  }
}
