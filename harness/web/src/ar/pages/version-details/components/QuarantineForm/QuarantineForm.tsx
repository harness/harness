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
import { Formik, FormikForm, FormInput } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { setFormikRef } from '@ar/common/utils'
import type { FormikFowardRef } from '@ar/common/types'

import type { QuarantineVersionFormData } from './type'

interface QuarantineVersionFormProps {
  onSubmit: (values: QuarantineVersionFormData) => void
}

function QuarantineVersionForm(
  props: QuarantineVersionFormProps,
  formikRef: FormikFowardRef<QuarantineVersionFormData>
) {
  const { onSubmit } = props
  const { getString } = useStrings()
  return (
    <Formik<QuarantineVersionFormData>
      validationSchema={Yup.object({
        reason: Yup.string().required(
          getString('validationMessages.entityRequired', {
            entity: getString('versionDetails.quarantineVersionModal.reasonField')
          })
        )
      })}
      initialValues={{ reason: '' }}
      onSubmit={onSubmit}
      formName="quarantineVersion">
      {formik => {
        setFormikRef(formikRef, formik)
        return (
          <FormikForm>
            <FormInput.TextArea
              label={getString('versionDetails.quarantineVersionModal.reasonField')}
              name="reason"
              placeholder={getString('enterPlaceholder', {
                name: getString('versionDetails.quarantineVersionModal.reasonField')
              })}
            />
          </FormikForm>
        )
      }}
    </Formik>
  )
}

export default forwardRef(QuarantineVersionForm)
