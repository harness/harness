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
  type UpstreamRegistryRequest,
  UpstreamRepositoryURLInputSource
} from '@ar/pages/upstream-proxy-details/types'
import {
  CreateRepositoryFormProps,
  RepositoryActionsProps,
  RepositoryConfigurationFormProps,
  RepositoryDetailsHeaderProps,
  RepositoryStep,
  RepositoryTreeNodeProps,
  RepositoySetupClientProps
} from '@ar/frameworks/RepositoryStep/Repository'

import RepositoryDetails from '../RepositoryDetails'
import type { Repository, VirtualRegistryRequest } from '../types'
import RepositoryActions from '../components/Actions/RepositoryActions'
import RedirectPageView from '../components/RedirectPageView/RedirectPageView'
import RepositoryTreeNode from '../components/RepositoryTreeNode/RepositoryTreeNode'
import SetupClientContent from '../components/SetupClientContent/SetupClientContent'
import RepositoryConfigurationForm from '../components/Forms/RepositoryConfigurationForm'
import RepositoryCreateFormContent from '../components/FormContent/RepositoryCreateFormContent'
import RepositoryDetailsHeader from '../components/RepositoryDetailsHeader/RepositoryDetailsHeader'

export class SwiftRepositoryType extends RepositoryStep<VirtualRegistryRequest> {
  protected packageType = RepositoryPackageType.SWIFT
  protected repositoryName = 'Swift Repository'
  protected repositoryIcon: IconName = 'swift-logo'
  protected supportedScanners = []
  protected supportsUpstreamProxy = false
  protected isWebhookSupported = true

  protected defaultValues: VirtualRegistryRequest = {
    packageType: RepositoryPackageType.SWIFT,
    identifier: '',
    config: {
      type: RepositoryConfigType.VIRTUAL
    },
    scanners: [],
    isPublic: false
  }

  protected defaultUpstreamProxyValues: UpstreamRegistryRequest = {
    packageType: RepositoryPackageType.SWIFT,
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

  renderCreateForm(_props: CreateRepositoryFormProps): JSX.Element {
    return <RepositoryCreateFormContent isEdit={false} />
  }

  renderCofigurationForm(props: RepositoryConfigurationFormProps<Repository>): JSX.Element {
    return <RepositoryConfigurationForm ref={props.formikRef} readonly={props.readonly} />
  }

  renderActions(props: RepositoryActionsProps<Repository>): JSX.Element {
    return <RepositoryActions data={props.data} readonly={props.readonly} pageType={props.pageType} />
  }

  renderSetupClient(props: RepositoySetupClientProps): JSX.Element {
    const { repoKey, onClose, artifactKey, versionKey } = props
    return (
      <SetupClientContent
        repoKey={repoKey}
        artifactKey={artifactKey}
        versionKey={versionKey}
        onClose={onClose}
        packageType={RepositoryPackageType.SWIFT}
      />
    )
  }

  renderRepositoryDetailsHeader(props: RepositoryDetailsHeaderProps<Repository>): JSX.Element {
    return <RepositoryDetailsHeader data={props.data} />
  }

  renderRedirectPage(): JSX.Element {
    return <RedirectPageView />
  }

  renderTreeNodeView(props: RepositoryTreeNodeProps): JSX.Element {
    return <RepositoryTreeNode {...props} icon={this.repositoryIcon} iconSize={20} />
  }

  renderTreeNodeDetails(): JSX.Element {
    return <RepositoryDetails />
  }
}
