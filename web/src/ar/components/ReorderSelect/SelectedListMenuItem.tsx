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
import { Expander, Menu } from '@blueprintjs/core'
import { Layout, Text } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'

import { useStrings } from '@ar/frameworks/strings'
import MenuItemAction from './MenuItemAction'
import type { RorderSelectOption } from './types'

import css from './ReorderSelect.module.scss'

interface SelectedListMenuItemProps {
  option: RorderSelectOption
  onDragStart: (option: RorderSelectOption) => void
  onDragEnter: (option: RorderSelectOption) => void
  onRemove: (option: RorderSelectOption) => void
  disabled: boolean
}

function SelectedListMenuItem(props: SelectedListMenuItemProps): JSX.Element {
  const { option, onDragStart, onDragEnter, onRemove, disabled: gDisabled } = props
  const { label, disabled: lDisabled } = option
  const { getString } = useStrings()
  const disabled = lDisabled || gDisabled
  return (
    <Menu.Item
      className={css.menuItem}
      disabled={disabled}
      draggable={!disabled}
      onDragStart={() => onDragStart(option)}
      onDragEnter={() => onDragEnter(option)}
      onDragOver={e => e.preventDefault()}
      text={
        <Layout.Horizontal spacing="small" flex={{ justifyContent: 'flex-start', alignItems: 'center' }}>
          {!disabled && <Icon name="drag-handle-vertical" size={12} />}
          <Text color="black">{label ? label : getString('na')}</Text>
          <Expander />
          {!disabled && (
            <MenuItemAction
              rightIcon="cross"
              onClick={e => {
                e.stopPropagation()
                onRemove(option)
              }}
            />
          )}
        </Layout.Horizontal>
      }
    />
  )
}

export default SelectedListMenuItem
