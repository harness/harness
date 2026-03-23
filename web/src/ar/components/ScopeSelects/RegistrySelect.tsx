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
import { useListRegistriesQuery } from '@harnessio/react-har-service-client'

import { useAppStore, useDebouncedValue } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import type { RepositoryConfigType } from '@ar/common/types'

const PAGE_SIZE = 500
const DEBOUNCE_MS = 400

interface RegistryItem {
  identifier: string
  uuid?: string
}

export interface RegistrySelectProps {
  value: string
  /** Called with (identifier, registryUuid). Use registryUuid for v3 copy API. */
  onChange: (identifier: string, registryUuid?: string) => void
  name?: string
  label?: string
  placeholder?: string
  disabled?: boolean
  className?: string
  /** Scope: org/project are optional filters */
  org?: string
  project?: string
  registryType?: RepositoryConfigType
  packageType?: string
}

export function RegistrySelect({
  value,
  onChange,
  name = 'registry',
  label,
  placeholder,
  disabled,
  className,
  org,
  project,
  registryType,
  packageType
}: RegistrySelectProps): JSX.Element {
  const { getString } = useStrings()
  const { scope } = useAppStore()
  const accountId = scope?.accountId ?? ''

  const [query, setQuery] = useState('')
  const debouncedQuery = useDebouncedValue(query, DEBOUNCE_MS)

  const { data, isFetching, isLoading, error } = useListRegistriesQuery(
    {
      queryParams: {
        account_identifier: accountId,
        org_identifier: org,
        project_identifier: project,
        search_term: debouncedQuery || undefined,
        page: 0,
        size: PAGE_SIZE,
        ...(registryType && { type: registryType }),
        ...(packageType && { package_type: [packageType] }),
        deleted: 'EXCLUDE'
      },
      stringifyQueryParamsOptions: { arrayFormat: 'repeat' }
    },
    { enabled: !!accountId }
  )

  const registriesList = useMemo((): RegistryItem[] => {
    const content = data?.content as { data?: { registries?: RegistryItem[] } } | undefined
    const registries = content?.data?.registries
    return Array.isArray(registries) ? registries : []
  }, [data])

  const items: SelectOption[] = useMemo(() => {
    if (isFetching) {
      return [{ label: getString('loading'), value: '', disabled: true }]
    }
    if (error) {
      return [{ label: getErrorInfoFromErrorObject(error) ?? getString('failedToLoadData'), value: '', disabled: true }]
    }
    return registriesList.map(item => ({
      label: item.identifier,
      value: item.uuid ?? item.identifier
    }))
  }, [registriesList, isFetching, data, error, getString])

  const selectValue = items.find(r => r.value === value) ?? (value ? { label: value, value } : { label: '', value: '' })

  const handleChange = (item: SelectOption | null) => {
    const uuid = String(item?.value ?? '')
    const identifier = String(item?.label ?? '')
    onChange(identifier, uuid)
  }

  return (
    <Container className={className}>
      <FormInput.Select
        key={`${org ?? ''}-${project ?? ''}`}
        name={name}
        label={label ?? getString('versionDetails.copyVersionModal.registry')}
        items={items}
        value={selectValue}
        onChange={handleChange}
        onQueryChange={setQuery}
        selectProps={{ usePortal: true }}
        disabled={disabled || isLoading}
        placeholder={
          isFetching ? getString('loading') : placeholder ?? getString('versionDetails.copyVersionModal.selectRegistry')
        }
      />
    </Container>
  )
}
