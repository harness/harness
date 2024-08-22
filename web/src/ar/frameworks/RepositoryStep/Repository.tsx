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

import type { IconName } from '@harnessio/icons'
import type { FormikFowardRef, RepositoryPackageType, RepositoryConfigType, PageType } from '@ar/common/types'

export interface CreateRepositoryFormProps {
  type: RepositoryConfigType
}

export interface RepositoryConfigurationFormProps<T> {
  readonly: boolean
  formikRef?: FormikFowardRef<T>
  type: RepositoryConfigType
}

export interface RepositoryActionsProps<T> {
  data: T
  readonly: boolean
  type: RepositoryConfigType
  pageType: PageType
}

export interface RepositoySetupClientProps {
  onClose: () => void
  repoKey: string
  artifactKey?: string
  versionKey?: string
}

export interface RepositoryDetailsHeaderProps<T> {
  data: T
  type: RepositoryConfigType
}

export abstract class RepositoryStep<T, U = unknown> {
  protected abstract packageType: RepositoryPackageType
  protected abstract repositoryName: string
  protected abstract defaultValues: T
  protected abstract defaultUpstreamProxyValues: U
  protected abstract repositoryIcon: IconName
  protected repositoryIconColor?: string
  protected repositoryIconSize?: number

  getPackageType(): string {
    return this.packageType
  }

  getDefaultValues(initialValues: T): T {
    return { ...this.defaultValues, ...initialValues }
  }

  getUpstreamProxyDefaultValues(initialValues: U): U {
    return {
      ...this.defaultUpstreamProxyValues,
      ...initialValues
    }
  }

  getIconName(): IconName {
    return this.repositoryIcon
  }

  getIconColor(): string | undefined {
    return this.repositoryIconColor
  }

  getIconSize(): number | undefined {
    return this.repositoryIconSize
  }

  getStepName(): string {
    return this.repositoryName
  }

  getRepositoryInitialValues(data: T): T {
    return data
  }

  getUpstreamProxyInitialValues(data: U): U {
    return data
  }

  processRepositoryFormData(values: U): U {
    return values
  }

  processUpstreamProxyFormData(values: T): T {
    return values
  }

  abstract renderCreateForm(props: CreateRepositoryFormProps): JSX.Element

  abstract renderCofigurationForm(props: RepositoryConfigurationFormProps<U>): JSX.Element

  abstract renderActions(props: RepositoryActionsProps<U>): JSX.Element

  abstract renderSetupClient(props: RepositoySetupClientProps): JSX.Element

  abstract renderRepositoryDetailsHeader(props: RepositoryDetailsHeaderProps<U>): JSX.Element
}
