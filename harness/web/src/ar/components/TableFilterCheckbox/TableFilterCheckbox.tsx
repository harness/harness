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
import classNames from 'classnames'
import { Checkbox } from '@harnessio/uicore'
import { Color } from '@harnessio/design-system'

import css from './TableFilterCheckbox.module.scss'

interface TableFilterCheckboxProps {
  value: boolean
  label: string
  disabled: boolean
  onChange: (val: boolean) => void
}

export default function TableFilterCheckbox(props: TableFilterCheckboxProps): JSX.Element {
  const { value, onChange, label, disabled } = props
  return (
    <Checkbox
      font={{ size: 'small', weight: 'semi-bold' }}
      color={Color.GREY_800}
      label={label}
      checked={value}
      disabled={disabled}
      onChange={e => onChange(e.currentTarget.checked)}
      className={classNames(css.filterCheckbox, { [css.selected]: value })}
    />
  )
}
