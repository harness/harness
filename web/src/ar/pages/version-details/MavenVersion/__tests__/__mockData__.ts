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
  FileDetailResponseResponse,
  ListArtifactVersion,
  ListArtifactVersionResponseResponse
} from '@harnessio/react-har-service-client'

export const mockMavenLatestVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      fileCount: 10,
      lastModified: '1730978736333',
      name: '1.0.0',
      packageType: 'MAVEN',
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

export const mockMavenVersionSummary: ArtifactVersionSummaryResponseResponse = {
  data: {
    imageName: 'maven-app',
    packageType: 'MAVEN',
    version: '1.0.0'
  },
  status: 'SUCCESS'
}

export const mockMavenVersionList: ListArtifactVersionResponseResponse = {
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
        packageType: 'MAVEN',
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
        lastModified: '1738923402541',
        name: '1.0.1',
        packageType: 'MAVEN',
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
        lastModified: '1738924148637',
        name: '1.0.2',
        packageType: 'MAVEN',
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

export const mockMavenArtifactDetails: ArtifactDetailResponseResponse = {
  data: {
    createdAt: '1738152289788',
    modifiedAt: '1738152289788',
    name: 'com.example:my-maven-project',
    packageType: 'MAVEN',
    size: '6.81KB',
    version: '1.0-SNAPSHOT',
    downloadCount: 0
  },
  status: 'SUCCESS'
}

export const mockMavenArtifactFiles: FileDetailResponseResponse = {
  files: [
    {
      checksums: [
        'SHA-512: 43b206d0651f2748d685c9ed63942091fe529a0c838effeb15b3d21139a5c25f1086bad9e2df57a3032bf1cf26837b7cfe504f7b033d872a6d2d41d311aba882',
        'SHA-256: 04cf2f4ad947a81667690d64059ee29d3edc0d74649f82df489565c6cc1edcc0',
        'SHA-1: 475741bfe798caedd27a1c2580ea211aeba32521',
        'MD5: 5eb4f955706b83204f7d4dbbdecdb0e6'
      ],
      createdAt: '1738316037624',
      downloadCommand:
        "curl --location 'https://pkg.qa.harness.io/maven/iWnhltqOT7GFt7R-F_zP7Q/maven-up-1/junit/junit/3.8.1/junit-3.8.1.jar.sha1' --header 'x-api-key: \u003cIDENTITY_TOKEN\u003e' -O",
      name: 'junit-3.8.1.jar.sha1',
      size: '40.00B'
    },
    {
      checksums: [
        'SHA-512: cfc89ca8303af5c04c75a73db181b61a34371b9e0dcc59e4d746190ac2e7636f0b257303ebef4db9a2cd980d192ab8879c91d84682d472b03fd3b9a732f184b6',
        'SHA-256: ab9b2ba5775492d85d45240f6f12e5880eb0ce26385fd80a1083e3b4ded402c2',
        'SHA-1: 11c996e14e70c07f6758f325838ea07e3bdf0742',
        'MD5: 4f215459aacbaaac97d02c29c41b2a57'
      ],
      createdAt: '1738316032125',
      downloadCommand:
        "curl --location 'https://pkg.qa.harness.io/maven/iWnhltqOT7GFt7R-F_zP7Q/maven-up-1/junit/junit/3.8.1/junit-3.8.1.pom.sha1' --header 'x-api-key: \u003cIDENTITY_TOKEN\u003e' -O",
      name: 'junit-3.8.1.pom.sha1',
      size: '58.00B'
    },
    {
      checksums: [
        'SHA-512: 8e6f9fa5eb3ba93a8b1b5a39e01a81c142b33078264dbd0a2030d60dd26735407249a12e66f5cdcab8056e93a5687124fe66e741c233b4c7a06cc8e49f82e98b',
        'SHA-256: b58e459509e190bed737f3592bc1950485322846cf10e78ded1d065153012d70',
        'SHA-1: 99129f16442844f6a4a11ae22fbbee40b14d774f',
        'MD5: 1f40fb782a4f2cf78f161d32670f7a3a'
      ],
      createdAt: '1738152520460',
      downloadCommand:
        "curl --location 'https://pkg.qa.harness.io/maven/iWnhltqOT7GFt7R-F_zP7Q/maven-up-1/junit/junit/3.8.1/junit-3.8.1.jar' --header 'x-api-key: \u003cIDENTITY_TOKEN\u003e' -O",
      name: 'junit-3.8.1.jar',
      size: '118.23KB'
    },
    {
      checksums: [
        'SHA-512: d43bddd7228b108eab508871d64725a730f6f159b0cee0e25a62df61f5362dc4c3e7c3413b5562b22e20934b40b5d994c1b1f66fec0e1a340613913e05203396',
        'SHA-256: e68f33343d832398f3c8aa78afcd808d56b7c1020de4d3ad8ce47909095ee904',
        'SHA-1: 16d74791c801c89b0071b1680ea0bc85c93417bb',
        'MD5: 50b40cb7342f52b702e6337d5debf1ae'
      ],
      createdAt: '1738152517836',
      downloadCommand:
        "curl --location 'https://pkg.qa.harness.io/maven/iWnhltqOT7GFt7R-F_zP7Q/maven-up-1/junit/junit/3.8.1/junit-3.8.1.pom' --header 'x-api-key: \u003cIDENTITY_TOKEN\u003e' -O",
      name: 'junit-3.8.1.pom',
      size: '998.00B'
    }
  ],
  itemCount: 4,
  pageCount: 10,
  pageIndex: 0,
  pageSize: 50,
  status: 'SUCCESS'
}
