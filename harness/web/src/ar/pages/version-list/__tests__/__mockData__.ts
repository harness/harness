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

import type { DockerManifests, ListArtifactVersion } from '@harnessio/react-har-service-client'

export const mockHelmLatestVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      deploymentMetadata: {
        nonProdEnvCount: 0,
        prodEnvCount: 0
      },
      digestCount: 1,
      lastModified: '1729861854693',
      name: '1.0.15',
      packageType: 'HELM',
      pullCommand: 'helm pull oci://pkg.qa.harness.io/iwnq/helm-repo-1/harness-delegate-ng:1.0.15',
      registryIdentifier: '',
      registryPath: '',
      size: '8.63KB',
      downloadsCount: 0
    }
  ],
  itemCount: 55,
  pageCount: 2,
  pageIndex: 0,
  pageSize: 50
}

export const mockHelmNoPullCmdVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      deploymentMetadata: {
        nonProdEnvCount: 0,
        prodEnvCount: 0
      },
      digestCount: 1,
      lastModified: '1729861854693',
      name: '1.0.15',
      packageType: 'HELM',
      pullCommand: '',
      registryIdentifier: '',
      registryPath: '',
      size: '8.63KB',
      downloadsCount: 0
    }
  ],
  itemCount: 55,
  pageCount: 2,
  pageIndex: 0,
  pageSize: 50
}

export const mockHelmOldVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      deploymentMetadata: {
        nonProdEnvCount: 0,
        prodEnvCount: 0
      },
      digestCount: 1,
      lastModified: '1729861854693',
      name: '1.0.15',
      packageType: 'HELM',
      pullCommand: 'helm pull oci://pkg.qa.harness.io/iwft7r-f_zp7q/helm-repo-1/harness-delegate-ng:1.0.15',
      registryIdentifier: '',
      registryPath: '',
      size: '8.63KB',
      downloadsCount: 0
    }
  ],
  itemCount: 55,
  pageCount: 2,
  pageIndex: 0,
  pageSize: 50
}

export const mockDockerNoPullCmdVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      deploymentMetadata: {
        nonProdEnvCount: 0,
        prodEnvCount: 0
      },
      digestCount: 1,
      lastModified: '1730978736333',
      name: '1.0.0',
      packageType: 'DOCKER',
      pullCommand: '',
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

export const mockDockerLatestVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      deploymentMetadata: {
        nonProdEnvCount: 0,
        prodEnvCount: 0
      },
      digestCount: 1,
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

export const mockDockerOldVersionListTableData: ListArtifactVersion = {
  artifactVersions: [
    {
      deploymentMetadata: {
        nonProdEnvCount: 0,
        prodEnvCount: 0
      },
      digestCount: 1,
      lastModified: '1730978736333',
      name: '1.0.0',
      packageType: 'DOCKER',
      pullCommand: 'docker pull pkg.qa.harness.io/iwnhltzp7q/docker-repo/podinfo-artifact:1.0.0',
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

export const mockDockerManifestTableData: DockerManifests = {
  imageName: 'podinfo-artifact',
  manifests: [
    {
      createdAt: '1730978736256',
      digest: 'sha256:b',
      osArch: 'linux/amd64',
      size: '69.56MB',
      stoExecutionId: '',
      stoPipelineId: ''
    }
  ],
  version: '1.0.0'
}

export const mockHelmUseGetAllArtifactVersionsQueryResponse = {
  isFetching: false,
  data: {
    content: {
      data: mockHelmOldVersionListTableData,
      status: 'SUCCESS'
    }
  },
  refetch: jest.fn(),
  error: null
}

export const mockDockerUseGetAllArtifactVersionsQueryResponse = {
  isFetching: false,
  data: {
    content: {
      data: mockDockerLatestVersionListTableData,
      status: 'SUCCESS'
    }
  },
  refetch: jest.fn(),
  error: null
}

export const mockEmptyUseGetAllArtifactVersionsQueryResponse = {
  isFetching: false,
  data: {
    content: {
      data: {
        artifactVersions: [],
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

export const mockNullDataUseGetAllArtifactVersionsQueryResponse = {
  isFetching: false,
  data: {
    content: {
      data: null,
      status: 'SUCCESS'
    }
  },
  refetch: jest.fn(),
  error: null
}

export const mockErrorUseGetAllArtifactVersionsQueryResponse = {
  isFetching: false,
  data: null,
  refetch: jest.fn(),
  error: {
    message: 'error message'
  }
}

export const mockUseGetDockerArtifactManifestsQueryResponse = {
  isFetching: false,
  data: {
    content: {
      data: mockDockerManifestTableData,
      status: 'SUCCESS'
    }
  },
  refetch: jest.fn(),
  error: null
}
