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

import { Color } from '@harnessio/design-system'
import type { IconProps } from '@harnessio/icons'
import type { Trigger, WebhookExecResult } from '@harnessio/react-har-service-client'

import type { StringKeys } from '@ar/frameworks/strings'

export const WebhookTriggerLabelMap: Record<Trigger, StringKeys> = {
  ARTIFACT_CREATION: 'webhookList.triggers.artifactCreation',
  ARTIFACT_DELETION: 'webhookList.triggers.artifactDeletion',
  ARTIFACT_MODIFICATION: 'webhookList.triggers.artifactModification'
}

export const WebhookStatusIconMap: Record<WebhookExecResult, IconProps> = {
  SUCCESS: { name: 'dot', color: Color.SUCCESS },
  RETRIABLE_ERROR: { name: 'dot', color: Color.WARNING },
  FATAL_ERROR: { name: 'dot', color: Color.ERROR }
}

export const DefaultStatusIconMap: IconProps = { name: 'dot', color: Color.GREY_500 }
