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

import type { GetAllArtifactsByRegistryOkResponse } from '@harnessio/react-har-service-client'

export const MockGetDockerRegistryResponseWithAllData = {
  content: {
    data: {
      config: {
        type: 'VIRTUAL',
        upstreamProxies: ['docker-hub-proxy']
      },
      createdAt: '1729754358172',
      description: 'test description',
      identifier: 'docker-repo',
      labels: ['label1', 'label2', 'label3', 'label4'],
      modifiedAt: '1730978957105',
      packageType: 'DOCKER',
      url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo',
      allowedPattern: ['test1', 'test2'],
      blockedPattern: ['test3', 'test4']
    },
    status: 'SUCCESS'
  }
}

export const MockGetArtifactsByRegistryResponse: GetAllArtifactsByRegistryOkResponse = {
  content: {
    data: {
      artifacts: [
        {
          downloadsCount: 0,
          lastModified: '1730978736333',
          latestVersion: '1.0.0',
          name: 'podinfo-artifact',
          packageType: 'DOCKER',
          registryIdentifier: 'docker-repo',
          registryPath: ''
        }
      ],
      itemCount: 1,
      pageCount: 1,
      pageIndex: 0,
      pageSize: 50
    },
    status: 'SUCCESS'
  }
}

export const MockGetSetupClientOnRegistryConfigPageResponse = {
  content: {
    data: {
      mainHeader: 'Docker Client Setup',
      secHeader: 'Follow these instructions to install/use Docker artifacts or compatible packages.',
      sections: [
        {
          header: 'Login to Docker',
          steps: [
            {
              commands: [
                'docker login pkg.qa.harness.io',
                'Username: shivanand.sonnad@harness.io',
                'Password: *see step 2*'
              ],
              header: 'Run this Docker command in your terminal to authenticate the client.',
              type: 'Static'
            },
            {
              header: 'For the Password field above, generate an identity token',
              type: 'GenerateToken'
            }
          ]
        },
        {
          header: 'Retag and Push the image',
          steps: [
            {
              commands: [
                'docker tag \u003cIMAGE_NAME\u003e:\u003cTAG\u003e pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/\u003cIMAGE_NAME\u003e:\u003cTAG\u003e'
              ],
              header: 'Run this Docker command in your terminal to tag the image.',
              type: 'Static'
            },
            {
              commands: [
                'docker push pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/\u003cIMAGE_NAME\u003e:\u003cTAG\u003e'
              ],
              header: 'Run this Docker command in your terminal to push the image.',
              type: 'Static'
            }
          ]
        },
        {
          header: 'Pull an image',
          steps: [
            {
              commands: [
                'docker pull pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/docker-repo/\u003cIMAGE_NAME\u003e:\u003cTAG\u003e'
              ],
              header: 'Run this Docker command in your terminal to pull image.',
              type: 'Static'
            }
          ]
        }
      ]
    },
    status: 'SUCCESS'
  }
}
