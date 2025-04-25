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

import type { StringKeys } from '@ar/frameworks/strings'
import { RepositoryConfigType, RepositoryPackageType } from '@ar/common/types'
import {
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

export class GenericRepositoryType extends RepositoryStep<VirtualRegistryRequest> {
  protected packageType = RepositoryPackageType.GENERIC
  protected repositoryName = 'Generic Repository'
  protected repositoryIcon: IconName = 'generic-repository-type'
  protected supportedScanners = []
  protected supportsUpstreamProxy = false
  enterpriseAdvancedOptionSubTitle: StringKeys =
    'repositoryDetails.repositoryForm.enterpriseAdvancedWithoutUpstreamOptionsSubTitle'

  protected defaultValues: VirtualRegistryRequest = {
    packageType: RepositoryPackageType.GENERIC,
    identifier: '',
    config: {
      type: RepositoryConfigType.VIRTUAL
    },
    scanners: []
  }

  protected defaultUpstreamProxyValues = null

  renderCreateForm(): JSX.Element {
    return <RepositoryCreateFormContent isEdit={false} />
  }

  renderCofigurationForm(props: RepositoryConfigurationFormProps<Repository>): JSX.Element {
    const { formikRef, readonly } = props
    return <RepositoryConfigurationForm ref={formikRef} readonly={readonly} />
  }

  renderActions(props: RepositoryActionsProps<Repository>): JSX.Element {
    const { data, readonly } = props
    return <RepositoryActions data={data} readonly={readonly} pageType={props.pageType} />
  }

  renderSetupClient(props: RepositoySetupClientProps): JSX.Element {
    const { repoKey, onClose, artifactKey, versionKey } = props
    return (
      <SetupClientContent
        repoKey={repoKey}
        artifactKey={artifactKey}
        versionKey={versionKey}
        onClose={onClose}
        packageType={RepositoryPackageType.GENERIC}
      />
    )
  }

  renderRepositoryDetailsHeader(props: RepositoryDetailsHeaderProps<Repository>): JSX.Element {
    const { data } = props
    return <RepositoryDetailsHeader data={data} />
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
