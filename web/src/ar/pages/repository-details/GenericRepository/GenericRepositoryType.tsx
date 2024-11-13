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
import UpstreamProxyCreateFormContent from '@ar/pages/upstream-proxy-details/components/FormContent/UpstreamProxyCreateFormContent'
import { RepositoryConfigType, RepositoryPackageType } from '@ar/common/types'
import { CreateRepositoryFormProps, RepositoryStep } from '@ar/frameworks/RepositoryStep/Repository'
import type { VirtualRegistryRequest } from '../types'
import RepositoryCreateFormContent from '../components/FormContent/RepositoryCreateFormContent'

export class GenericRepositoryType extends RepositoryStep<VirtualRegistryRequest> {
  protected packageType = RepositoryPackageType.GENERIC
  protected repositoryName = 'Generic Repository'
  protected repositoryIcon: IconName = 'generic-repository-type'
  protected supportedScanners = []

  protected defaultValues: VirtualRegistryRequest = {
    packageType: RepositoryPackageType.GENERIC,
    identifier: '',
    config: {
      type: RepositoryConfigType.VIRTUAL
    },
    scanners: []
  }

  protected defaultUpstreamProxyValues = null

  renderCreateForm(props: CreateRepositoryFormProps): JSX.Element {
    const { type } = props
    if (type === RepositoryConfigType.VIRTUAL) {
      return <RepositoryCreateFormContent isEdit={false} />
    } else {
      return <UpstreamProxyCreateFormContent isEdit={false} readonly={false} />
    }
  }

  renderCofigurationForm(): JSX.Element {
    return <></>
  }

  renderActions(): JSX.Element {
    return <></>
  }

  renderSetupClient(): JSX.Element {
    return <></>
  }

  renderRepositoryDetailsHeader(): JSX.Element {
    return <></>
  }
}
