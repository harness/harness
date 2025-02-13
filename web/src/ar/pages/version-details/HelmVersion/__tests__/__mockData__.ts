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
  ListArtifactVersion,
  ListArtifactVersionResponseResponse
} from '@harnessio/react-har-service-client'

export const mockHelmLatestVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      digestCount: 1,
      downloadsCount: 100,
      islatestVersion: true,
      lastModified: '1730978736333',
      name: '1.0.0',
      packageType: 'HELM',
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

export const mockHelmVersionSummary: ArtifactVersionSummaryResponseResponse = {
  data: {
    imageName: 'maven-app',
    isLatestVersion: true,
    packageType: 'HELM',
    version: '1.0.0'
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
        islatestVersion: false,
        lastModified: '1738923119434',
        name: '1.0.0',
        packageType: 'HELM',
        pullCommand: 'helm pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.0',
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
        packageType: 'HELM',
        pullCommand: 'helm pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.1',
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
        packageType: 'HELM',
        pullCommand: 'helm pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/maven-app:1.0.2',
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

export const mockHelmArtifactDetails: HelmArtifactDetailResponseResponse = {
  data: {
    artifact: 'production',
    createdAt: '1729862063842',
    downloadsCount: 0,
    isLatestVersion: true,
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
