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

import React, { useEffect, useMemo } from 'react'
import { debounce } from 'lodash-es'
import { DropDown, SelectOption } from '@harnessio/uicore'
import { useGetDockerArtifactManifestsQuery } from '@harnessio/react-har-service-client'

import { encodeRef } from '@ar/hooks/useGetSpaceRef'
import { useDecodedParams, useGetSpaceRef } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import HeaderTitle from '@ar/components/Header/Title'
import type { VersionDetailsPathParams } from '@ar/routes/types'

import css from './ArchitectureSelector.module.scss'

export interface ArchitectureSelectorProps {
  value?: string
  onChange(id: string): void
  shouldSelectFirstByDefault: boolean
  version: string
}

export default function ArchitectureSelector(props: ArchitectureSelectorProps): JSX.Element {
  const { value, shouldSelectFirstByDefault = true, version } = props
  const { artifactIdentifier } = useDecodedParams<VersionDetailsPathParams>()
  const { getString } = useStrings()
  const spaceRef = useGetSpaceRef()

  const {
    data,
    isFetching: loading,
    error
  } = useGetDockerArtifactManifestsQuery({
    registry_ref: spaceRef,
    artifact: encodeRef(artifactIdentifier),
    version: version
  })

  const deboucedOnChange = debounce(props.onChange, 100)

  const responseData = data?.content?.data?.manifests || []

  // select first option by default if flag shouldSelectFirstByDefault is true
  useEffect(() => {
    if (!shouldSelectFirstByDefault) return
    if (value) return
    if (Array.isArray(responseData) && responseData.length) {
      deboucedOnChange(responseData[0].digest)
    }
  }, [data, value, responseData, shouldSelectFirstByDefault])

  const listOptions: SelectOption[] = useMemo(() => {
    if (loading) {
      return [{ label: 'Loading', value: '' }]
    }
    if (error) {
      return [{ label: error.message, value: '' }]
    }
    if (!responseData) return []
    return responseData.map(item => ({
      label: item.osArch || '',
      value: item.digest || ''
    }))
  }, [loading, error, responseData])

  return (
    <DropDown
      buttonTestId="version-arch-select"
      className={css.versionSelectorDropdown}
      onChange={option => {
        props.onChange(option.value as string)
      }}
      value={value}
      items={listOptions}
      usePortal={true}
      addClearBtn={false}
      itemDisabled={item => !item.value}
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
