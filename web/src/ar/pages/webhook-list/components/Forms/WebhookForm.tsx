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

import React, { forwardRef, useMemo } from 'react'
import * as Yup from 'yup'
import { Formik } from '@harnessio/uicore'
import type { Webhook, WebhookRequest } from '@harnessio/react-har-service-client'

import { useAppStore } from '@ar/hooks'
import { GENERIC_URL_REGEX } from '@ar/constants'
import { setFormikRef } from '@ar/common/utils'
import { useStrings } from '@ar/frameworks/strings'
import type { FormikFowardRef } from '@ar/common/types'

import type { WebhookRequestUI } from './types'
import WebhookFormContent from './WebhookFormContent'
import { transformFormValuesToSubmitValues, transformWebhookDataToFormValues } from './utils'

interface CreateWebhookFormProps {
  data?: Webhook
  onSubmit: (values: WebhookRequest) => void
  setDirty?: (dirty: boolean) => void
  readonly?: boolean
  isEdit?: boolean
}

function WebhookForm(props: CreateWebhookFormProps, formikRef: FormikFowardRef<WebhookRequestUI>) {
  const { onSubmit, readonly, isEdit, data, setDirty } = props
  const { getString } = useStrings()
  const { parent, scope } = useAppStore()
  const initialValues: WebhookRequestUI = useMemo(() => {
    if (isEdit && data) {
      return transformWebhookDataToFormValues(data, parent)
    }
    return {
      identifier: '',
      name: '',
      url: '',
      triggerType: 'all',
      enabled: true,
      insecure: false,
      extraHeaders: [{ key: '', value: '' }]
    }
  }, [])

  const handleSubmit = (values: WebhookRequestUI) => {
    const formValues = transformFormValuesToSubmitValues(values, parent, scope)
    onSubmit(formValues)
  }

  return (
    <Formik<WebhookRequestUI>
      formName="webhook-form"
      validationSchema={Yup.object().shape({
        identifier: Yup.string().required(getString('validationMessages.identifierRequired')),
        name: Yup.string().required(getString('validationMessages.nameRequired')),
        url: Yup.string()
          .required(getString('validationMessages.urlRequired'))
          .matches(GENERIC_URL_REGEX, getString('validationMessages.genericURLPattern')),
        triggers: Yup.array().when(['triggerType'], {
          is: (triggerType: WebhookRequestUI['triggerType']) => triggerType === 'custom',
          then: (schema: Yup.StringSchema) => schema.required(getString('validationMessages.required')),
          otherwise: (schema: Yup.StringSchema) => schema.notRequired()
        })
      })}
      onSubmit={handleSubmit}
      initialValues={initialValues}>
      {formik => {
        setDirty?.(formik.dirty)
        setFormikRef(formikRef, formik)
        return <WebhookFormContent formikProps={formik} isEdit={isEdit} readonly={readonly} />
      }}
    </Formik>
  )
}

export default forwardRef(WebhookForm)
