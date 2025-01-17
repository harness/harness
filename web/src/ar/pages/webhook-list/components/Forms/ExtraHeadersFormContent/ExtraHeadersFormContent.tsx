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
import { FieldArray, FormikProps } from 'formik'
import { Layout } from '@harnessio/uicore'

import type { WebhookRequestUI } from '../types'
import ExtraHeadersList from './ExtraHeadersList'

interface ExtraHeadersFormContentProps {
  formikProps: FormikProps<WebhookRequestUI>
  disabled?: boolean
}

export default function ExtraHeadersFormContent(props: ExtraHeadersFormContentProps) {
  const { formikProps, disabled } = props
  return (
    <Layout.Vertical spacing="small">
      <FieldArray
        name="extraHeaders"
        render={({ push, remove }) => {
          return (
            <ExtraHeadersList
              onAdd={push}
              onRemove={remove}
              name="extraHeaders"
              formikProps={formikProps}
              disabled={disabled}
            />
          )
        }}
      />
    </Layout.Vertical>
  )
}
