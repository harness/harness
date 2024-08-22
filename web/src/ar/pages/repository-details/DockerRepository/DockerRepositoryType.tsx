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

import { RepositoryStep } from '@ar/frameworks/RepositoryStep/Repository'
import type {
  CreateRepositoryFormProps,
  RepositoryActionsProps,
  RepositoryConfigurationFormProps,
  RepositoryDetailsHeaderProps,
  RepositoySetupClientProps
} from '@ar/frameworks/RepositoryStep/Repository'

import { RepositoryConfigType, RepositoryPackageType } from '@ar/common/types'
import UpstreamProxyDetailsHeader from '@ar/pages/upstream-proxy-details/components/UpstreamProxyDetailsHeader/UpstreamProxyDetailsHeader'
import UpstreamProxyActions from '@ar/pages/upstream-proxy-details/components/UpstreamProxyActions/UpstreamProxyActions'
import UpstreamProxyConfigurationForm from '@ar/pages/upstream-proxy-details/components/Forms/UpstreamProxyConfigurationForm'
import UpstreamProxyCreateFormContent from '@ar/pages/upstream-proxy-details/components/FormContent/UpstreamProxyCreateFormContent'
import {
  DockerRepositoryURLInputSource,
  UpstreamProxyAuthenticationMode,
  UpstreamRegistryRequest
} from '@ar/pages/upstream-proxy-details/types'
import type { Repository, VirtualRegistryRequest } from '@ar/pages/repository-details/types'

import RepositoryActions from '../components/Actions/RepositoryActions'
import RepositoryConfigurationForm from '../components/Forms/RepositoryConfigurationForm'
import SetupClientContent from '../components/SetupClientContent/SetupClientContent'
import RepositoryCreateFormContent from '../components/FormContent/RepositoryCreateFormContent'
import RepositoryDetailsHeader from '../components/RepositoryDetailsHeader/RepositoryDetailsHeader'

export class DockerRepositoryType extends RepositoryStep<VirtualRegistryRequest> {
  protected packageType = RepositoryPackageType.DOCKER
  protected repositoryName = 'Docker Repository'
  protected repositoryIcon: IconName = 'docker-step'

  protected defaultValues: VirtualRegistryRequest = {
    packageType: RepositoryPackageType.DOCKER,
    identifier: '',
    config: {
      type: RepositoryConfigType.VIRTUAL,
      upstreamProxies: []
    },
    cleanupPolicy: []
  }

  protected defaultUpstreamProxyValues: UpstreamRegistryRequest = {
    packageType: RepositoryPackageType.DOCKER,
    identifier: '',
    config: {
      type: RepositoryConfigType.UPSTREAM,
      source: DockerRepositoryURLInputSource.Dockerhub,
      url: '',
      authType: UpstreamProxyAuthenticationMode.ANONYMOUS
    },
    cleanupPolicy: []
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
        packageType={RepositoryPackageType.DOCKER}
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
}
