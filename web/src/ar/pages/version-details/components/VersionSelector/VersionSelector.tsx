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

import React, { useState } from 'react'
import { DropDown, SelectOption } from '@harnessio/uicore'
import { GetAllArtifactVersionsOkResponse, getAllArtifactVersions } from '@harnessio/react-har-service-client'

import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import HeaderTitle from '@ar/components/Header/Title'
import type { VersionDetailsPathParams } from '@ar/routes/types'

import css from './VersionSelector.module.scss'

export interface VersionSelectorProps {
  value?: string
  onChange(id: string): void
}

export default function VersionSelector(props: VersionSelectorProps): JSX.Element {
  const { value } = props
  const { artifactIdentifier } = useDecodedParams<VersionDetailsPathParams>()
  const [query, setQuery] = useState('')
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef()

  const refetchAllVersions = (): Promise<GetAllArtifactVersionsOkResponse> => {
    return getAllArtifactVersions({
      registry_ref: spaceRef,
      artifact: encodeRef(artifactIdentifier),
      queryParams: {
        size: 100,
        page: 0,
        search_term: query
      }
    })
  }

  const dummyPromise = (): Promise<SelectOption[]> => {
    return new Promise<SelectOption[]>(resolve => {
      refetchAllVersions()
        .then(result => {
          const selectItems = result?.content?.data?.artifactVersions?.map(item => {
            return {
              label: item.name || '',
              value: item.name || ''
            }
          }) as SelectOption[]
          resolve(selectItems || [])
        })
        .catch(() => {
          resolve([])
        })
    })
  }

  return (
    <DropDown
      buttonTestId="version-select"
      className={css.versionSelectorDropdown}
      onChange={option => {
        props.onChange(option.value as string)
      }}
      value={value}
      items={dummyPromise}
      usePortal={true}
      addClearBtn={false}
      query={query}
      onQueryChange={setQuery}
      placeholder={getString('artifactList.table.allRepositories')}
      isLabel
      getCustomLabel={option => (
        <HeaderTitle tag="span" className={css.primaryColor}>
          {option.label}
        </HeaderTitle>
      )}
    />
  )
}
