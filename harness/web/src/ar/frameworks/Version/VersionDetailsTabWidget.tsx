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

import React, { useEffect } from 'react'
import { Text } from '@harnessio/uicore'

import { useDecodedParams } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryPackageType } from '@ar/common/types'
import type { VersionDetailsTabPathParams } from '@ar/routes/types'
import type { VersionDetailsTab } from '@ar/pages/version-details/components/VersionDetailsTabs/constants'

import versionFactory from './VersionFactory'
import type { VersionDetailsTabProps } from './Version'
import type { VersionAbstractFactory } from './VersionAbstractFactory'

interface VersionDetailsTabWidgetProps extends VersionDetailsTabProps {
  packageType: RepositoryPackageType
  factory?: VersionAbstractFactory
  onInit?: (versionTab: VersionDetailsTab) => void
}

export default function VersionDetailsTabWidget<T>(props: VersionDetailsTabWidgetProps): JSX.Element {
  const { factory = versionFactory, packageType, tab } = props
  const { getString } = useStrings()
  const { versionTab } = useDecodedParams<VersionDetailsTabPathParams>()
  const repositoryType = factory?.getVersionType<T>(packageType)

  useEffect(() => {
    if (typeof props.onInit === 'function') {
      props.onInit(versionTab as VersionDetailsTab)
    }
  }, [versionTab])

  if (!repositoryType) {
    return <Text intent="warning">{getString('stepNotFound')}</Text>
  }
  return repositoryType.renderVersionDetailsTab({ tab })
}
