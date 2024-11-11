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

import type { GetAllRegistriesOkResponse } from '@harnessio/react-har-service-client'

export const mockEmptyUseGetAllHarnessArtifactsQueryResponse = {
  isFetching: false,
  data: {
    content: {
      data: {
        artifacts: [],
        itemCount: 0,
        pageCount: 0,
        pageIndex: 0,
        pageSize: 50
      },
      status: 'SUCCESS'
    }
  },
  refetch: jest.fn(),
  error: null
}

export const mockErrorUseGetAllHarnessArtifactsQueryResponse = {
  isFetching: false,
  data: null,
  refetch: jest.fn(),
  error: {
    message: 'error message'
  }
}

export const mockEmptyGetAllRegistriesResponse: GetAllRegistriesOkResponse = {
  content: {
    data: {
      itemCount: 0,
      pageCount: 0,
      pageIndex: 0,
      pageSize: 0,
      registries: []
    },
    status: 'SUCCESS'
  }
}

export const mockUseGetAllHarnessArtifactsQueryResponse = {
  isFetching: false,
  data: {
    content: {
      data: {
        artifacts: [
          {
            deploymentMetadata: {
              nonProdEnvCount: 0,
              prodEnvCount: 0
            },
            downloadsCount: 0,
            labels: null,
            lastModified: '1730978736333',
            latestVersion: '',
            name: 'podinfo-artifact',
            packageType: 'DOCKER',
            pullCommand: 'docker pull pkg.qa.harness.io/iwr-f_zp7q/repo1/podinfo-artifact:1.0.0',
            registryIdentifier: 'repo1',
            registryPath: '',
            version: '1.0.0'
          },
          {
            deploymentMetadata: {
              nonProdEnvCount: 0,
              prodEnvCount: 0
            },
            downloadsCount: 0,
            labels: null,
            lastModified: '1729862063842',
            latestVersion: '',
            name: 'production',
            packageType: 'HELM',
            pullCommand: 'helm install pkg.qa.harness.io/iwr-f_zp7q/repo2/production:1.0.15',
            registryIdentifier: 'repo2',
            registryPath: '',
            scannedDigest: [],
            version: '1.0.15'
          },
          {
            deploymentMetadata: {
              nonProdEnvCount: 0,
              prodEnvCount: 0
            },
            downloadsCount: 2,
            labels: null,
            lastModified: '1729861854693',
            latestVersion: '',
            name: 'harness-delegate-ng',
            packageType: 'HELM',
            pullCommand: 'helm install pkg.qa.harness.io/iwr-f_zp7q/upstream_1/harness-delegate-ng:1.0.15',
            registryIdentifier: 'upstream_1',
            registryPath: '',
            scannedDigest: [],
            version: '1.0.15'
          }
        ],
        itemCount: 3,
        pageCount: 2,
        pageIndex: 0,
        pageSize: 50
      },
      status: 'SUCCESS'
    }
  },
  refetch: jest.fn(),
  error: null
}

export const mockGetAllRegistriesResponse: GetAllRegistriesOkResponse = {
  content: {
    data: {
      registries: [
        {
          identifier: 'repo1',
          packageType: 'DOCKER',
          type: 'VIRTUAL',
          url: 'space/repo1'
        },
        {
          identifier: 'repo2',
          packageType: 'DOCKER',
          description: 'Test Discription',
          labels: ['label1', 'label2', 'label2'],
          type: 'VIRTUAL',
          url: 'space/repo1',
          downloadsCount: 100,
          registrySize: '100 MB',
          artifactsCount: 100
        },
        {
          identifier: 'upstream_1',
          packageType: 'DOCKER',
          type: 'UPSTREAM',
          url: 'space/upstream_1'
        }
      ],
      itemCount: 3,
      pageCount: 10,
      pageIndex: 0,
      pageSize: 10
    },
    status: 'SUCCESS'
  }
}

export const mockEmptyUseGetAllArtifactsByRegistryQuery = {
  isFetching: false,
  data: {
    content: {
      data: {
        artifacts: [],
        itemCount: 0,
        pageCount: 0,
        pageIndex: 0,
        pageSize: 50
      },
      status: 'SUCCESS'
    }
  },
  refetch: jest.fn(),
  error: null
}

export const mockErrorUseGetAllArtifactsByRegistryQuery = {
  isFetching: false,
  data: null,
  refetch: jest.fn(),
  error: {
    message: 'error message'
  }
}

export const mockUseGetAllArtifactsByRegistryQuery = {
  isFetching: false,
  data: {
    content: {
      data: {
        artifacts: [
          {
            deploymentMetadata: {
              nonProdEnvCount: 0,
              prodEnvCount: 0
            },
            downloadsCount: 0,
            labels: null,
            lastModified: '1730978736333',
            latestVersion: '',
            name: 'podinfo-artifact',
            packageType: 'DOCKER',
            pullCommand: 'docker pull pkg.qa.harness.io/iwr-f_zp7q/repo1/podinfo-artifact:1.0.0',
            registryIdentifier: 'repo1',
            registryPath: '',
            version: '1.0.0'
          }
        ],
        itemCount: 1,
        pageCount: 2,
        pageIndex: 0,
        pageSize: 50
      },
      status: 'SUCCESS'
    }
  },
  refetch: jest.fn(),
  error: null
}

export const mockUseGetArtifactSummaryQueryResponse = {
  isFetching: false,
  data: {
    content: {
      data: {
        createdAt: '1729861185934',
        downloadsCount: 0,
        imageName: 'harness-delegate-ng',
        labels: null,
        modifiedAt: '1729861854693',
        packageType: 'HELM'
      },
      status: 'SUCCESS'
    }
  },
  refetch: jest.fn(),
  error: null
}
