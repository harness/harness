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
import { Formik } from 'formik'
import { object, string } from 'yup'
import { Intent } from '@blueprintjs/core'
import { Button, ButtonVariation, Container, FormInput, Layout, Text } from '@harnessio/uicore'

import { useStrings } from '@ar/frameworks/strings'

export interface StatusChangeFormValues {
  value: string
}

interface StatusChangeModalProps {
  content: string
  inputLabel: string
  placeholder: string
  submitButtonText: string
  submitButtonIntent: Intent
  cancelButtonText: string
  cancelButtonIntent?: Intent
  onSubmit: (values: StatusChangeFormValues) => void
  onClose: () => void
}

function StatusChangeModal(props: StatusChangeModalProps) {
  const { getString } = useStrings()

  return (
    <Formik<StatusChangeFormValues>
      initialValues={{ value: '' }}
      onSubmit={props.onSubmit}
      validationSchema={object({
        value: string().required(getString('validationMessages.required'))
      })}>
      {formik => (
        <Container>
          <Layout.Vertical spacing="small">
            <Text margin={{ bottom: 'medium' }}>{props.content}</Text>
            <FormInput.TextArea
              label={props.inputLabel}
              textArea={{ style: { minHeight: 130 } }}
              name="value"
              placeholder={props.placeholder}
              intent={Intent.PRIMARY}
            />
          </Layout.Vertical>
          <Layout.Horizontal spacing="medium" margin={{ top: 'large' }}>
            <Button
              variation={ButtonVariation.PRIMARY}
              intent={props.submitButtonIntent}
              text={props.submitButtonText}
              onClick={formik.submitForm}
            />
            <Button
              variation={ButtonVariation.SECONDARY}
              text={props.cancelButtonText}
              intent={props.cancelButtonIntent}
              onClick={props.onClose}
            />
          </Layout.Horizontal>
        </Container>
      )}
    </Formik>
  )
}

export default StatusChangeModal
