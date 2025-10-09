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

import React, { forwardRef } from 'react'
import { Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import type { FormikFowardRef, RepositoryPackageType } from '@ar/common/types'
import repositoryFactory from './RepositoryFactory'
import type { RepositoryAbstractFactory } from './RepositoryAbstractFactory'
import type { RepositoryConfigurationFormProps } from './Repository'

interface RepositoryConfigurationFormWidgetProps<T> extends RepositoryConfigurationFormProps<T> {
  factory?: RepositoryAbstractFactory
  packageType: RepositoryPackageType
}

function RepositoryConfigurationFormWidget<T>(
  props: RepositoryConfigurationFormWidgetProps<T>,
  formikRef: FormikFowardRef
): JSX.Element {
  const { factory = repositoryFactory, packageType, readonly, type } = props
  const { getString } = useStrings()
  const repositoryType = factory?.getRepositoryType<T>(packageType)
  if (!repositoryType) {
    return <Text intent="warning">{getString('stepNotFound')}</Text>
  }
  return repositoryType.renderCofigurationForm({
    readonly,
    formikRef,
    type
  })
}

export default forwardRef(RepositoryConfigurationFormWidget)
