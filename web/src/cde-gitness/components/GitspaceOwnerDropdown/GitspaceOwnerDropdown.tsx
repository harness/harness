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
import { GitspaceOwnerType, GitspaceOwnerTypes } from 'cde-gitness/constants'
import { useStrings } from 'framework/strings'

interface GitspaceOwnerDropdownProps {
  value?: string
  onChange: (val: GitspaceOwnerType) => void
}
export default function GitspaceOwnerDropdown(props: GitspaceOwnerDropdownProps): JSX.Element {
  const { value, onChange } = props
  const { getString } = useStrings()
  const dropdownList = GitspaceOwnerTypes(getString)
  return (
    <DropDown
      width={180}
      buttonTestId="gitspace-owner-select"
      items={dropdownList}
      value={value}
      onChange={option => {
        onChange(option.value as GitspaceOwnerType)
      }}
      placeholder={getString('cde.owners')}
    />
  )
}
