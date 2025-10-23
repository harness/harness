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

import { useStrings } from '@ar/frameworks/strings'
import type { EnvironmentType } from '@ar/common/types'
import { EnvironmentTypeList } from '@ar/common/constants'
import MultiSelectDropdownList from '@ar/components/MultiDropdownSelect/MultiDropdownSelect'

interface EnvironmentTypeSelectorProps {
  value: EnvironmentType[]
  onChange: (val: EnvironmentType[]) => void
}

export default function EnvironmentTypeSelector(props: EnvironmentTypeSelectorProps): JSX.Element {
  const { value, onChange } = props
  const { getString } = useStrings()
  return (
    <MultiSelectDropdownList
      buttonTestId="environment-type-select"
      items={EnvironmentTypeList.filter(each => !each.disabled).map(each => ({
        ...each,
        label: getString(each.label)
      }))}
      value={value}
      onSelect={onChange}
      placeholder={getString('repositoryList.selectEnvironments')}
      allowSearch
    />
  )
}
