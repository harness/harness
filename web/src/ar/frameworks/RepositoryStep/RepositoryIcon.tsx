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
import { defaultTo } from 'lodash-es'
import { Icon, IconProps } from '@harnessio/icons'
import { Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import type { RepositoryAbstractFactory } from './RepositoryAbstractFactory'
import repositoryFactory from './RepositoryFactory'

interface RepositoryIconProps {
  packageType: RepositoryPackageType
  factory?: RepositoryAbstractFactory
  iconProps?: Omit<IconProps, 'name'>
}

export default function RepositoryIcon(props: RepositoryIconProps): JSX.Element {
  const { packageType, iconProps, factory = repositoryFactory } = props
  const { getString } = useStrings()
  const repositoryType = factory?.getRepositoryType(packageType as RepositoryPackageType)
  if (!repositoryType) {
    return <Text intent="warning">{getString('stepNotFound')}</Text>
  }
  return (
    <Icon
      key={packageType}
      name={repositoryType.getIconName()}
      size={defaultTo(iconProps?.size, 16)}
      inverse={iconProps?.inverse}
    />
  )
}
