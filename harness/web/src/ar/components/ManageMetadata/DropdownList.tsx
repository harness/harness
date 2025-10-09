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

import React, { useEffect, useMemo, useState } from 'react'
import { Menu, MenuItem } from '@blueprintjs/core'
import { getErrorInfoFromErrorObject, Layout } from '@harnessio/uicore'

import { useDebouncedValue } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

import SearchInput from './SearchInput'

import css from './ManageMetadata.module.scss'

interface DropdownListProps<T> {
  getItems: (searchTerm: string) => Promise<Array<T>>
  renderItem: (item: T, index: number) => React.ReactNode
  renderNewItem?: (item: string) => React.ReactNode
  getRowId: (item: T) => string
  getRowValue: (item: T) => string
  onSelect: (item: T) => void
  shouldAllowCreateNewOption?: boolean
  onCreateNewOption?: (optionName: string) => void
  placeholder?: string
  leftElement?: React.ReactNode
}

function DropdownList<T>(props: DropdownListProps<T>) {
  const [query, setQuery] = useState('')
  const [options, setOptions] = useState<Array<T>>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const debouncedQuery = useDebouncedValue(query, 500)

  const { getString } = useStrings()

  const handleSearch = async (searchTerm: string) => {
    setLoading(true)
    setQuery(searchTerm)
    try {
      const res = await props.getItems(searchTerm)
      setOptions(res)
      setError(null)
    } catch (e: any) {
      setError(getErrorInfoFromErrorObject(e))
    } finally {
      setLoading(false)
    }
  }

  const shouldShowCreateNewOption = useMemo(() => {
    if (!props.shouldAllowCreateNewOption) return false
    if (!query) return false
    if (options.length === 0) return true
    return !options.some(each => props.getRowValue(each).toLowerCase() === query.toLowerCase())
  }, [options, query, props.shouldAllowCreateNewOption])

  useEffect(() => {
    handleSearch(debouncedQuery)
  }, [debouncedQuery])

  return (
    <Layout.Vertical padding="small" spacing="small">
      <SearchInput
        autoFocus
        placeholder={props.placeholder || getString('search')}
        onChange={setQuery}
        value={query}
        leftElement={props.leftElement}
      />
      <Menu className={css.menuContainer}>
        {loading && <MenuItem disabled text={getString('loading')} />}
        {error && <MenuItem disabled text={error} />}
        {!loading && !error && options.length === 0 && !shouldShowCreateNewOption && (
          <MenuItem disabled text={getString('noResultsFound')} />
        )}
        {!loading &&
          !error &&
          options.map((each, index) => (
            <MenuItem
              key={props.getRowId(each)}
              text={props.renderItem(each, index)}
              onClick={() => props.onSelect(each)}
              shouldDismissPopover={false}
            />
          ))}
        {!loading && !error && shouldShowCreateNewOption && (
          <MenuItem
            text={
              props.renderNewItem
                ? props.renderNewItem(query)
                : getString('labels.addNewValueDynamic', { value: query })
            }
            onClick={() => props.onCreateNewOption?.(query)}
            shouldDismissPopover={false}
          />
        )}
      </Menu>
    </Layout.Vertical>
  )
}

export default DropdownList
