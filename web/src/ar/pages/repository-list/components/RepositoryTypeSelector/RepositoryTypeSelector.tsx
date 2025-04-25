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
import { DropDown } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { RepositoryConfigTypes } from '@ar/common/constants'
import type { RepositoryConfigType } from '@ar/common/types'

interface RepositoryTypeSelectorProps {
  value?: RepositoryConfigType
  onChange: (val: RepositoryConfigType) => void
}
export default function RepositoryTypeSelector(props: RepositoryTypeSelectorProps): JSX.Element {
  const { value, onChange } = props
  const { getString } = useStrings()
  return (
    <DropDown
      width={180}
      usePortal
      buttonTestId="registry-type-select"
      items={RepositoryConfigTypes.filter(each => !each.disabled).map(each => ({
        ...each,
        label: getString(each.label)
      }))}
      value={value}
      onChange={option => {
        onChange(option.value as RepositoryConfigType)
      }}
      placeholder={getString('repositoryList.selectRegistryType')}
      addClearBtn
    />
  )
}
