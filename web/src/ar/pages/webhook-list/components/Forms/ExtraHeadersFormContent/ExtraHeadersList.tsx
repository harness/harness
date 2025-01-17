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
import type { FormikProps } from 'formik'
import type { ExtraHeader } from '@harnessio/react-har-service-client'
import { Button, ButtonSize, ButtonVariation, FormInput, Layout } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

import type { WebhookRequestUI } from '../types'

interface ExtraHeadersListProps {
  onAdd: (item: ExtraHeader) => void
  onRemove: (index: number) => void
  name: keyof WebhookRequestUI
  formikProps: FormikProps<WebhookRequestUI>
  disabled?: boolean
}

export default function ExtraHeadersList(props: ExtraHeadersListProps) {
  const { onAdd, onRemove, name, formikProps, disabled } = props
  const list = formikProps.values[name] as ExtraHeader[]
  const { getString } = useStrings()
  return (
    <Layout.Vertical flex={{ alignItems: 'flex-start' }}>
      {list?.map((_each: ExtraHeader, index: number) => (
        <Layout.Horizontal key={index} spacing="medium">
          <FormInput.Text
            inline
            name={`${name}[${index}].key`}
            label={getString('webhookList.formFields.extraHeader')}
            placeholder={getString('webhookList.formFields.extraHeaderPlaceholder')}
            disabled={disabled}
          />
          <FormInput.Text
            inline
            name={`${name}[${index}].value`}
            label={getString('webhookList.formFields.extraValue')}
            placeholder={getString('webhookList.formFields.extraValuePlaceholder')}
            disabled={disabled}
          />
          <Button variation={ButtonVariation.ICON} icon="code-delete" onClick={() => onRemove(index)} />
        </Layout.Horizontal>
      ))}
      <Button
        size={ButtonSize.SMALL}
        icon="plus"
        variation={ButtonVariation.LINK}
        onClick={() => onAdd({ key: '', value: '' })}>
        {getString('webhookList.formFields.addNewKeyValuePair')}
      </Button>
    </Layout.Vertical>
  )
}
