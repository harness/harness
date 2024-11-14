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

export const MockGetDockerRegistryResponseWithMinimumData = {
  content: {
    data: {
      config: {
        type: 'VIRTUAL'
      },
      createdAt: '1729754358172',
      identifier: 'docker-repo',
      packageType: 'DOCKER',
      url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo',
      cleanupPolicy: []
    },
    status: 'SUCCESS'
  }
}

export const MockGetDockerRegistryResponseWithMinimumDataForOSS = {
  content: {
    data: {
      config: {
        type: 'VIRTUAL',
        upstreamProxies: ['test1']
      },
      createdAt: '1729754358172',
      identifier: 'docker-repo',
      modifiedAt: '1730978957105',
      packageType: 'DOCKER',
      url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo',
      cleanupPolicy: []
    },
    status: 'SUCCESS'
  }
}

export const MockGetUpstreamProxyRegistryListResponse = {
  content: {
    data: {
      itemCount: 9,
      pageCount: 1,
      pageIndex: 0,
      pageSize: 100,
      registries: [
        {
          description: '',
          identifier: 'docker-proxy',
          lastModified: '1729701628765',
          packageType: 'DOCKER',
          registrySize: '0.00B',
          type: 'UPSTREAM',
          url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-proxy'
        },
        {
          description: '',
          identifier: 'docker-proxy-2',
          lastModified: '1729701782524',
          packageType: 'DOCKER',
          registrySize: '0.00B',
          type: 'UPSTREAM',
          url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-proxy-2'
        },
        {
          description: '',
          identifier: 'proxy-1',
          lastModified: '1729703542869',
          packageType: 'DOCKER',
          registrySize: '0.00B',
          type: 'UPSTREAM',
          url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/proxy-1'
        },
        {
          description: '',
          identifier: 'proxy-3',
          lastModified: '1729703916934',
          packageType: 'DOCKER',
          registrySize: '0.00B',
          type: 'UPSTREAM',
          url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/proxy-3'
        },
        {
          description: '',
          identifier: 'proxy-4',
          lastModified: '1729703967869',
          packageType: 'DOCKER',
          registrySize: '0.00B',
          type: 'UPSTREAM',
          url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/proxy-4'
        },
        {
          description: '',
          identifier: 'proxy-5',
          lastModified: '1729704178318',
          packageType: 'DOCKER',
          registrySize: '0.00B',
          type: 'UPSTREAM',
          url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/proxy-5'
        },
        {
          description: '',
          identifier: 'proxy-6',
          lastModified: '1729704209947',
          packageType: 'DOCKER',
          registrySize: '0.00B',
          type: 'UPSTREAM',
          url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/proxy-6'
        },
        {
          description: '',
          identifier: 'proxy-10',
          lastModified: '1729759142005',
          packageType: 'DOCKER',
          registrySize: '0.00B',
          type: 'UPSTREAM',
          url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/proxy-10'
        },
        {
          description: '',
          identifier: 'docker-hub-proxy',
          lastModified: '1730978772571',
          packageType: 'DOCKER',
          registrySize: '0.00B',
          type: 'UPSTREAM',
          url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-hub-proxy'
        }
      ]
    },
    status: 'SUCCESS'
  }
}
