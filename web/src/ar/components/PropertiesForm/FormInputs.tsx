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
import { get } from 'lodash-es'
import { useFormikContext } from 'formik'
import { MenuItem } from '@blueprintjs/core'
import { FormInput, getErrorInfoFromErrorObject, SelectOption } from '@harnessio/uicore'
import { getMetadataKeys, getMetadataValues } from '@harnessio/react-har-service-v2-client'

import { useAppStore } from '@ar/hooks'
import { useStrings } from '@ar/frameworks/strings'
import { DEFAULT_PAGE_INDEX, DEFAULT_PAGE_SIZE } from '@ar/constants'

import type { PropertiesFormValues } from './types'

interface KeyInputProps {
  name: string
  placeholder: string
  disabled?: boolean
  supportQuery?: boolean
}

function KeyInput(props: KeyInputProps): JSX.Element {
  const { name, placeholder, disabled, supportQuery } = props
  const [items, setItems] = useState<SelectOption[]>([])
  const formik = useFormikContext<PropertiesFormValues>()
  const value = get(formik.values, name)
  const { getString } = useStrings()
  const { scope } = useAppStore()

  const fetchItems = async (query: string) => {
    try {
      setItems([{ label: getString('loading'), value: '' }])
      const response = await getMetadataKeys({
        queryParams: {
          account_identifier: scope.accountId as string,
          org_identifier: scope.orgIdentifier,
          project_identifier: scope.projectIdentifier,
          size: DEFAULT_PAGE_SIZE,
          page: DEFAULT_PAGE_INDEX,
          search_term: query
        }
      })
      setItems(response.content?.data?.map((item: string) => ({ label: item, value: item })) || [])
    } catch (e: any) {
      setItems([{ label: getErrorInfoFromErrorObject(e) ?? getString('failedToLoadData'), value: '' }])
    }
  }

  if (!supportQuery) {
    return <FormInput.Text name={name} placeholder={placeholder} disabled={disabled} />
  }
  return (
    <FormInput.Select
      items={items}
      value={{ label: value, value }}
      selectProps={{
        allowCreatingNewItems: true,
        itemRenderer: (item, itemProps) => {
          return <MenuItem text={item.label} onClick={itemProps.handleClick} disabled={item.value === ''} />
        },
        inputProps: {
          onFocus: () => fetchItems(''),
          placeholder
        }
      }}
      name={name}
      disabled={disabled}
      onQueryChange={query => fetchItems(query)}
    />
  )
}

interface ValueInputProps {
  name: string
  placeholder: string
  disabled?: boolean
  propertyKey: string
  supportQuery?: boolean
}

function ValueInput(props: ValueInputProps): JSX.Element {
  const { name, placeholder, disabled, supportQuery, propertyKey } = props
  const [items, setItems] = useState<SelectOption[]>([])
  const formik = useFormikContext<PropertiesFormValues>()
  const value = get(formik.values, name)
  const { getString } = useStrings()
  const { scope } = useAppStore()

  const fetchItems = async (query: string) => {
    try {
      setItems([{ label: getString('loading'), value: '' }])
      const response = await getMetadataValues({
        queryParams: {
          account_identifier: scope.accountId as string,
          org_identifier: scope.orgIdentifier,
          project_identifier: scope.projectIdentifier,
          size: DEFAULT_PAGE_SIZE,
          page: DEFAULT_PAGE_INDEX,
          search_term: query,
          key: propertyKey
        }
      })
      setItems(response.content?.data?.map((item: string) => ({ label: item, value: item })) || [])
    } catch (e: any) {
      setItems([{ label: getErrorInfoFromErrorObject(e) ?? getString('failedToLoadData'), value: '' }])
    }
  }

  if (!supportQuery) {
    return <FormInput.Text name={name} placeholder={placeholder} disabled={disabled} />
  }
  return (
    <FormInput.Select
      items={items}
      value={{ label: value, value }}
      selectProps={{
        allowCreatingNewItems: true,
        inputProps: {
          onFocus: () => fetchItems(''),
          placeholder
        },
        itemRenderer: (item, itemProps) => {
          return <MenuItem text={item.label} onClick={itemProps.handleClick} disabled={item.value === ''} />
        }
      }}
      name={name}
      disabled={disabled}
      onQueryChange={query => fetchItems(query)}
    />
  )
}

const PropertiesFormInput = {
  KeyInput,
  ValueInput
}

export default PropertiesFormInput
