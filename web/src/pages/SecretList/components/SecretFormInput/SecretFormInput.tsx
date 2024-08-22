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
import { get } from 'lodash-es'
import { useFormikContext } from 'formik'
import { useMemo, useState } from 'react'
import { ButtonVariation, FormInput, Layout, SelectOption } from '@harnessio/uicore'

import { useStrings } from 'framework/strings'
import { useGetSecretList } from 'pages/SecretList/hooks/useGetSecretList'
import { NewSecretModalButton } from 'components/NewSecretModalButton/NewSecretModalButton'

import css from './SecretFormInput.module.scss'

interface SecretFormInputProps {
  name: string
  spaceIdFieldName: string
  scope: Record<string, string>
  label?: React.ReactNode
  placeholder?: string
  disabled?: boolean
}

export default function SecretFormInput(props: SecretFormInputProps) {
  const { name, label, placeholder, disabled, scope, spaceIdFieldName } = props
  const [searchTerm, setSearchTerm] = useState('')

  const { getString } = useStrings()

  const formik = useFormikContext()

  const { data, loading, error, refetch } = useGetSecretList({
    space: scope.space,
    queryParams: {
      limit: 100,
      query: searchTerm
    }
  })

  const items = useMemo(() => {
    if (loading) {
      return [{ label: getString('loading'), value: '-1' }]
    }
    if (error) {
      return [{ label: error.message, value: '-1' }]
    }
    if (data?.length) {
      return data.map(each => ({
        label: each.identifier,
        value: each.identifier,
        spaceId: each.space_id
      }))
    }
    return []
  }, [loading, error, data])

  const formikValue = get(formik.values, name, '')

  const selectedValue = formikValue ? { label: formikValue, value: formikValue } : null

  return (
    <Layout.Horizontal spacing="small" flex={{ justifyContent: 'flex-start', alignItems: 'center' }}>
      <FormInput.Select
        name={name}
        label={label}
        disabled={disabled}
        value={selectedValue}
        placeholder={placeholder}
        items={items as SelectOption[]}
        onQueryChange={query => setSearchTerm(query)}
        onChange={option => {
          formik.setFieldValue(name, option.value)
          formik.setFieldValue(spaceIdFieldName, get(option, 'spaceId'))
        }}
        selectProps={{
          loadingItems: loading,
          itemDisabled: () => !!loading || !!error?.message
        }}
      />
      <NewSecretModalButton
        className={css.createNewBtn}
        space={scope.space}
        modalTitle={getString('secrets.create')}
        text={getString('secrets.newSecretButton')}
        variation={ButtonVariation.LINK}
        icon="plus"
        onSuccess={secret => {
          refetch()
          formik.setFieldValue(name, secret.identifier)
          formik.setFieldValue(spaceIdFieldName, secret.space_id)
        }}
      />
    </Layout.Horizontal>
  )
}
