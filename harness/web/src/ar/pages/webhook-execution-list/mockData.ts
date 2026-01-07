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

import type { ListWebhooksExecutions } from '@harnessio/react-har-service-client'

export const MOCK_WEBHOOK_EXECUTION_LIST: ListWebhooksExecutions = {
  pageCount: 1,
  pageIndex: 0,
  pageSize: 10,
  itemCount: 100,
  executions: [
    {
      id: 1,
      webhookId: 1,
      triggerType: 'ARTIFACT_CREATION',
      result: 'SUCCESS',
      created: 1631111111,
      duration: 1000,
      retriggerable: true
    },
    {
      id: 2,
      webhookId: 2,
      triggerType: 'ARTIFACT_CREATION',
      result: 'RETRIABLE_ERROR',
      created: 1631111112,
      duration: 2000,
      retriggerable: false
    },
    {
      id: 3,
      webhookId: 3,
      triggerType: 'ARTIFACT_CREATION',
      result: 'FATAL_ERROR',
      created: 1631111113,
      duration: 3000,
      retriggerable: false,
      error: 'Failed because of unknown error'
    }
  ]
}
