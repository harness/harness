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

export const MockGetHelmRegistryResponseWithAllData = {
  content: {
    data: {
      config: {
        type: 'VIRTUAL',
        upstreamProxies: ['helm-hub-proxy']
      },
      createdAt: '1729754358172',
      description: 'test description',
      identifier: 'helm-repo',
      labels: ['label1', 'label2', 'label3', 'label4'],
      modifiedAt: '1730978957105',
      packageType: 'HELM',
      url: 'https://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/helm-repo',
      allowedPattern: ['test1', 'test2'],
      blockedPattern: ['test3', 'test4']
    },
    status: 'SUCCESS'
  }
}

export const MockGetHelmArtifactsByRegistryResponse: GetAllArtifactsByRegistryOkResponse = {
  content: {
    data: {
      artifacts: [
        {
          downloadsCount: 0,
          lastModified: '1730978736333',
          latestVersion: '1.0.0',
          name: 'podinfo-artifact',
          packageType: 'HELM',
          registryIdentifier: 'helm-repo',
          registryPath: '',
          isPublic: false
        }
      ],
      itemCount: 1,
      pageCount: 2,
      pageIndex: 0,
      pageSize: 50
    },
    status: 'SUCCESS'
  }
}

export const MockGetHelmSetupClientOnRegistryConfigPageResponse = {
  content: {
    data: {
      mainHeader: 'Helm Client Setup',
      secHeader: 'Follow these instructions to install/use Helm artifacts or compatible packages.',
      sections: [
        {
          header: 'Login to Helm',
          steps: [
            {
              commands: ['helm registry login pkg.qa.harness.io'],
              header: 'Run this Helm command in your terminal to authenticate the client.',
              type: 'Static'
            },
            {
              header: 'For the Password field above, generate an identity token',
              type: 'GenerateToken'
            }
          ]
        },
        {
          header: 'Push a version',
          steps: [
            {
              commands: [
                'helm push \u003cCHART_TGZ_FILE\u003e oci://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/helm-create-test'
              ],
              header:
                'Run this Helm push command in your terminal to push a chart in OCI form. Note: Make sure you add oci:// prefix to the repository URL.',
              type: 'Static'
            }
          ]
        },
        {
          header: 'Pull a version',
          steps: [
            {
              commands: [
                'helm pull oci://pkg.qa.harness.io/iwnhltqot7gft7r-f_zp7q/helm-create-test/\u003cIMAGE_NAME\u003e --version \u003cTAG\u003e'
              ],
              header: 'Run this Helm command in your terminal to pull a specific chart version.',
              type: 'Static'
            }
          ]
        }
      ]
    },
    status: 'SUCCESS'
  }
}

export const MockGetHelmUpstreamRegistryResponseWithCustomSourceAllData = {
  content: {
    data: {
      allowedPattern: ['test1', 'test2'],
      cleanupPolicy: [],
      config: {
        auth: null,
        authType: 'Anonymous',
        source: 'Custom',
        type: 'UPSTREAM',
        url: 'https://aws.ecr.com'
      },
      createdAt: '1738516362995',
      identifier: 'helm-up-repo',
      description: 'test description',
      modifiedAt: '1738516362995',
      packageType: 'HELM',
      labels: ['label1', 'label2', 'label3', 'label4'],
      url: '',
      isPublic: false
    },
    status: 'SUCCESS'
  }
}
