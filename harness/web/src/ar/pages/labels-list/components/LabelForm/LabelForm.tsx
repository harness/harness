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

import React, { forwardRef } from 'react'
import * as Yup from 'yup'
import { uniqBy } from 'lodash-es'
import { FieldArray } from 'formik'
import { FontVariation } from '@harnessio/design-system'
import { Checkbox, Container, FormError, Formik, FormInput, Layout, Text } from '@harnessio/uicore'

import { setFormikRef } from '@ar/common/utils'
import type { FormikFowardRef } from '@ar/common/types'
import { useStrings } from '@ar/frameworks/strings'

import type { ColorName } from 'utils/Utils'

import type { LabelFormData } from './types'
import TextInputWithColorPicker from './TextInputWithColorPicker'
import LabelValuesListField from './LabelValuesListField'
import LabelValuesListContent from '../LabelValuesList/LabelValuesListContent'

import css from './LabelForm.module.scss'

interface LabelFormProps {
  isEdit: boolean
  initialValues?: LabelFormData
  onSubmit: (formData: LabelFormData) => void
  disabled?: boolean
}

function LabelForm(props: LabelFormProps, ref: FormikFowardRef) {
  const { isEdit, initialValues, onSubmit, disabled } = props
  const { getString } = useStrings()
  const getInitialValues = (): LabelFormData => {
    if (isEdit && initialValues) {
      return { ...initialValues }
    }
    return {
      key: '',
      type: 'static',
      description: '',
      color: 'blue',
      id: 0,
      labelValues: []
    }
  }

  return (
    <Formik<LabelFormData>
      formName="labelModal"
      initialValues={getInitialValues()}
      validationSchema={Yup.object({
        key: Yup.string()
          .max(50, getString('validationMessages.stringMax', { entity: getString('labelsList.labelForm.name') }))
          .noNewLine(getString('validationMessages.noNewLine', { entity: getString('labelsList.labelForm.name') }))
          .required(
            getString('validationMessages.entityRequired', {
              entity: getString('labelsList.labelForm.name')
            })
          ),
        labelValues: Yup.array()
          .of(
            Yup.object({
              value: Yup.string()
                .max(50, getString('validationMessages.stringMax', { entity: getString('labelsList.labelForm.value') }))
                .noNewLine(
                  getString('validationMessages.noNewLine', { entity: getString('labelsList.labelForm.value') })
                )
                .required(
                  getString('validationMessages.entityRequired', {
                    entity: getString('labelsList.labelForm.value')
                  })
                ),
              color: Yup.string()
            })
          )
          .test(
            'valuesShouldBeUnique',
            getString('validationMessages.uniqueValues', {
              entity: getString('labelsList.labelForm.value')
            }),
            values => {
              if (!values) return true
              return uniqBy(values, 'value').length === values.length
            }
          )
      })}
      onSubmit={onSubmit}>
      {formik => {
        setFormikRef(ref, formik)
        return (
          <Layout.Horizontal height="100%">
            <Layout.Vertical className={css.formContainer} spacing="large">
              <Container className={css.fieldContainer}>
                <Text font={{ variation: FontVariation.BODY, weight: 'bold' }}>
                  {getString('labelsList.labelForm.name')}
                </Text>
                <TextInputWithColorPicker
                  placeholder={getString('enterPlaceholder', {
                    name: getString('labelsList.labelForm.name')
                  })}
                  name="key"
                  colorField="color"
                />
              </Container>
              <Container className={css.fieldContainer}>
                <Text font={{ variation: FontVariation.BODY, weight: 'bold' }}>
                  {getString('labelsList.labelForm.description')}
                </Text>
                <FormInput.Text
                  name="description"
                  placeholder={getString('enterPlaceholder', {
                    name: getString('labelsList.labelForm.description')
                  })}
                />
              </Container>
              <Container className={css.fieldContainer}>
                <Text font={{ variation: FontVariation.BODY, weight: 'bold' }}>
                  {getString('labelsList.labelForm.values')}
                </Text>
                <FieldArray
                  name="labelValues"
                  render={({ push, remove }) => {
                    return (
                      <LabelValuesListField
                        addButtonLabel={getString('labelsList.labelForm.newValue')}
                        onAdd={() => push({ value: '', color: 'blue', id: 0 })}
                        onRemove={remove}
                        name="labelValues"
                        formikProps={formik}
                        disabled={disabled}
                      />
                    )
                  }}
                />
                <FormError
                  name="labelValues"
                  errorMessage={typeof formik.errors.labelValues === 'string' ? formik.errors.labelValues : undefined}
                />
              </Container>
              <Container className={css.fieldContainer}>
                <Checkbox
                  name="dynamic"
                  label={getString('labelsList.labelForm.dynamic')}
                  disabled={disabled}
                  checked={formik.values.type === 'dynamic'}
                  onChange={e => {
                    if (e.currentTarget.checked) {
                      formik.setFieldValue('type', 'dynamic')
                    } else {
                      formik.setFieldValue('type', 'static')
                    }
                  }}
                />
              </Container>
            </Layout.Vertical>
            <Layout.Vertical className={css.previewContainer}>
              <Container>
                <Text font={{ variation: FontVariation.BODY, weight: 'bold' }}>
                  {getString('labelsList.labelForm.preview')}
                </Text>
                <Layout.Vertical spacing="large" className={css.labelsListContainer}>
                  <LabelValuesListContent
                    list={formik.values.labelValues}
                    labelName={formik.values.key}
                    color={formik.values.color as ColorName}
                    allowDynamicValues={formik.values.type === 'dynamic'}
                  />
                </Layout.Vertical>
              </Container>
            </Layout.Vertical>
          </Layout.Horizontal>
        )
      }}
    </Formik>
  )
}

export default forwardRef(LabelForm)
