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

import React, { useMemo } from 'react'
import { FiltersSelectDropDown } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { RepositoryScopeTypes } from '@ar/common/constants'
import type { EntityScope, RepositoryScopeType } from '@ar/common/types'

interface RepositoryScopeSelectorProps {
  scope: EntityScope
  value?: RepositoryScopeType
  onChange: (val: RepositoryScopeType) => void
}
export default function RepositoryScopeSelector(props: RepositoryScopeSelectorProps): JSX.Element {
  const { value, onChange, scope } = props
  const { getString } = useStrings()
  const options = useMemo(
    () =>
      RepositoryScopeTypes.filter(each => each.scope === scope).map(each => ({
        ...each,
        label: getString(each.label)
      })),
    [scope, getString]
  )
  const selectedOption = options.find(each => each.value === value)
  return (
    <FiltersSelectDropDown
      width={200}
      usePortal
      buttonTestId="registry-scope-select"
      items={options}
      value={selectedOption}
      onChange={option => {
        onChange(option.value as RepositoryScopeType)
      }}
      placeholder={getString('repositoryList.selectScope')}
      allowClearSelection={false}
    />
  )
}
