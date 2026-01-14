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
import { Button, Container, FormInput, Layout } from '@harnessio/uicore'

import { ColorName } from 'utils/Utils'
import { ColorSelectorDropdown } from 'pages/Labels/LabelModal/LabelModal'

import type { LabelFormData } from './types'
import css from './LabelForm.module.scss'

interface TextInputWithColorPickerProps {
  placeholder: string
  name: string
  colorField: string
  onRemove?: () => void
  disabled?: boolean
}

function TextInputWithColorPicker(props: TextInputWithColorPickerProps) {
  const formik = useFormikContext<LabelFormData>()
  const currentColor = get(formik.values, props.colorField, ColorName.Blue)
  return (
    <Container width="100%">
      <Layout.Horizontal
        flex={{
          alignItems: formik.isValid ? 'center' : 'flex-start',
          justifyContent: 'flex-start'
        }}
        spacing="small">
        <ColorSelectorDropdown
          currentColorName={currentColor}
          onClick={(colorName: ColorName) => {
            formik.setFieldValue(props.colorField, colorName)
          }}
          disabled={props.disabled}
        />
        <FormInput.Text
          key={props.name}
          className={css.textInput}
          name={props.name}
          placeholder={props.placeholder}
          tooltipProps={{
            dataTooltipId: 'labels.newLabel'
          }}
          inputGroup={{ autoFocus: true }}
          disabled={props.disabled}
        />
        {props.onRemove && (
          <Button icon="code-close" minimal onClick={props.onRemove} data-testid="delete-label-value" />
        )}
      </Layout.Horizontal>
    </Container>
  )
}

export default TextInputWithColorPicker
