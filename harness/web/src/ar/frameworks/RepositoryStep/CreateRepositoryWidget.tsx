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
import { Text } from '@harnessio/uicore'
import type { PackageType } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import repositoryFactory from './RepositoryFactory'
import type { RepositoryAbstractFactory } from './RepositoryAbstractFactory'
import type { CreateRepositoryFormProps } from './Repository'

interface CreateRepositoryWidgetProps extends CreateRepositoryFormProps {
  factory?: RepositoryAbstractFactory
  packageType: PackageType
}

export default function CreateRepositoryWidget(props: CreateRepositoryWidgetProps): JSX.Element {
  const { factory = repositoryFactory, type, packageType } = props
  const { getString } = useStrings()
  const repositoryType = factory?.getRepositoryType(packageType)
  if (!repositoryType) {
    return <Text intent="warning">{getString('stepNotFound')}</Text>
  }
  return repositoryType.renderCreateForm({ type })
}
