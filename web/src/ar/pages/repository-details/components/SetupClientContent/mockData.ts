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

import type { ClientSetupDetails } from '@harnessio/react-har-service-client'

export const MOCK_SETUP_CLIENT_RESPONSE: ClientSetupDetails = {
  mainHeader: 'Docker Client Setup',
  secHeader: 'Follow these instructions to install/use Docker artifacts or compatible packages.',
  sections: [
    {
      header: 'Login to Docker',
      secHeader: 'Run this Docker command in your terminal to login to Docker.',
      type: 'INLINE',
      steps: [
        {
          header: 'For the Password field above, generate an identity token',
          type: 'GenerateToken'
        }
      ]
    },
    {
      header: 'Retag and Push the image',
      type: 'TABS',
      tabs: [
        {
          header: 'Maven',
          sections: [
            {
              header: 'Login to Maven',
              secHeader: 'Run this Docker command in your terminal to login to Docker.',
              type: 'INLINE',
              steps: [
                {
                  commands: [
                    {
                      label: '',
                      value:
                        'docker tag \u003cIMAGE_NAME\u003e:\u003cTAG\u003e host.docker.internal:3000/artifact-registry/docker-repo/\u003cIMAGE_NAME\u003e:\u003cTAG\u003e'
                    }
                  ],
                  header: 'Run this Docker command in your terminal to tag the image.',
                  type: 'Static'
                },
                {
                  commands: [
                    {
                      label: '',
                      value:
                        'docker push host.docker.internal:3000/artifact-registry/docker-repo/\u003cIMAGE_NAME\u003e:\u003cTAG\u003e'
                    }
                  ],
                  header: 'Run this Docker command in your terminal to push the image.',
                  type: 'Static'
                }
              ]
            }
          ]
        },
        {
          header: 'Gradle',
          sections: [
            {
              header: 'Login to Gradle',
              type: 'INLINE',
              steps: [
                {
                  commands: [
                    {
                      label: '',
                      value:
                        'docker tag \u003cIMAGE_NAME\u003e:\u003cTAG\u003e host.docker.internal:3000/artifact-registry/docker-repo/\u003cIMAGE_NAME\u003e:\u003cTAG\u003e'
                    }
                  ],
                  header: 'Run this Docker command in your terminal to tag the image.',
                  type: 'Static'
                },
                {
                  commands: [
                    {
                      label: '',
                      value:
                        'docker push host.docker.internal:3000/artifact-registry/docker-repo/\u003cIMAGE_NAME\u003e:\u003cTAG\u003e'
                    }
                  ],
                  header: 'Run this Docker command in your terminal to push the image.',
                  type: 'Static'
                }
              ]
            }
          ]
        },
        {
          header: 'Scala',
          sections: [
            {
              header: 'Login to Scala',
              type: 'INLINE',
              steps: [
                {
                  commands: [
                    {
                      label: '',
                      value:
                        'docker tag \u003cIMAGE_NAME\u003e:\u003cTAG\u003e host.docker.internal:3000/artifact-registry/docker-repo/\u003cIMAGE_NAME\u003e:\u003cTAG\u003e'
                    }
                  ],
                  header: 'Run this Docker command in your terminal to tag the image.',
                  type: 'Static'
                },
                {
                  commands: [
                    {
                      label: '',
                      value:
                        'docker push host.docker.internal:3000/artifact-registry/docker-repo/\u003cIMAGE_NAME\u003e:\u003cTAG\u003e'
                    }
                  ],
                  header: 'Run this Docker command in your terminal to push the image.',
                  type: 'Static'
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}
