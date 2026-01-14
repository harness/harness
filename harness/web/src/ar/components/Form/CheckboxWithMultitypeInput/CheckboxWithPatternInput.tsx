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
import { FormikContextType, connect } from 'formik'
import { Checkbox, Layout } from '@harnessio/uicore'

import PatternInput from '../PatternInput/PatternInput'

interface CheckboxWithPatternInputProps {
  name: string
  label?: string
  labelElement?: React.ReactNode
  placeholder?: string
  className?: string
  disabled?: boolean
}

function CheckboxWithPatternInput<T>(
  props: CheckboxWithPatternInputProps & { formik: FormikContextType<T> }
): JSX.Element {
  const { label, name, placeholder, labelElement, className, formik, disabled } = props
  const [checked, setChecked] = useState(getInitialState())

  function getInitialState(): boolean {
    return !!get(formik.values, name, '')
  }

  return (
    <Layout.Vertical className={className} spacing="small">
      <Checkbox
        labelElement={labelElement}
        label={label}
        checked={checked}
        onChange={evt => {
          const isChecked = evt.currentTarget.checked
          setChecked(isChecked)
          if (!isChecked) {
            formik.setFieldValue(name, [])
          }
        }}
      />
      {checked && <PatternInput name={name} placeholder={placeholder} disabled={disabled} />}
    </Layout.Vertical>
  )
}

export default connect<CheckboxWithPatternInputProps>(CheckboxWithPatternInput)
