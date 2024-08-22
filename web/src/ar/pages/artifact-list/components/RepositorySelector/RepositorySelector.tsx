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
import type { SelectOption } from '@harnessio/uicore'
import { DropDown } from '@harnessio/uicore'
import { getAllRegistries } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useGetSpaceRef } from '@ar/hooks'

export interface RepositorySelectorProps {
  value?: string
  onChange(id: string): void
}

export default function RepositorySelector(props: RepositorySelectorProps): JSX.Element {
  const [query, setQuery] = React.useState('')
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef()

  const queryRepositories = async (): Promise<SelectOption[]> => {
    return getAllRegistries({
      space_ref: spaceRef,
      queryParams: {
        size: 10,
        page: 0,
        search_term: query
      }
    })
      .then(result => {
        const selectItems = result?.content?.data?.registries?.map(item => {
          return { label: item.identifier, value: item.identifier }
        }) as SelectOption[]
        return selectItems || []
      })
      .catch(() => {
        return []
      })
  }

  return (
    <DropDown
      minWidth={120}
      buttonTestId="pipeline-select"
      onChange={option => {
        props.onChange(option.value as string)
      }}
      value={props.value}
      items={queryRepositories}
      usePortal={true}
      addClearBtn={true}
      query={query}
      onQueryChange={setQuery}
      placeholder={getString('artifactList.table.allRepositories')}
    />
  )
}
