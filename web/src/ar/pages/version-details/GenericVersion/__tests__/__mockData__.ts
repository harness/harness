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
  FileDetail,
  FileDetailResponseResponse,
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

export const mockGenericArtifactFiles: FileDetailResponseResponse = {
  files: [
    {
      checksums: [
        'SHA-512: ebfd0613c1c298d97fd7743ecd731b96a9c0a7f7cdbfdd9f19ec4682f9b4ceb400420a6191c9671bfb3e1cc5a9fef873ea1ad78f1b30794989a0fdb591f847cd',
        'SHA-256: cc5ac7831841b65901c5578a47d6b125259f9a4364d1d73b0b5d8891ad3b7d34',
        'SHA-1: b0e3200eb5eaca07d773916e306cd1ed9866d3a4',
        'MD5: cc576cbab9119ad7589cae7b57146af6'
      ],
      createdAt: '1738258177381',
      downloadCommand:
        "curl --location 'https://pkg.qa.harness.io/generic/iWnhltqOT7GFt7R-F_zP7Q/generic-registry/artifact:v1:image.png' --header 'x-api-key: \u003cAPI_KEY\u003e' -J -O",
      name: 'image.png',
      size: '170.18KB'
    },
    {
      createdAt: '1738085520008',
      name: 'hello.yaml',
      size: '2.79MB'
    } as FileDetail
  ],
  itemCount: 2,
  pageCount: 4,
  pageIndex: 0,
  pageSize: 50,
  status: 'SUCCESS'
}
