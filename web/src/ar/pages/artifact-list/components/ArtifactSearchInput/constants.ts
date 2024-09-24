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

import type { AIOption } from '@ar/components/AISearchInput/types'

export const ArtifactListAIOptions: AIOption[] = [
  {
    label: 'Need help? Here are some prompts example:',
    value: '-1',
    disabled: true
  },
  {
    label: 'Show me artifacts not downloaded in last 30 days',
    value: 'Show me artifacts not downloaded in last 30 days'
  },
  {
    label: 'Find out artifacts in prod but not deployed in lower environments',
    value: 'Find out artifacts in prod but not deployed in lower environments'
  },
  {
    label: 'List artifacts in prod but with critical vulnerabilities',
    value: 'List artifacts in prod but with critical vulnerabilities'
  }
]
