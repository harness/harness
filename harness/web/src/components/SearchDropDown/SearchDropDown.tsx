/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React, { useState, useEffect } from 'react'
import cx from 'classnames'
import { Container, Select, SelectOption } from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color } from '@harnessio/design-system'
import { Avatar } from '@harnessio/uicore'
import { useStrings } from 'framework/strings'
import { CodeIcon } from 'utils/GitUtils'
import { PrincipalType } from 'utils/Utils'
import { useAppContext } from 'AppContext'
import css from './SearchDropDown.module.scss'

export const renderPrincipalIcon = (type: PrincipalType, displayName: string) => {
  switch (type) {
    case PrincipalType.USER_GROUP:
      return <Icon name="user-groups" className={cx(css.avatar, css.icon, css.ugicon)} size={24} />
    case PrincipalType.SERVICE_ACCOUNT:
      return <Icon name="service-accounts" className={cx(css.avatar, css.icon, css.saicon)} size={24} />
    case PrincipalType.USER:
    default:
      return <Avatar className={css.avatar} name={displayName} size="normal" hoverCard={false} />
  }
}

interface SearchDropDownProps {
  searchTerm: string
  placeholder: string
  className?: string
  popoverClassName?: string
  options: SelectOption[]
  onClick: (val: SelectOption) => void
  onChange: (val: string) => void
  itemRenderer?: (item: SelectOption, props: { handleClick: () => void; isActive?: boolean }) => React.ReactNode
  loading?: boolean
}

export default function SearchDropDown({
  searchTerm,
  placeholder,
  className,
  popoverClassName,
  options,
  onClick,
  onChange,
  itemRenderer,
  loading
}: SearchDropDownProps) {
  const { standalone } = useAppContext()
  const { getString } = useStrings()
  const [query, setQuery] = useState(searchTerm || '')

  // Keep searchTerm in sync
  useEffect(() => {
    if (searchTerm !== query) {
      setQuery(searchTerm || '')
    }
  }, [searchTerm, query])

  // Handle search query changes
  const handleQueryChange = (queryString: string) => {
    setQuery(queryString)
    onChange(queryString)
  }

  // Custom wrapper for the itemRenderer that works with BP's expectations
  const wrapItemRenderer = itemRenderer
    ? (item: SelectOption, itemProps: any): React.ReactElement | null => {
        const handleClick = () => {
          try {
            itemProps.handleClick()
          } catch (e) {
            // eslint-disable-next-line no-empty
          }
        }
        const isActive = itemProps.modifiers?.active === true
        const rendered = itemRenderer(item, {
          handleClick,
          isActive
        })

        return rendered as React.ReactElement | null
      }
    : undefined

  return (
    <Container className={cx(css.dropDownContainer, className)}>
      <Select
        loadingItems={loading}
        items={options}
        onChange={onClick}
        onQueryChange={handleQueryChange}
        itemRenderer={wrapItemRenderer}
        value={{ label: '', value: '' }}
        inputProps={{
          placeholder: standalone ? getString('selectUsers') : placeholder,
          value: query,
          leftElement: (
            <Icon name={loading ? CodeIcon.InputSpinner : CodeIcon.InputSearch} color={Color.GREY_500} size={12} />
          )
        }}
        popoverClassName={cx(css.popoverHeight, popoverClassName)}
        resetOnSelect={true}
        resetOnClose={false}
      />
    </Container>
  )
}
