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
import { Menu } from '@blueprintjs/core'
import { Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import MenuItemAction from './MenuItemAction'
import type { RorderSelectOption } from './types'

import css from './ReorderSelect.module.scss'

interface SelectListMenuItemProps {
  option: RorderSelectOption
  onClick: (option: RorderSelectOption) => void
  disabled: boolean
}

function SelectListMenuItem(props: SelectListMenuItemProps): JSX.Element {
  const { option, onClick, disabled: gDisabled } = props
  const { label, disabled: lDisabled } = option
  const { getString } = useStrings()
  const disabled = lDisabled || gDisabled
  return (
    <Menu.Item
      className={css.menuItem}
      disabled={disabled}
      text={
        <Layout.Horizontal flex={{ justifyContent: 'space-between', alignItems: 'center' }}>
          <Text color="black">{label ? label : getString('na')}</Text>
          {!disabled && <MenuItemAction rightIcon="main-chevron-right" text={getString('add')} />}
        </Layout.Horizontal>
      }
      onClick={() => onClick(option)}
    />
  )
}

export default SelectListMenuItem
