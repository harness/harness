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

import { createContext } from 'react'
import type { Webhook } from '@harnessio/react-har-service-client'

interface WebhookDetailsContextSpec {
  data?: Webhook
  loading?: boolean
  setDirty?: (dirty: boolean) => void
  setUpdating?: (updating: boolean) => void
}

export const WebhookDetailsContext = createContext<WebhookDetailsContextSpec>({
  data: {} as Webhook,
  loading: false
})
