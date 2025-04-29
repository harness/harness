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

export const MockGetGenericRegistryResponseWithAllData = {
  content: {
    data: {
      config: {
        type: 'VIRTUAL',
        upstreamProxies: []
      },
      createdAt: '1738040653409',
      description: 'custom description',
      identifier: 'generic-repo',
      labels: ['label1', 'label2', 'label3', 'label4'],
      modifiedAt: '1738040653409',
      packageType: 'GENERIC',
      url: 'http://host.docker.internal:3000/artifact-registry/generic-repo',
      allowedPattern: ['test1', 'test2'],
      blockedPattern: ['test3', 'test4']
    },
    status: 'SUCCESS'
  }
}

export const MockGetGenericArtifactsByRegistryResponse: GetAllArtifactsByRegistryOkResponse = {
  content: {
    data: {
      artifacts: [
        {
          downloadsCount: 0,
          lastModified: '1738048875014',
          latestVersion: 'v1',
          name: 'artifact',
          packageType: 'GENERIC',
          registryIdentifier: 'generic-repo',
          registryPath: ''
        }
      ],
      itemCount: 0,
      pageCount: 2,
      pageIndex: 0,
      pageSize: 50
    },
    status: 'SUCCESS'
  }
}

export const MockGetGenericSetupClientOnRegistryConfigPageResponse = {
  content: {
    data: {
      mainHeader: 'Generic Client Setup',
      secHeader: 'Follow these instructions to install/use Generic artifacts or compatible packages.',
      sections: [
        {
          header: 'Generate identity token',
          steps: [
            {
              header: 'An identity token will serve as the password for uploading and downloading artifact.',
              type: 'GenerateToken'
            }
          ],
          type: 'INLINE'
        },
        {
          header: 'Upload Artifact',
          steps: [
            {
              commands: [
                {
                  label: '',
                  value:
                    "curl --location --request PUT 'http://host.docker.internal:3000/generic/artifact-registry/generic-repo/\u003cARTIFACT_NAME\u003e/\u003cVERSION\u003e' \\\n--form 'filename=\"\u003cFILENAME\u003e\"' \\\n--form 'file=@\"\u003cFILE_PATH\u003e\"' \\\n--form 'description=\"\u003cDESC\u003e\"' \\\n--header 'Authorization: Bearer \u003cAPI_KEY\u003e'"
                }
              ],
              header: 'Run this curl command in your terminal to push the artifact.',
              type: 'Static'
            }
          ],
          type: 'INLINE'
        },
        {
          header: 'Download Artifact',
          steps: [
            {
              commands: [
                {
                  label: '',
                  value:
                    "curl --location 'http://host.docker.internal:3000/generic/artifact-registry/generic-repo/\u003cARTIFACT_NAME\u003e:\u003cVERSION\u003e:\u003cFILENAME\u003e' --header 'Authorization: Bearer \u003cAPI_KEY\u003e' -J -O"
                }
              ],
              header: 'Run this command in your terminal to download the artifact.',
              type: 'Static'
            }
          ],
          type: 'INLINE'
        }
      ]
    },
    status: 'SUCCESS'
  }
}
