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

import produce from 'immer'
import { set } from 'lodash-es'
import {
  DockerRepositoryURLInputSource,
  UpstreamProxyAuthenticationMode,
  type UpstreamRegistryRequest
} from '../../types'

export function getFormattedFormDataForAuthType(values: UpstreamRegistryRequest): UpstreamRegistryRequest {
  return produce(values, (draft: UpstreamRegistryRequest) => {
    if (draft.config.authType === UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD) {
      set(draft, 'config.auth.authType', draft.config.authType)
    }
    if (draft.config.source !== DockerRepositoryURLInputSource.Custom) {
      set(draft, 'config.url', '')
    }
  })
}
