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

import React, { useMemo } from 'react'
import { Menu, MenuItem } from '@blueprintjs/core'
import type { SelectOption } from '@harnessio/uicore'
import { useGetAllRegistriesQuery } from '@harnessio/react-har-service-client'
import { getErrorInfoFromErrorObject, MultiSelectDropDown } from '@harnessio/uicore'

import { useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

export interface RepositorySelectorProps {
  value?: string[]
  onChange(ids: string[]): void
}

export default function RepositorySelector(props: RepositorySelectorProps): JSX.Element {
  const [query, setQuery] = React.useState('')
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef()

  const { data, isFetching, error } = useGetAllRegistriesQuery({
    space_ref: spaceRef,
    queryParams: {
      size: 10,
      page: 0,
      search_term: query
    }
  })

  const items: SelectOption[] = useMemo(() => {
    if (isFetching) {
      return [{ label: getString('loading'), value: '-1', disabled: true }]
    }
    if (error) {
      return [{ label: getErrorInfoFromErrorObject(error), value: '-1', disabled: true }]
    }
    if (data?.content.data.registries.length === 0) {
      return [{ label: getString('noResultsFound'), value: '-1', disabled: true }]
    }
    return (
      data?.content.data.registries?.map(item => {
        return { label: item.identifier, value: item.identifier }
      }) || []
    )
  }, [data, isFetching, error])

  const selectedValues = useMemo(() => {
    if (Array.isArray(props.value)) {
      return props.value.map(each => ({
        label: each,
        value: each
      }))
    }
    return []
  }, [props.value])

  return (
    <MultiSelectDropDown
      width={180}
      buttonTestId="regitry-select"
      onChange={options => {
        if (!Array.isArray(options)) {
          props.onChange([])
          return
        }
        props.onChange(options.map(each => each.value as string))
      }}
      value={selectedValues}
      items={items}
      usePortal={true}
      allowSearch
      expandingSearchInputProps={{
        defaultValue: query,
        onChange: q => setQuery(q)
      }}
      itemListRenderer={itemListProps => {
        return (
          <Menu>
            {itemListProps.items.map((item, idx) => {
              if (item.disabled) {
                return <MenuItem key={item.label} text={item.label} disabled />
              }
              return itemListProps.renderItem(item, idx)
            })}
          </Menu>
        )
      }}
      placeholder={getString('artifactList.table.allRepositories')}
      onPopoverClose={() => setQuery('')}
    />
  )
}
