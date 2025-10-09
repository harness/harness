/*
 * Copyright 2023 Harness, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {
  useToaster,
  type ButtonProps,
  Button,
  Dialog,
  Layout,
  Heading,
  Container,
  Formik,
  FormikForm,
  FormInput,
  FlexExpander,
  ButtonVariation
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Intent, FontVariation } from '@harnessio/design-system'
import React from 'react'
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import type { OpenapiCreateSecretRequest, TypesSecret } from 'services/code'
import { getErrorMessage } from 'utils/Utils'
import Config from 'Config'
import css from './NewSecretModalButton.module.scss'

export interface SecretFormData {
  value: string
  description: string
  name: string
  showValue: boolean
}

const formInitialValues: SecretFormData = {
  value: '',
  description: '',
  name: '',
  showValue: false
}

export interface NewSecretModalButtonProps extends Omit<ButtonProps, 'onClick' | 'onSubmit'> {
  space: string
  modalTitle: string
  submitButtonTitle?: string
  cancelButtonTitle?: string
  onSuccess: (secret: TypesSecret) => void
}

export const NewSecretModalButton: React.FC<NewSecretModalButtonProps> = ({
  space,
  modalTitle,
  submitButtonTitle,
  cancelButtonTitle,
  onSuccess,
  ...props
}) => {
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const { showError, showSuccess } = useToaster()

    const { mutate: createSecret, loading } = useMutate<TypesSecret>({
      verb: 'POST',
      path: `/api/v1/secrets`
    })

    const handleSubmit = async (formData: SecretFormData) => {
      try {
        const payload: OpenapiCreateSecretRequest = {
          space_ref: space,
          data: formData.value,
          description: formData.description,
          identifier: formData.name
        }
        const response = await createSecret(payload)
        hideModal()
        showSuccess(getString('secrets.createSuccess'))
        onSuccess(response)
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('secrets.failedToCreate'))
      }
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={
          <Heading level={3} font={{ variation: FontVariation.H3 }}>
            {modalTitle}
          </Heading>
        }
        style={{ width: 700, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical style={{ height: '100%' }} data-testid="add-secret-modal">
          <Container>
            <Formik
              initialValues={formInitialValues}
              formName="addSecret"
              enableReinitialize={true}
              validationSchema={yup.object().shape({
                name: yup
                  .string()
                  .trim()
                  .required()
                  .min(1, getString('validation.nameTooShort'))
                  .max(100, getString('validation.nameTooLong'))
                  .matches(/^[a-zA-Z_][a-zA-Z0-9-_.]*$/, getString('validation.nameLogic')),
                value: yup.string().trim().required()
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              {formik => (
                <FormikForm>
                  <Container>
                    <FormInput.Text
                      name="name"
                      label={getString('name')}
                      placeholder={getString('secrets.enterSecretName')}
                      tooltipProps={{
                        dataTooltipId: 'secretNameTextField'
                      }}
                      inputGroup={{ autoFocus: true }}
                    />
                    <FormInput.TextArea
                      name="value"
                      label={getString('value')}
                      placeholder={getString('secrets.enterSecretValue')}
                      tooltipProps={{
                        dataTooltipId: 'secretDescriptionTextField'
                      }}
                      maxLength={Config.SECRET_LIMIT_IN_BYTES}
                      autoComplete="off"
                      className={formik.values.showValue ? css.showValue : css.hideValue}
                    />
                    <FormInput.CheckBox
                      name="showValue"
                      label={getString('secrets.showValue')}
                      tooltipProps={{
                        dataTooltipId: 'secretDescriptionTextField'
                      }}
                      style={{ display: 'flex' }}
                    />
                    <FormInput.Text
                      name="description"
                      label={getString('description')}
                      placeholder={getString('enterDescription')}
                      tooltipProps={{
                        dataTooltipId: 'secretDescriptionTextField'
                      }}
                      isOptional
                    />
                  </Container>

                  <Layout.Horizontal
                    spacing="small"
                    padding={{ right: 'xxlarge', top: 'xxxlarge' }}
                    style={{ alignItems: 'center' }}>
                    <Button
                      type="submit"
                      text={getString('secrets.createSecret')}
                      variation={ButtonVariation.PRIMARY}
                      disabled={loading}
                    />
                    <Button
                      text={cancelButtonTitle || getString('cancel')}
                      minimal
                      onClick={hideModal}
                      variation={ButtonVariation.SECONDARY}
                    />
                    <FlexExpander />
                    {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
                  </Layout.Horizontal>
                </FormikForm>
              )}
            </Formik>
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }

  const [openModal, hideModal] = useModalHook(ModalComponent, [onSuccess])

  return <Button onClick={openModal} {...props} />
}
