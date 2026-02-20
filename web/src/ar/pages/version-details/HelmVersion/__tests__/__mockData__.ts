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
  HelmArtifactDetailResponseResponse,
  HelmArtifactManifestResponseResponse,
  ListArtifactVersion,
  ListArtifactVersionResponseResponse
} from '@harnessio/react-har-service-client'

export const mockHelmLatestVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      digestCount: 1,
      downloadsCount: 100,
      lastModified: '1730978736333',
      name: '1.0.0',
      packageType: 'HELM',
      pullCommand: 'docker pull pkg.qa.harness.io/iwnhltqp7q/docker-repo/podinfo-artifact:1.0.0',
      registryIdentifier: '',
      registryPath: '',
      size: '69.56MB',
      uuid: 'uuid',
      registryUUID: 'uuid'
    }
  ],
  itemCount: 55,
  pageCount: 2,
  pageIndex: 0,
  pageSize: 50
}

export const mockHelmVersionSummary: ArtifactVersionSummaryResponseResponse = {
  data: {
    imageName: 'maven-app',
    packageType: 'HELM',
    version: '1.0.0',
    uuid: 'uuid',
    registryUUID: 'uuid',
    purl: 'test'
  },
  status: 'SUCCESS'
}

export const mockHelmVersionList: ListArtifactVersionResponseResponse = {
  data: {
    artifactVersions: [
      {
        deploymentMetadata: {
          nonProdEnvCount: 0,
          prodEnvCount: 0
        },
        digestCount: 1,
        downloadsCount: 11,
        lastModified: '1738923119434',
        name: '1.0.0',
        packageType: 'HELM',
        pullCommand: 'helm pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.0',
        registryIdentifier: '',
        registryPath: '',
        size: '246.43MB',
        uuid: 'uuid',
        registryUUID: 'uuid'
      },
      {
        deploymentMetadata: {
          nonProdEnvCount: 0,
          prodEnvCount: 0
        },
        digestCount: 1,
        downloadsCount: 11,
        lastModified: '1738923402541',
        name: '1.0.1',
        packageType: 'HELM',
        pullCommand: 'helm pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.1',
        registryIdentifier: '',
        registryPath: '',
        size: '246.89MB',
        uuid: 'uuid',
        registryUUID: 'uuid'
      },
      {
        deploymentMetadata: {
          nonProdEnvCount: 0,
          prodEnvCount: 0
        },
        digestCount: 1,
        downloadsCount: 11,
        lastModified: '1738924148637',
        name: '1.0.2',
        packageType: 'HELM',
        pullCommand: 'helm pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.2',
        registryIdentifier: '',
        registryPath: '',
        size: '246.89MB',
        uuid: 'uuid',
        registryUUID: 'uuid'
      }
    ],
    itemCount: 3,
    pageCount: 1,
    pageIndex: 0,
    pageSize: 100
  },
  status: 'SUCCESS'
}

export const mockHelmArtifactDetails: HelmArtifactDetailResponseResponse = {
  data: {
    artifact: 'production',
    createdAt: '1729862063842',
    downloadsCount: 0,
    modifiedAt: '1729862063842',
    packageType: 'HELM',
    pullCommand: 'helm pull oci://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/helm-repo-1/production:1.0.15',
    registryPath: 'helm-repo-1/production/sha256:a0fc1e52764215bd82ed15b4a1544e3716c16a99f0e43116137b330e6c45b3de',
    size: '8.62KB',
    url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/helm-repo-1/production/1.0.15',
    version: '1.0.15'
  },
  status: 'SUCCESS'
}

export const mockHelmArtifactManifest: HelmArtifactManifestResponseResponse = {
  data: {
    manifest:
      '{"schemaVersion":2,"config":{"mediaType":"application/vnd.cncf.helm.config.v1+json","digest":"sha256:3d3ca122368140982b0a494f4f357fff2fa9894f9adcc8809fa8e74e2a327d94","size":161},"layers":[{"mediaType":"application/vnd.cncf.helm.chart.content.v1.tar+gzip","digest":"sha256:a8b12f90950f22927e8c2e4f3e9b32655ae5287e95ae801662cef7cf66bd9be3","size":8062}],"annotations":{"org.opencontainers.image.created":"2024-10-25T18:43:34+05:30","org.opencontainers.image.description":"A Helm chart for deploying harness-delegate","org.opencontainers.image.title":"production","org.opencontainers.image.version":"1.0.15"}}'
  },
  status: 'SUCCESS'
}
