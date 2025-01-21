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

import type { IconName } from '@harnessio/icons'
import type { WebhookExecResult } from '@harnessio/react-har-service-client'

import type { StringKeys } from '@ar/frameworks/strings'

interface StatusConfig {
  label: StringKeys
  iconName: IconName
  className: string
}
export const StatusConfigMap: Record<WebhookExecResult, StatusConfig> = {
  FATAL_ERROR: { label: 'failed', iconName: 'execution-error', className: 'error' },
  RETRIABLE_ERROR: { label: 'retriableError', iconName: 'execution-warning', className: 'error' },
  SUCCESS: { label: 'success', iconName: 'execution-success', className: 'success' }
}

export const UnknownStatusConfig: StatusConfig = {
  label: 'unknown',
  iconName: 'unknown-vehicle',
  className: 'unknown'
}
