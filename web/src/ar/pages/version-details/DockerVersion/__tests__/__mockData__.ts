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

import type { DockerManifestsResponseResponse, ListArtifactVersion } from '@harnessio/react-har-service-client'

export const mockDockerLatestVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      deploymentMetadata: {
        nonProdEnvCount: 0,
        prodEnvCount: 0
      },
      digestCount: 1,
      islatestVersion: true,
      lastModified: '1730978736333',
      name: '1.0.0',
      packageType: 'DOCKER',
      pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqp7q/docker-repo/podinfo-artifact:1.0.0',
      registryIdentifier: '',
      registryPath: '',
      size: '69.56MB'
    }
  ],
  itemCount: 55,
  pageCount: 2,
  pageIndex: 0,
  pageSize: 50
}

export const mockDockerManifestListTableData: DockerManifestsResponseResponse = {
  data: {
    imageName: 'maven-app',
    manifests: [
      {
        createdAt: '1738923119376',
        digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
        downloadsCount: 11,
        osArch: 'linux/arm64',
        size: '246.43MB',
        stoExecutionId: 'Tbi7s6nETjmOMKU3Qrnm7A',
        stoPipelineId: 'HARNESS_ARTIFACT_SCAN_PIPELINE'
      },
      {
        createdAt: '1738923119376',
        digest: 'sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
        downloadsCount: 11,
        osArch: 'linux/arm64',
        size: '246.43MB'
      }
    ],
    version: '1.0.0'
  },
  status: 'SUCCESS'
}
