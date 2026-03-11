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

import React from 'react'
import type { IconName } from '@harnessio/icons'

import { RepositoryConfigType, RepositoryPackageType } from '@ar/common/types'
import {
  UpstreamProxyAuthenticationMode,
  UpstreamRegistryRequest,
  UpstreamRepositoryURLInputSource
} from '@ar/pages/upstream-proxy-details/types'
import type { VirtualRegistryRequest } from '@ar/pages/repository-details/types'
import type { CreateRepositoryFormProps } from '@ar/frameworks/RepositoryStep/Repository'
import UpstreamProxyCreateFormContent from '@ar/pages/upstream-proxy-details/components/FormContent/UpstreamProxyCreateFormContent'

import { RepositoryDetailsTab } from '../constants'
import RepositoryCreateFormContent from '../components/FormContent/RepositoryCreateFormContent'
import { GenericRepositoryType } from '../GenericRepository/GenericRepositoryType'

export class RawRepositoryType extends GenericRepositoryType {
  protected override packageType = RepositoryPackageType.RAW
  protected override repositoryName = 'Raw File Registry'
  protected override repositoryIcon: IconName = 'raw_icon'
  protected override supportsUpstreamProxy = true
  protected override supportedUpstreamURLSources = [UpstreamRepositoryURLInputSource.Custom]

  override supportedRepositoryTabs = [
    RepositoryDetailsTab.FILES,
    RepositoryDetailsTab.CONFIGURATION,
    RepositoryDetailsTab.METADATA,
    RepositoryDetailsTab.WEBHOOKS
  ]

  protected override defaultValues: VirtualRegistryRequest = {
    packageType: RepositoryPackageType.RAW,
    identifier: '',
    config: { type: RepositoryConfigType.VIRTUAL },
    scanners: [],
    isPublic: false
  }

  protected override defaultUpstreamProxyValues: UpstreamRegistryRequest = {
    packageType: RepositoryPackageType.RAW,
    identifier: '',
    config: {
      type: RepositoryConfigType.UPSTREAM,
      source: UpstreamRepositoryURLInputSource.Custom,
      authType: UpstreamProxyAuthenticationMode.ANONYMOUS,
      url: ''
    },
    cleanupPolicy: [],
    scanners: [],
    isPublic: false
  }

  override renderCreateForm(props: CreateRepositoryFormProps): JSX.Element {
    if (props.type === RepositoryConfigType.VIRTUAL) {
      return <RepositoryCreateFormContent isEdit={false} />
    }
    return <UpstreamProxyCreateFormContent isEdit={false} readonly={false} />
  }

  override processRepositoryFormData(values: VirtualRegistryRequest): VirtualRegistryRequest {
    const processed = super.processRepositoryFormData(values) as VirtualRegistryRequest
    return { ...processed, packageType: RepositoryPackageType.RAW }
  }
}
