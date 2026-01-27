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
import UpstreamProxyActions from '@ar/pages/upstream-proxy-details/components/UpstreamProxyActions/UpstreamProxyActions'
import UpstreamProxyConfigurationForm from '@ar/pages/upstream-proxy-details/components/Forms/UpstreamProxyConfigurationForm'
import UpstreamProxyCreateFormContent from '@ar/pages/upstream-proxy-details/components/FormContent/UpstreamProxyCreateFormContent'
import UpstreamProxyDetailsHeader from '@ar/pages/upstream-proxy-details/components/UpstreamProxyDetailsHeader/UpstreamProxyDetailsHeader'
import {
  type CreateRepositoryFormProps,
  type RepositoryActionsProps,
  type RepositoryConfigurationFormProps,
  type RepositoryDetailsHeaderProps,
  RepositoryStep,
  type RepositoryTreeNodeProps,
  type RepositoySetupClientProps
} from '@ar/frameworks/RepositoryStep/Repository'
import {
  UpstreamProxyAuthenticationMode,
  type UpstreamRegistryRequest,
  UpstreamRepositoryURLInputSource
} from '@ar/pages/upstream-proxy-details/types'

import RepositoryDetails from '../RepositoryDetails'
import type { Repository, VirtualRegistryRequest } from '../types'
import RepositoryActions from '../components/Actions/RepositoryActions'
import RedirectPageView from '../components/RedirectPageView/RedirectPageView'
import SetupClientContent from '../components/SetupClientContent/SetupClientContent'
import RepositoryTreeNode from '../components/RepositoryTreeNode/RepositoryTreeNode'
import RepositoryConfigurationForm from '../components/Forms/RepositoryConfigurationForm'
import RepositoryCreateFormContent from '../components/FormContent/RepositoryCreateFormContent'
import RepositoryDetailsHeader from '../components/RepositoryDetailsHeader/RepositoryDetailsHeader'

export class CargoRepositoryType extends RepositoryStep<VirtualRegistryRequest> {
  protected packageType = RepositoryPackageType.CARGO
  protected repositoryName = 'Cargo Repository'
  protected repositoryIcon: IconName = 'rust-logo'
  protected supportedScanners = []
  protected supportsUpstreamProxy = true
  protected isWebhookSupported = true
  protected supportedUpstreamURLSources = [
    UpstreamRepositoryURLInputSource.Crates,
    UpstreamRepositoryURLInputSource.Custom
  ]
  protected isDependencyFirewallSupported = true

  protected defaultValues: VirtualRegistryRequest = {
    packageType: RepositoryPackageType.CARGO,
    identifier: '',
    config: {
      type: RepositoryConfigType.VIRTUAL
    },
    scanners: [],
    isPublic: false
  }

  protected defaultUpstreamProxyValues: UpstreamRegistryRequest = {
    packageType: RepositoryPackageType.CARGO,
    identifier: '',
    config: {
      type: RepositoryConfigType.UPSTREAM,
      source: UpstreamRepositoryURLInputSource.Crates,
      authType: UpstreamProxyAuthenticationMode.ANONYMOUS,
      url: ''
    },
    cleanupPolicy: [],
    scanners: [],
    isPublic: false
  }

  renderCreateForm(props: CreateRepositoryFormProps): JSX.Element {
    const { type } = props
    if (type === RepositoryConfigType.VIRTUAL) {
      return <RepositoryCreateFormContent isEdit={false} />
    } else {
      return <UpstreamProxyCreateFormContent isEdit={false} readonly={false} />
    }
  }

  renderCofigurationForm(props: RepositoryConfigurationFormProps<Repository>): JSX.Element {
    const { type } = props
    if (type === RepositoryConfigType.VIRTUAL) {
      return <RepositoryConfigurationForm ref={props.formikRef} readonly={props.readonly} />
    } else {
      return <UpstreamProxyConfigurationForm ref={props.formikRef} readonly={props.readonly} />
    }
  }

  renderActions(props: RepositoryActionsProps<Repository>): JSX.Element {
    if (props.type === RepositoryConfigType.VIRTUAL) {
      return <RepositoryActions data={props.data} readonly={props.readonly} pageType={props.pageType} />
    }
    return <UpstreamProxyActions data={props.data} readonly={props.readonly} pageType={props.pageType} />
  }

  renderSetupClient(props: RepositoySetupClientProps): JSX.Element {
    const { repoKey, onClose, artifactKey, versionKey } = props
    return (
      <SetupClientContent
        repoKey={repoKey}
        artifactKey={artifactKey}
        versionKey={versionKey}
        onClose={onClose}
        packageType={RepositoryPackageType.CARGO}
      />
    )
  }

  renderRepositoryDetailsHeader(props: RepositoryDetailsHeaderProps<Repository>): JSX.Element {
    const { type } = props
    if (type === RepositoryConfigType.VIRTUAL) {
      return <RepositoryDetailsHeader data={props.data} />
    } else {
      return <UpstreamProxyDetailsHeader data={props.data} />
    }
  }

  renderRedirectPage(): JSX.Element {
    return <RedirectPageView />
  }

  renderTreeNodeView(props: RepositoryTreeNodeProps): JSX.Element {
    return <RepositoryTreeNode {...props} icon={this.repositoryIcon} />
  }

  renderTreeNodeDetails(): JSX.Element {
    return <RepositoryDetails />
  }
}
