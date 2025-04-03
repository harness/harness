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

import type { ListWebhooks } from '@harnessio/react-har-service-client'

export const MOCK_WEBHOK_LIST_TABLE: ListWebhooks = {
  itemCount: 10,
  pageCount: 1,
  pageIndex: 0,
  pageSize: 10,
  webhooks: [
    {
      name: 'Webhook 1',
      identifier: 'webhook-1',
      triggers: ['ARTIFACT_CREATION'],
      url: 'https://webhook-1.com',
      insecure: false,
      enabled: true
    },
    {
      name: 'Webhook 2',
      identifier: 'webhook-2',
      triggers: ['ARTIFACT_CREATION', 'ARTIFACT_DELETION'],
      url: 'https://webhook-2.com',
      insecure: false,
      enabled: true
    },
    {
      name: 'Webhook 3',
      identifier: 'webhook-3',
      triggers: ['ARTIFACT_CREATION', 'ARTIFACT_DELETION'],
      url: 'https://webhook-3.com',
      insecure: false,
      enabled: true
    }
  ]
}
