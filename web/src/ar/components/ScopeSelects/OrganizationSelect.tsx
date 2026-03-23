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
import { useGetOrganizationsQuery } from '@harnessio/react-ng-manager-client'

import { useAppStore, useDebouncedValue } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'

const PAGE_SIZE = 200
const DEBOUNCE_MS = 400

export interface OrganizationSelectProps {
  value: string
  onChange: (value: string) => void
  name?: string
  label?: string
  placeholder?: string
  disabled?: boolean
  className?: string
}

export function OrganizationSelect({
  value,
  onChange,
  name = 'organization',
  label,
  placeholder,
  disabled,
  className
}: OrganizationSelectProps): JSX.Element {
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const accountId = scope?.accountId ?? ''

  const [query, setQuery] = useState('')
  const debouncedQuery = useDebouncedValue(query, DEBOUNCE_MS)

  const { data, isFetching, isLoading, error } = useGetOrganizationsQuery(
    {
      queryParams: {
        search_term: debouncedQuery || undefined,
        page: 0,
        limit: PAGE_SIZE
      }
    },
    { enabled: !!accountId }
  )

  const items: SelectOption[] = useMemo(() => {
    if (isFetching) {
      return [{ label: getString('loading'), value: '', disabled: true }]
    }
    if (error) {
      return [{ label: getErrorInfoFromErrorObject(error) ?? getString('failedToLoadData'), value: '', disabled: true }]
    }
    const content = data?.content
    const list = Array.isArray(content) ? content : []
    return list.map(item => ({
      label: item.org?.name ?? item.org?.identifier ?? '',
      value: item.org?.identifier ?? ''
    }))
  }, [data, isFetching, error, getString])

  const selectValue = items.find(o => o.value === value) ?? (value ? { label: value, value } : { label: '', value: '' })

  return (
    <Container className={className}>
      <FormInput.Select
        name={name}
        label={label ?? getString('versionDetails.copyVersionModal.organization')}
        items={items}
        value={selectValue}
        onChange={item => onChange(String(item?.value ?? ''))}
        onQueryChange={setQuery}
        selectProps={{ usePortal: true }}
        disabled={disabled || isLoading}
        addClearButton
        placeholder={
          isFetching
            ? getString('loading')
            : placeholder ?? getString('versionDetails.copyVersionModal.selectOrganization')
        }
      />
    </Container>
  )
}
