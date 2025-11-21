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
import { FormikContextType, connect } from 'formik'
import { FormGroup, Intent } from '@blueprintjs/core'
import { FormError, errorCheck } from '@harnessio/uicore'
import type { StyledProps } from '@harnessio/design-system'

import MultiTagsInput from '@ar/components/MultiTagsInput/MultiTagsInput'

interface PatternInputProps extends StyledProps {
  label?: string
  name: string
  placeholder?: string
  disabled?: boolean
}

function PatternInput<T>(props: PatternInputProps & { formik: FormikContextType<T> }): JSX.Element {
  const { label, name, placeholder, formik, disabled } = props

  const formValue = get(formik.values, name, []) as string[]

  const hasError = errorCheck(name, formik)
  const intent = hasError ? Intent.DANGER : Intent.NONE
  const helperText = hasError ? <FormError name={name} errorMessage={get(formik?.errors, name)} /> : null

  return (
    <FormGroup label={label} labelFor={name} helperText={helperText} intent={intent}>
      <MultiTagsInput
        selectedItems={formValue}
        hidePopover
        items={[]}
        allowNewTag
        inputProps={{
          name
        }}
        placeholder={placeholder}
        onChange={selectedItems => {
          formik.setFieldValue(name, selectedItems)
        }}
        readonly={disabled}
      />
    </FormGroup>
  )
}

export default connect<PatternInputProps>(PatternInput)
