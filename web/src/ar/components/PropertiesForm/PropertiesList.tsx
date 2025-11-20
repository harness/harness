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
import { FontVariation } from '@harnessio/design-system'
import { Button, ButtonSize, ButtonVariation, Container, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import PropertiesFormInput from './FormInputs'
import type { PropertiesFormValues, PropertySpec } from './types'
import css from './PropertiesForm.module.scss'

interface PropertiesListProps {
  name: string
  addButtonLabel: string
  onAdd: () => void
  onRemove: (idx: number) => void
  disabled?: boolean
  supportQuery?: boolean
}

export default function PropertiesList(props: PropertiesListProps) {
  const { name, onAdd, onRemove, disabled, addButtonLabel, supportQuery } = props
  const formik = useFormikContext<PropertiesFormValues>()
  const { getString } = useStrings()
  const values = get(formik.values, name)
  return (
    <Layout.Vertical spacing="small" flex={{ alignItems: 'flex-start' }}>
      <Text font={{ variation: FontVariation.CARD_TITLE }}>{getString('customMetadata')}</Text>
      {values.length > 0 && (
        <Layout.Horizontal className={css.propertyRow} spacing="small">
          <Text font={{ variation: FontVariation.SMALL, weight: 'bold' }}>{getString('key')}</Text>
          <Text font={{ variation: FontVariation.SMALL, weight: 'bold' }}>{getString('value')}</Text>
          <Container />
        </Layout.Horizontal>
      )}
      {values.map((property: PropertySpec, index: number) => (
        <Layout.Horizontal className={css.propertyRow} key={index} spacing="small" onClick={e => e.stopPropagation()}>
          <PropertiesFormInput.KeyInput
            name={`${name}[${index}].key`}
            placeholder={getString('key')}
            disabled={disabled}
            supportQuery={supportQuery}
          />
          <PropertiesFormInput.ValueInput
            name={`${name}[${index}].value`}
            placeholder={getString('value')}
            disabled={disabled}
            supportQuery={supportQuery}
            propertyKey={property.key}
          />
          <Button icon="main-trash" minimal onClick={() => onRemove(index)} disabled={disabled} />
        </Layout.Horizontal>
      ))}
      <Button
        icon="plus"
        variation={ButtonVariation.SECONDARY}
        onClick={() => onAdd()}
        disabled={disabled}
        text={addButtonLabel}
        size={ButtonSize.SMALL}
      />
    </Layout.Vertical>
  )
}
