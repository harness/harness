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

export const mockRepositoryListApiResponse: GetAllRegistriesOkResponse = {
  content: {
    data: {
      registries: [
        {
          identifier: 'repo1',
          packageType: 'DOCKER',
          type: 'VIRTUAL',
          url: 'space/repo1',
          isPublic: false
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
          artifactsCount: 100,
          isPublic: false
        },
        {
          identifier: 'upstream_1',
          packageType: 'DOCKER',
          type: 'UPSTREAM',
          url: 'space/upstream_1',
          isPublic: false
        }
      ],
      itemCount: 2,
      pageCount: 10,
      pageIndex: 0,
      pageSize: 10
    },
    status: 'SUCCESS'
  }
}
