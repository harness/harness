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
  ArtifactDetailResponseResponse,
  ArtifactVersionSummaryResponseResponse,
  ListArtifactVersion,
  ListArtifactVersionResponseResponse
} from '@harnessio/react-har-service-client'

export const mockGenericLatestVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      fileCount: 10,
      lastModified: '1730978736333',
      name: '1.0.0',
      packageType: 'GENERIC',
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

export const mockGenericVersionSummary: ArtifactVersionSummaryResponseResponse = {
  data: {
    imageName: 'maven-app',
    packageType: 'GENERIC',
    version: '1.0.0'
  },
  status: 'SUCCESS'
}

export const mockGenericVersionList: ListArtifactVersionResponseResponse = {
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
        packageType: 'GENERIC',
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
        packageType: 'GENERIC',
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
        packageType: 'GENERIC',
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

export const mockGenericArtifactDetails: ArtifactDetailResponseResponse = {
  data: {
    createdAt: '1738085520013',
    description: 'test descriptions',
    modifiedAt: '1738085520013',
    name: 'artifact',
    packageType: 'GENERIC',
    version: 'v1'
  },
  status: 'SUCCESS'
}
