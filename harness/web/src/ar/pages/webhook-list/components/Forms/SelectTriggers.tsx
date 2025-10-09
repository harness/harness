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
import { Container, FormInput, Layout } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import CheckboxGroup from '@ar/components/Form/CheckboxGroup/CheckboxGroup'

import FormLabel from './FormLabel'
import type { WebhookRequestUI } from './types'
import { TriggerLabelOptions } from './constants'

import css from './Forms.module.scss'

interface SelectTriggersProps {
  formikProps: FormikProps<WebhookRequestUI>
  disabled?: boolean
}

export default function SelectTriggers(props: SelectTriggersProps) {
  const { formikProps, disabled } = props
  const triggerType = formikProps.values.triggerType
  const { getString } = useStrings()
  return (
    <Layout.Vertical spacing="small">
      <FormLabel>{getString('webhookList.formFields.triggerLabel')}</FormLabel>
      <FormInput.RadioGroup
        key={triggerType}
        className={css.triggerType}
        disabled={disabled}
        name="triggerType"
        items={[
          { label: getString('webhookList.formFields.allTrigger'), value: 'all' },
          { label: getString('webhookList.formFields.customTrigger'), value: 'custom' }
        ]}
      />
      {triggerType === 'custom' && (
        <Container margin={{ left: 'large' }}>
          <CheckboxGroup
            disabled={disabled}
            items={TriggerLabelOptions.map(each => ({ ...each, label: getString(each.label) }))}
            name="triggers"
          />
        </Container>
      )}
    </Layout.Vertical>
  )
}
