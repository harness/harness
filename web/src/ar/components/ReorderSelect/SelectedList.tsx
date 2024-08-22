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

import React, { useRef } from 'react'
import classNames from 'classnames'
import { noop } from 'lodash-es'
import { FontVariation } from '@harnessio/design-system'
import { Layout, Text } from '@harnessio/uicore'
import { Menu } from '@blueprintjs/core'

import SelectedListMenuItem from './SelectedListMenuItem'
import type { RorderSelectOption } from './types'
import css from './ReorderSelect.module.scss'

interface SelectedListProps {
  options: RorderSelectOption[]
  onReorder: (options: RorderSelectOption[]) => void
  onRemove: (option: RorderSelectOption) => void
  note?: React.ReactNode
  title?: string | React.ReactNode
  disabled: boolean
}

const EmptyOption: RorderSelectOption = {
  label: 'No data',
  value: '',
  disabled: true
}

function SelectedList(props: SelectedListProps): JSX.Element {
  const { options, onReorder, note, title, onRemove, disabled } = props
  const draggingPos = useRef<number | null>(null)
  const dragOverPos = useRef<number | null>(null)

  const handleDragStart = (position: number): void => {
    draggingPos.current = position
  }

  const handleDragEnter = (position: number): void => {
    dragOverPos.current = position
    const newItems = [...options]
    if (draggingPos.current === null) return
    const draggingItem = newItems[draggingPos.current]
    if (!draggingItem) return

    newItems.splice(draggingPos.current, 1)
    newItems.splice(dragOverPos.current, 0, draggingItem)

    draggingPos.current = position
    dragOverPos.current = null
    onReorder(newItems)
  }

  const renderTitle = (): JSX.Element | React.ReactNode => {
    if (typeof title === 'string') {
      return (
        <Text
          lineClamp={1}
          className={classNames(css.message, css.defaultBackground)}
          font={{ variation: FontVariation.SMALL }}>
          {title}
        </Text>
      )
    }
    return title
  }

  const renderNote = (): React.ReactNode => {
    if (note) return note
    return null
  }

  const renderOptions = (): React.ReactNode => {
    if (options.length) {
      return options.map((each, index) => (
        <SelectedListMenuItem
          key={`${each.value}_${index}`}
          option={each}
          onDragStart={() => handleDragStart(index)}
          onDragEnter={() => handleDragEnter(index)}
          onRemove={onRemove}
          disabled={disabled}
        />
      ))
    }
    return (
      <SelectedListMenuItem
        option={EmptyOption}
        onDragStart={noop}
        onDragEnter={noop}
        onRemove={noop}
        disabled={disabled}
      />
    )
  }

  return (
    <Layout.Vertical spacing="xsmall">
      {renderTitle()}
      {renderNote()}
      <Menu aria-label="orderable-list" className={css.listContainer}>
        {renderOptions()}
      </Menu>
    </Layout.Vertical>
  )
}

export default SelectedList
