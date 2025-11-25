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

import React, { forwardRef, useEffect } from 'react'
import * as Yup from 'yup'
import { FieldArray, useFormikContext } from 'formik'
import { Formik, FormikForm } from '@harnessio/uicore'

import { setFormikRef } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings'
import type { FormikFowardRef } from '@ar/common/types'

import PropertiesList from './PropertiesList'
import type { PropertiesFormProps, PropertiesFormValues, PropertySpec } from './types'

interface PropertiesFormContentProps {
  readonly?: boolean
  onChangeDirty: (dirty: boolean) => void
  minItems?: number
  supportQuery?: boolean
}

const DEFAULT_VALUE: PropertySpec = { key: '', value: '', type: 'MANUAL' }

function PropertiesFormContent(props: PropertiesFormContentProps) {
  const { readonly, onChangeDirty, minItems = 0, supportQuery } = props
  const { dirty, values, setFieldValue } = useFormikContext<PropertiesFormValues>()
  const { getString } = useStrings()

  useEffect(() => {
    if (minItems > 0 && values.value.length < minItems) {
      setFieldValue(
        'value',
        Array.from({ length: minItems }, () => DEFAULT_VALUE)
      )
    }
  }, [minItems, values.value.length, setFieldValue])

  useEffect(() => {
    onChangeDirty(dirty)
  }, [dirty, onChangeDirty])

  return (
    <FieldArray
      name="value"
      render={({ push, remove }) => {
        return (
          <PropertiesList
            name="value"
            onRemove={remove}
            onAdd={() => push(DEFAULT_VALUE)}
            disabled={readonly}
            addButtonLabel={getString('addMetadata')}
            supportQuery={supportQuery}
          />
        )
      }}
    />
  )
}

function PropertiesForm(props: PropertiesFormProps, formikRef: FormikFowardRef) {
  const { readonly, value, onChangeDirty, minItems, supportQuery } = props
  const { getString } = useStrings()

  return (
    <Formik<PropertiesFormValues>
      formName="properties-form"
      initialValues={value}
      onSubmit={props.onSubmit}
      validationSchema={Yup.object({
        value: Yup.array().of(
          Yup.lazy((v: PropertySpec) => {
            return Yup.object().shape({
              key: v.value ? Yup.string().required(getString('validationMessages.required')) : Yup.string(),
              value: v.key ? Yup.string().required(getString('validationMessages.required')) : Yup.string(),
              type: Yup.string()
            }) as Yup.Schema<PropertySpec>
          })
        )
      })}>
      {formik => {
        setFormikRef(formikRef, formik)
        return (
          <FormikForm>
            <PropertiesFormContent
              readonly={readonly}
              onChangeDirty={onChangeDirty}
              minItems={minItems}
              supportQuery={supportQuery}
            />
          </FormikForm>
        )
      }}
    </Formik>
  )
}

PropertiesForm.displayName = 'PropertiesForm'

export default forwardRef(PropertiesForm)
