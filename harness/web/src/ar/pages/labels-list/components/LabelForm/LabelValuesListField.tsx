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
import { Button, Layout } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import TextInputWithColorPicker from './TextInputWithColorPicker'
import type { LabelFormData } from './types'

import css from './LabelForm.module.scss'

interface LabelValuesListFieldProps {
  name: string
  formikProps: FormikProps<LabelFormData>
  onAdd: () => void
  onRemove: (idx: number) => void
  addButtonLabel: string
  disabled?: boolean
}
export default function LabelValuesListField(props: LabelValuesListFieldProps): JSX.Element {
  const { name, formikProps, disabled, onAdd, addButtonLabel, onRemove } = props
  const { getString } = useStrings()
  const values = formikProps.values.labelValues || []

  return (
    <Layout.Vertical spacing="medium" flex={{ alignItems: 'flex-start' }}>
      {values?.map((each, index) => (
        <TextInputWithColorPicker
          key={`${index}-${each.value}`}
          name={`${name}[${index}].value`}
          placeholder={getString('enterPlaceholder', { name: getString('labelsList.labelForm.value') })}
          colorField={`${name}[${index}].color`}
          onRemove={() => onRemove(index)}
          disabled={disabled}
        />
      ))}
      <Button className={css.addBtn} icon="plus" minimal intent="primary" onClick={onAdd} text={addButtonLabel} />
    </Layout.Vertical>
  )
}
