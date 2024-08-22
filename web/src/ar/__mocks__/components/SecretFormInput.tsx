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
import { FormInput } from '@harnessio/uicore'
import type { Scope } from '@ar/MFEAppTypes'

interface SecretFormInputProps<T> {
  name: string
  scope: Scope
  spaceIdFieldName: string
  label: React.ReactNode
  disabled?: boolean
  placeholder?: string
  formik?: FormikProps<T>
}

export default function SecretFormInput<T>(props: SecretFormInputProps<T>) {
  const { disabled, name, label, placeholder } = props
  return (
    <FormInput.Text
      name={name}
      label={label}
      placeholder={placeholder}
      disabled={disabled}
      inputGroup={{
        type: 'password'
      }}
    />
  )
}
