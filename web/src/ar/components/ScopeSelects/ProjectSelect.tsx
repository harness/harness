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

import React, { useMemo, useState } from 'react'
import { Container, FormInput, getErrorInfoFromErrorObject } from '@harnessio/uicore'
import type { SelectOption } from '@harnessio/uicore'
import { useGetOrgScopedProjectsQuery } from '@harnessio/react-ng-manager-client'

import { useDebouncedValue } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

const PAGE_SIZE = 200
const DEBOUNCE_MS = 400

export interface ProjectSelectProps {
  org: string
  value: string
  onChange: (value: string) => void
  name?: string
  label?: string
  placeholder?: string
  disabled?: boolean
  className?: string
}

export function ProjectSelect({
  org,
  value,
  onChange,
  name = 'project',
  label,
  placeholder,
  disabled,
  className
}: ProjectSelectProps): JSX.Element {
  const { getString } = useStrings()
  const [query, setQuery] = useState('')
  const debouncedQuery = useDebouncedValue(query, DEBOUNCE_MS)

  const { data, isFetching, isLoading, error } = useGetOrgScopedProjectsQuery(
    {
      org,
      queryParams: {
        search_term: debouncedQuery || undefined,
        page: 0,
        limit: PAGE_SIZE
      }
    },
    { enabled: !!org }
  )

  const items: SelectOption[] = useMemo(() => {
    if (!org) {
      return []
    }
    if (isFetching) {
      return [{ label: getString('loading'), value: '', disabled: true }]
    }
    if (error) {
      return [{ label: getErrorInfoFromErrorObject(error) ?? getString('failedToLoadData'), value: '', disabled: true }]
    }
    const content = data?.content
    const list = Array.isArray(content) ? content : []
    return list.map(item => ({
      label: item.project?.name ?? item.project?.identifier ?? '',
      value: item.project?.identifier ?? ''
    }))
  }, [org, data, isFetching, error, getString])

  const selectValue = items.find(p => p.value === value) ?? (value ? { label: value, value } : { label: '', value: '' })

  return (
    <Container className={className}>
      <FormInput.Select
        key={org}
        name={name}
        label={label ?? getString('versionDetails.copyVersionModal.project')}
        items={items}
        value={selectValue}
        onChange={item => onChange(String(item?.value ?? ''))}
        onQueryChange={setQuery}
        selectProps={{ usePortal: true }}
        disabled={disabled || !org || isLoading}
        addClearButton
        placeholder={
          isFetching ? getString('loading') : placeholder ?? getString('versionDetails.copyVersionModal.selectProject')
        }
      />
    </Container>
  )
}
