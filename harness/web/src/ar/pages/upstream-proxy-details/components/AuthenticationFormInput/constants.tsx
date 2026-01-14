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

import type { StringsMap } from '@ar/strings/types'
import { UpstreamProxyAuthenticationMode, UpstreamRepositoryURLInputSource } from '../../types'

interface RadioGroupItem {
  label: keyof StringsMap
  subLabel?: keyof StringsMap
  value: UpstreamProxyAuthenticationMode
}

export const AuthTypeRadioItems: Record<UpstreamProxyAuthenticationMode, RadioGroupItem> = {
  [UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD]: {
    label: 'upstreamProxyDetails.createForm.authentication.userNameAndPassword',
    value: UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD
  },
  [UpstreamProxyAuthenticationMode.ANONYMOUS]: {
    label: 'upstreamProxyDetails.createForm.authentication.anonymous',
    subLabel: 'upstreamProxyDetails.createForm.authentication.anonymousSubLabel',
    value: UpstreamProxyAuthenticationMode.ANONYMOUS
  },
  [UpstreamProxyAuthenticationMode.ACCESS_KEY_AND_SECRET_KEY]: {
    label: 'upstreamProxyDetails.createForm.authentication.accessKeyAndSecretKey',
    value: UpstreamProxyAuthenticationMode.ACCESS_KEY_AND_SECRET_KEY
  }
}

export const URLSourceToSupportedAuthTypesMapping: Record<
  UpstreamRepositoryURLInputSource,
  UpstreamProxyAuthenticationMode[]
> = {
  [UpstreamRepositoryURLInputSource.Dockerhub]: [
    UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
    UpstreamProxyAuthenticationMode.ANONYMOUS
  ],
  [UpstreamRepositoryURLInputSource.AwsEcr]: [
    UpstreamProxyAuthenticationMode.ACCESS_KEY_AND_SECRET_KEY,
    UpstreamProxyAuthenticationMode.ANONYMOUS
  ],
  [UpstreamRepositoryURLInputSource.Custom]: [
    UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
    UpstreamProxyAuthenticationMode.ANONYMOUS
  ],
  [UpstreamRepositoryURLInputSource.MavenCentral]: [
    UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
    UpstreamProxyAuthenticationMode.ANONYMOUS
  ],
  [UpstreamRepositoryURLInputSource.NpmJS]: [
    UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
    UpstreamProxyAuthenticationMode.ANONYMOUS
  ],
  [UpstreamRepositoryURLInputSource.PyPi]: [
    UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
    UpstreamProxyAuthenticationMode.ANONYMOUS
  ],
  [UpstreamRepositoryURLInputSource.NugetOrg]: [
    UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
    UpstreamProxyAuthenticationMode.ANONYMOUS
  ],
  [UpstreamRepositoryURLInputSource.Crates]: [
    UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
    UpstreamProxyAuthenticationMode.ANONYMOUS
  ],
  [UpstreamRepositoryURLInputSource.GoProxy]: [
    UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
    UpstreamProxyAuthenticationMode.ANONYMOUS
  ],
  [UpstreamRepositoryURLInputSource.HuggingFace]: [
    UpstreamProxyAuthenticationMode.USER_NAME_AND_PASSWORD,
    UpstreamProxyAuthenticationMode.ANONYMOUS
  ]
}
