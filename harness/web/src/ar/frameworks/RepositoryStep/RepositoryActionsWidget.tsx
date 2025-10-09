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

import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import repositoryFactory from './RepositoryFactory'
import type { RepositoryAbstractFactory } from './RepositoryAbstractFactory'
import type { RepositoryActionsProps } from './Repository'

interface RepositoryActionsWidgetProps<T> extends RepositoryActionsProps<T> {
  factory?: RepositoryAbstractFactory
  packageType: RepositoryPackageType
}

export default function RepositoryActionsWidget<T>(props: RepositoryActionsWidgetProps<T>): JSX.Element {
  const { factory = repositoryFactory, packageType, type, data, readonly, pageType } = props
  const { getString } = useStrings()
  const repositoryType = factory?.getRepositoryType(packageType)
  if (!repositoryType) {
    return <Text intent="warning">{getString('stepNotFound')}</Text>
  }
  return repositoryType.renderActions({
    data,
    readonly,
    type,
    pageType
  })
}
