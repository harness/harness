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

import React, { ReactNode, useMemo, useState } from 'react'
import classNames from 'classnames'
import { Menu } from '@blueprintjs/core'
import { FontVariation } from '@harnessio/design-system'
import { ExpandingSearchInput, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import SelectListMenuItem from './SelectListMenuItem'
import type { RorderSelectOption } from './types'
import css from './ReorderSelect.module.scss'

interface SelectListProps {
  options: RorderSelectOption[]
  onSelect: (option: RorderSelectOption) => void
  withSearch?: boolean
  title?: string | ReactNode
  disabled: boolean
}

const EmptyOption: RorderSelectOption = {
  label: 'No data',
  value: '',
  disabled: true
}

function SelectList(props: SelectListProps): JSX.Element {
  const { options, onSelect, withSearch, title, disabled } = props
  const [searchTerm, setSearchTerm] = useState('')
  const { getString } = useStrings()

  const filteredOptions: RorderSelectOption[] = useMemo(() => {
    try {
      return options.filter(each => each.label.toLowerCase().includes(searchTerm.toLowerCase()))
    } catch {
      return options
    }
  }, [options, searchTerm])

  const renderTitle = (): JSX.Element | ReactNode => {
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

  const renderOptions = (): JSX.Element => {
    if (filteredOptions.length) {
      return (
        <>
          {filteredOptions.map((each, index) => (
            <SelectListMenuItem key={`${each.value}_${index}`} option={each} onClick={onSelect} disabled={disabled} />
          ))}
        </>
      )
    }
    return <SelectListMenuItem option={EmptyOption} onClick={onSelect} disabled={disabled} />
  }

  return (
    <Layout.Vertical spacing="xsmall">
      {renderTitle()}
      {withSearch && (
        <ExpandingSearchInput
          className={css.searchInput}
          autoFocus={false}
          alwaysExpanded
          placeholder={getString('search')}
          onChange={text => {
            setSearchTerm(text)
          }}
          defaultValue={searchTerm}
          disabled={disabled}
        />
      )}
      <Menu aria-label="selectable-list" className={css.listContainer}>
        {renderOptions()}
      </Menu>
    </Layout.Vertical>
  )
}

export default SelectList
