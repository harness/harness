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

interface DeleteModalContentProps {
  entity: string
  value: string
  onSubmit: () => void
  onClose: () => void
  content: string
  placeholder: string
  inputLabel: string
  deleteBtnText?: string
}

function DeleteModalContent({
  entity,
  value,
  onSubmit,
  onClose,
  content,
  placeholder,
  inputLabel,
  deleteBtnText
}: DeleteModalContentProps) {
  const { getString } = useStrings()
  return (
    <Formik
      initialValues={{ value: '' }}
      onSubmit={onSubmit}
      validationSchema={object({
        value: string().required(getString('validationMessages.required')).oneOf(
          [value],
          getString('validationMessages.equals', {
            entity,
            value
          })
        )
      })}>
      {formik => (
        <Container>
          <Layout.Vertical spacing="medium">
            <Text>{content}</Text>
            <FormInput.Text label={inputLabel} name="value" placeholder={placeholder} intent={Intent.PRIMARY} />
          </Layout.Vertical>
          <Layout.Horizontal spacing="medium" margin={{ top: 'large' }}>
            <Button
              variation={ButtonVariation.PRIMARY}
              intent="danger"
              text={deleteBtnText || getString('delete')}
              onClick={formik.submitForm}
            />
            <Button variation={ButtonVariation.SECONDARY} text={getString('cancel')} onClick={onClose} />
          </Layout.Horizontal>
        </Container>
      )}
    </Formik>
  )
}

export default DeleteModalContent
