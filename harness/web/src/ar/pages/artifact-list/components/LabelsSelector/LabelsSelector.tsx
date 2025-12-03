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
import classNames from 'classnames'
import { MultiSelectDropDown, MultiSelectDropDownProps } from '@harnessio/uicore'
import { useListArtifactLabelsQuery } from '@harnessio/react-har-service-client'

import { useStrings } from '@ar/frameworks/strings'
import { useGetSpaceRef } from '@ar/hooks'

import css from './LabelSelector.module.scss'

interface LabelsSelectorProps extends Omit<MultiSelectDropDownProps, 'value' | 'onChange' | 'items'> {
  value: string[]
  onChange: (val: string[]) => void
}

export default function LabelsSelector(props: LabelsSelectorProps): JSX.Element {
  const { value, onChange, ...rest } = props
  const { getString } = useStrings()
  const actualValue = value.map(each => ({ label: each, value: each }))
  const [query, setQuery] = useState('')
  const spaceRef = useGetSpaceRef()

  const {
    data,
    isFetching: loading,
    error
  } = useListArtifactLabelsQuery({
    registry_ref: spaceRef,
    queryParams: {
      page: 0,
      size: 100,
      search_term: query
    }
  })

  const getOptions = () => {
    if (loading) {
      return [{ label: 'Loading...', value: '' }]
    }
    if (error) {
      return [{ label: error.message, value: '' }]
    }
    if (data && Array.isArray(data?.content.data.labels)) {
      return data.content.data.labels.map(each => ({
        label: each,
        value: each
      }))
    }
    return []
  }
  return (
    <MultiSelectDropDown
      minWidth={120}
      popoverClassName={classNames(css.labelsSelectorPopover, {
        [css.itemDisabled]: loading || !!error
      })}
      buttonTestId="label-select"
      value={actualValue}
      onChange={option => {
        onChange(option.map(each => each.value) as string[])
      }}
      items={getOptions()}
      usePortal={true}
      placeholder={getString('repositoryList.selectLabels')}
      allowSearch
      query={query}
      onQueryChange={setQuery}
      itemDisabled={() => loading || !!error}
      {...rest}
    />
  )
}
