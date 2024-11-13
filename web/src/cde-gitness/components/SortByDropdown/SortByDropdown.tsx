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
import { SortByTypes } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import type { EnumGitspaceSort } from 'services/cde'

interface SortByDropdownProps {
  value?: EnumGitspaceSort
  onChange: (val: EnumGitspaceSort) => void
}
export default function SortByDropdown(props: SortByDropdownProps): JSX.Element {
  const { value, onChange } = props
  const { getString } = useStrings()
  const dropdownList = SortByTypes(getString)
  return (
    <DropDown
      width={180}
      buttonTestId="gitspace-sort-select"
      items={dropdownList}
      value={value}
      onChange={option => {
        onChange(option.value as EnumGitspaceSort)
      }}
      placeholder={getString('cde.sortBy')}
      addClearBtn
    />
  )
}
