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
import { MultiSelectDropDown, MultiSelectDropDownProps, type MultiSelectOption } from '@harnessio/uicore'
import type { IconName } from '@harnessio/icons'

interface Option<T> {
  label: string
  value: T
  icon?: IconName
}

interface MultiDropdownSelectProps<T> extends Omit<MultiSelectDropDownProps, 'value' | 'onSelect' | 'items'> {
  value: T[]
  onSelect: (val: T[]) => void
  items: Option<T>[]
}

function MultiSelectDropdownList<T>(props: MultiDropdownSelectProps<T>): JSX.Element {
  const { items, onSelect, className, value, placeholder, buttonTestId, ...rest } = props
  const actualValue = items.filter(each => value.includes(each.value))
  return (
    <MultiSelectDropDown
      minWidth={120}
      buttonTestId={buttonTestId}
      value={actualValue as MultiSelectOption[]}
      onChange={option => {
        onSelect(option.map(each => each.value as T))
      }}
      items={items as MultiSelectOption[]}
      usePortal={true}
      placeholder={placeholder}
      className={className}
      {...rest}
    />
  )
}

export default MultiSelectDropdownList
