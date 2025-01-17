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
import { Checkbox, FormikForm, FormInput, Layout } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { useAppStore, useParentComponents } from '@ar/hooks'

import FormLabel from './FormLabel'
import SelectTriggers from './SelectTriggers'
import type { WebhookRequestUI } from './types'
import ExtraHeadersFormContent from './ExtraHeadersFormContent/ExtraHeadersFormContent'

interface WebhookFormContentProps {
  formikProps: FormikProps<WebhookRequestUI>
  readonly?: boolean
  isEdit?: boolean
}

export default function WebhookFormContent(props: WebhookFormContentProps) {
  const { formikProps, readonly, isEdit } = props
  const { scope } = useAppStore()
  const { getString } = useStrings()
  const { SecretFormInput } = useParentComponents()
  const values = formikProps.values
  return (
    <FormikForm>
      <Layout.Vertical spacing="small">
        <FormInput.InputWithIdentifier
          inputName="name"
          idName="identifier"
          inputLabel={getString('webhookList.formFields.name')}
          idLabel={getString('webhookList.formFields.id')}
          inputGroupProps={{
            placeholder: getString('enterPlaceholder', { name: getString('webhookList.formFields.name') }),
            disabled: readonly
          }}
          isIdentifierEditable={!isEdit && !readonly}
        />
        <FormInput.TextArea
          name="description"
          label={getString('optionalField', { name: getString('webhookList.formFields.description') })}
          placeholder={getString('enterPlaceholder', { name: getString('webhookList.formFields.description') })}
          disabled={readonly}
        />
        <FormInput.Text
          name="url"
          label={getString('webhookList.formFields.url')}
          placeholder={getString('enterPlaceholder', { name: getString('webhookList.formFields.url') })}
          disabled={readonly}
        />
        <SecretFormInput
          name="secretIdentifier"
          spaceIdFieldName="secretSpaceId"
          label={getString('webhookList.formFields.secret')}
          placeholder={getString('enterPlaceholder', { name: getString('webhookList.formFields.secret') })}
          scope={scope}
          disabled={readonly}
          formik={formikProps}
        />
      </Layout.Vertical>
      <Layout.Vertical spacing="large">
        <SelectTriggers formikProps={formikProps} disabled={readonly} />
        <Layout.Vertical spacing="small">
          <FormLabel>{getString('webhookList.formFields.SSLVerification')}</FormLabel>
          <Checkbox
            label={getString('webhookList.formFields.enableSSLVerification')}
            checked={!values.insecure}
            disabled={readonly}
            onChange={e => {
              formikProps.setFieldValue('insecure', !e.currentTarget.checked)
            }}
          />
        </Layout.Vertical>
        {isEdit && (
          <Layout.Vertical spacing="small">
            <FormLabel>{getString('webhookList.formFields.enabled')}</FormLabel>
            <FormInput.CheckBox
              disabled={readonly}
              label={getString('webhookList.formFields.enabled')}
              name="enabled"
            />
          </Layout.Vertical>
        )}
        <Layout.Vertical spacing="small">
          <FormLabel>{getString('optionalField', { name: getString('webhookList.formFields.advanced') })}</FormLabel>
          <ExtraHeadersFormContent formikProps={formikProps} disabled={readonly} />
        </Layout.Vertical>
      </Layout.Vertical>
    </FormikForm>
  )
}
