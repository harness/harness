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
import { connect } from 'formik'
import { FormGroup, IFormGroupProps, Intent } from '@blueprintjs/core'
import { Checkbox, FormError, getFormFieldLabel, errorCheck } from '@harnessio/uicore'

import type { FormikContextProps } from '@ar/common/types'

interface CheckboxItem {
  label: string
  value: string
  disabled?: boolean
  tooltipId?: string
}

export interface CheckboxGroupProps extends Omit<IFormGroupProps, 'labelFor'> {
  name: string
  items: CheckboxItem[]
}

function CheckboxGroup(props: CheckboxGroupProps & FormikContextProps<any>) {
  const { formik, name } = props
  const hasError = errorCheck(name, formik)
  const {
    intent = hasError ? Intent.DANGER : Intent.NONE,
    helperText = hasError ? <FormError name={name} errorMessage={get(formik?.errors, name)} /> : null,
    disabled = formik?.disabled,
    items = [],
    label,
    ...rest
  } = props

  const formValue: string[] = get(formik?.values, name, [])

  const handleChange = (val: string, checked: boolean) => {
    const newValue = checked ? [...formValue, val] : formValue.filter(v => v !== val)
    formik?.setFieldValue(name, newValue)
  }

  return (
    <FormGroup
      label={getFormFieldLabel(label, name, props)}
      labelFor={name}
      helperText={helperText}
      intent={intent}
      disabled={disabled}
      {...rest}>
      {items.map(item => {
        return (
          <Checkbox
            key={item.value}
            value={item.value}
            disabled={disabled}
            checked={formValue?.includes(item.value)}
            onChange={e => {
              handleChange(item.value, e.currentTarget.checked)
            }}
            label={item.label}
          />
        )
      })}
    </FormGroup>
  )
}

export default connect(CheckboxGroup)
