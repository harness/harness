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

import type {
  ArtifactVersionSummaryResponseResponse,
  DockerArtifactDetailIntegrationResponseResponse,
  DockerArtifactDetailResponseResponse,
  DockerManifestsResponseResponse,
  ListArtifactVersion,
  ListArtifactVersionResponseResponse
} from '@harnessio/react-har-service-client'

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

export const mockDockerVersionSummary: ArtifactVersionSummaryResponseResponse = {
  data: {
    imageName: 'maven-app',
    packageType: 'DOCKER',
    sscaArtifactId: '67a5dccf6d75916b0c3ea1b6',
    sscaArtifactSourceId: '67a5dccf6d75916b0c3ea1b5',
    stoExecutionId: 'Tbi7s6nETjmOMKU3Qrnm7A',
    stoPipelineId: 'HARNESS_ARTIFACT_SCAN_PIPELINE',
    version: '1.0.0'
  },
  status: 'SUCCESS'
}

export const mockDockerVersionList: ListArtifactVersionResponseResponse = {
  data: {
    artifactVersions: [
      {
        deploymentMetadata: {
          nonProdEnvCount: 0,
          prodEnvCount: 0
        },
        digestCount: 1,
        downloadsCount: 11,
        islatestVersion: false,
        lastModified: '1738923119434',
        name: '1.0.0',
        packageType: 'DOCKER',
        pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.0',
        registryIdentifier: '',
        registryPath: '',
        size: '246.43MB'
      },
      {
        deploymentMetadata: {
          nonProdEnvCount: 0,
          prodEnvCount: 0
        },
        digestCount: 1,
        downloadsCount: 11,
        islatestVersion: false,
        lastModified: '1738923402541',
        name: '1.0.1',
        packageType: 'DOCKER',
        pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.1',
        registryIdentifier: '',
        registryPath: '',
        size: '246.89MB'
      },
      {
        deploymentMetadata: {
          nonProdEnvCount: 0,
          prodEnvCount: 0
        },
        digestCount: 1,
        downloadsCount: 11,
        islatestVersion: true,
        lastModified: '1738924148637',
        name: '1.0.2',
        packageType: 'DOCKER',
        pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.2',
        registryIdentifier: '',
        registryPath: '',
        size: '246.89MB'
      }
    ],
    itemCount: 3,
    pageCount: 1,
    pageIndex: 0,
    pageSize: 100
  },
  status: 'SUCCESS'
}

export const mockDockerManifestList: DockerManifestsResponseResponse = {
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
        digest: 'sha256:112cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
        downloadsCount: 11,
        osArch: 'linux/amd64',
        size: '246.43MB',
        stoExecutionId: 'Tbi7s6nETjmOMKU3Qrnm7A',
        stoPipelineId: 'HARNESS_ARTIFACT_SCAN_PIPELINE'
      }
    ],
    version: '1.0.0'
  },
  status: 'SUCCESS'
}

export const mockDockerArtifactDetails: DockerArtifactDetailResponseResponse = {
  data: {
    createdAt: '1738923119434',
    downloadsCount: 11,
    imageName: 'maven-app',
    isLatestVersion: false,
    modifiedAt: '1738923119434',
    packageType: 'DOCKER',
    pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.0',
    registryPath: 'docker-repo/maven-app/sha256:144cdab68a435424250fe06e9a4f8a5f6b6b8a8a55d257bc6ee77476a6ec520d',
    size: '246.43MB',
    url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app/1.0.0',
    version: '1.0.0'
  },
  status: 'SUCCESS'
}

export const mockDockerArtifactIntegrationDetails: DockerArtifactDetailIntegrationResponseResponse = {
  data: {
    buildDetails: {
      orgIdentifier: 'default',
      pipelineDisplayName: 'deploy1',
      pipelineExecutionId: 'gs0w_JRPQMSPSd4PG1--hQ',
      pipelineIdentifier: 'deploy1',
      projectIdentifier: 'donotdeleteshivanand',
      stageExecutionId: 'Eo4H_SmyTZifDw3ygOha9Q',
      stepExecutionId: '4jYgJqIbR3q9jvh4oJEmjQ'
    },
    deploymentsDetails: {
      nonProdDeployment: 0,
      prodDeployment: 0,
      totalDeployment: 0
    },
    sbomDetails: {
      allowListViolations: 0,
      artifactId: '67a5dccf6d75916b0c3ea1b6',
      artifactSourceId: '67a5dccf6d75916b0c3ea1b5',
      avgScore: '7.305059523809524',
      componentsCount: 143,
      denyListViolations: 0,
      maxScore: '10',
      orchestrationId: 'yw0D70fiTqetxx0HIyvEUQ',
      orgId: 'default',
      projectId: 'donotdeleteshivanand'
    },
    slsaDetails: {
      provenanceId: '',
      status: ''
    },
    stoDetails: {
      critical: 17,
      executionId: 'Tbi7s6nETjmOMKU3Qrnm7A',
      high: 19,
      ignored: 0,
      info: 0,
      lastScanned: '1738923283',
      low: 0,
      medium: 13,
      pipelineId: 'HARNESS_ARTIFACT_SCAN_PIPELINE',
      total: 49
    }
  },
  status: 'SUCCESS'
}
