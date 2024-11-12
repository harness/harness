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
import { GitspaceStatusTypes, GitspaceStatusTypesListItem } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'
import MultiSelectDropdownList from '../MultiDropdownSelect/MultiDropdownSelect'

interface StatusDropdownProps {
  value: string[]
  onChange: (val: string[]) => void
}
export default function StatusDropdown(props: StatusDropdownProps): JSX.Element {
  const { value, onChange } = props
  const { getString } = useStrings()
  const dropdownList = GitspaceStatusTypes(getString)
  return (
    <MultiSelectDropdownList
      width={120}
      buttonTestId="gitspace-status-select"
      items={dropdownList.map((each: GitspaceStatusTypesListItem) => ({
        ...each,
        label: each.label
      }))}
      value={value}
      onSelect={onChange}
      placeholder={getString('status')}
    />
  )
}
