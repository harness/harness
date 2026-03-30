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
import type { ArtifactScanV3 } from '@harnessio/react-har-service-client'
import { Formik, FormikForm } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'
import { setFormikRef } from '@ar/common/utils'
import type { FormikFowardRef } from '@ar/common/types'

import type { ExemptionFormSpec } from './types'
import { DEFAULT_EXEMPTION_DURATION } from './constants'
import CreateExemptionFormContent from './ExemptionFormContent'

import css from './ExemptionForm.module.scss'

interface ExemptionFormProps {
  data: ArtifactScanV3
  onSubmit: (data: ExemptionFormSpec) => void
  isEdit?: boolean
  title?: string
  subTitle?: string
}

function ExemptionForm(props: ExemptionFormProps, formikRef: FormikFowardRef): JSX.Element {
  const { data, onSubmit, isEdit = false, title, subTitle } = props
  const { getString } = useStrings()

  const getInitialValues = () => {
    return {
      registryId: data.registryId,
      packageName: data.packageName,
      versionList: [{ label: data.version, value: data.version }],
      businessJustification: '',
      remediationPlan: '',
      expireAfter: DEFAULT_EXEMPTION_DURATION
    }
  }

  return (
    <Formik<ExemptionFormSpec>
      validationSchema={Yup.object().shape({
        packageName: Yup.string().trim().required(getString('validationMessages.required')),
        expireAfter: Yup.number().required(getString('validationMessages.required')),
        businessJustification: Yup.string().trim().required(getString('validationMessages.required')),
        remediationPlan: Yup.string().trim().required(getString('validationMessages.required'))
      })}
      formName="exemption-form"
      initialValues={getInitialValues()}
      onSubmit={onSubmit}>
      {formik => {
        setFormikRef(formikRef, formik)
        return (
          <FormikForm className={css.formContainer}>
            <CreateExemptionFormContent data={props.data} isEdit={isEdit} title={title} subTitle={subTitle} />
          </FormikForm>
        )
      }}
    </Formik>
  )
}

export default forwardRef(ExemptionForm)
