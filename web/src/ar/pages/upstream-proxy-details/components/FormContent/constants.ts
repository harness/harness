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

import type { StringKeys } from '@ar/frameworks/strings'
import { UpstreamProxyConfigFirewallModeEnum } from '../../types'

type Mode = {
  label: StringKeys
  subTitle: StringKeys
  value: UpstreamProxyConfigFirewallModeEnum
}

export const AllowedModes: Record<UpstreamProxyConfigFirewallModeEnum, Mode> = {
  [UpstreamProxyConfigFirewallModeEnum.WARN]: {
    label: 'repositoryDetails.repositoryForm.warn',
    subTitle: 'repositoryDetails.repositoryForm.warnSubtitle',
    value: UpstreamProxyConfigFirewallModeEnum.WARN
  },
  [UpstreamProxyConfigFirewallModeEnum.BLOCK]: {
    label: 'repositoryDetails.repositoryForm.block',
    subTitle: 'repositoryDetails.repositoryForm.blockSubtitle',
    value: UpstreamProxyConfigFirewallModeEnum.BLOCK
  },
  [UpstreamProxyConfigFirewallModeEnum.ALLOW]: {
    label: 'repositoryDetails.repositoryForm.allow',
    subTitle: 'repositoryDetails.repositoryForm.allowSubtitle',
    value: UpstreamProxyConfigFirewallModeEnum.ALLOW
  }
}
