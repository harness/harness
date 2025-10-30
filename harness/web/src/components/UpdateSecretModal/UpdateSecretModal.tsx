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

import React, { useRef, useState } from 'react'
import * as yup from 'yup'
import { useMutate } from 'restful-react'
import { FontVariation, Intent } from '@harnessio/design-system'
import {
  Button,
  Dialog,
  Layout,
  Heading,
  Container,
  Formik,
  FormikForm,
  FormInput,
  FlexExpander,
  useToaster,
  StringSubstitute,
  ButtonVariation
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { useStrings } from 'framework/strings'
import { useModalHook } from 'hooks/useModalHook'
import { useGetSpaceParam } from 'hooks/useGetSpaceParam'
import type { OpenapiUpdateSecretRequest, TypesSecret } from 'services/code'
import type { SecretFormData } from 'components/NewSecretModalButton/NewSecretModalButton'
import { getErrorMessage, truncateString } from 'utils/Utils'
import Config from 'Config'
import css from './UpdateSecretModal.module.scss'

const useUpdateSecretModal = () => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { showError, showSuccess } = useToaster()
  const [secret, setSecret] = useState<TypesSecret>()
  const postUpdate = useRef<() => Promise<void>>()

  const { mutate: updateSecret, loading } = useMutate<TypesSecret>({
    verb: 'PATCH',
    path: `/api/v1/secrets/${space}/${secret?.identifier}/+`
  })

  const handleSubmit = async (formData: SecretFormData) => {
    try {
      const payload: OpenapiUpdateSecretRequest = {
        data: formData.value,
        description: formData.description,
        identifier: formData.name
      }
      await updateSecret(payload)
      hideModal()
      showSuccess(
        <StringSubstitute
          str={getString('secrets.secretUpdated')}
          vars={{
            uid: truncateString(formData.name, 20)
          }}
        />
      )
      postUpdate.current?.()
    } catch (exception) {
      showError(getErrorMessage(exception), 0, getString('secrets.failedToUpdateSecret'))
    }
  }

  const [openModal, hideModal] = useModalHook(() => {
    const onClose = () => {
      hideModal()
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={
          <Heading level={3} font={{ variation: FontVariation.H3 }}>
            {getString('secrets.updateSecret')}
          </Heading>
        }
        style={{ width: 700, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical style={{ height: '100%' }} data-testid="add-secret-modal">
          <Container>
            <Formik
              initialValues={{
                name: secret?.identifier || '',
                description: secret?.description || '',
                value: '',
                showValue: false
              }}
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

                  <Layout.Horizontal
                    spacing="small"
                    padding={{ right: 'xxlarge', top: 'xxxlarge' }}
                    style={{ alignItems: 'center' }}>
                    <Button
                      type="submit"
                      text={getString('secrets.updateSecret')}
                      variation={ButtonVariation.PRIMARY}
                      disabled={loading}
                    />
                    <Button
                      text={getString('cancel')}
                      minimal
                      variation={ButtonVariation.SECONDARY}
                      onClick={onClose}
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
  }, [secret])

  return {
    openModal: ({
      secretToUpdate,
      openSecretUpdate
    }: {
      secretToUpdate: TypesSecret
      openSecretUpdate: () => Promise<void>
    }) => {
      setSecret(secretToUpdate)
      postUpdate.current = openSecretUpdate
      openModal()
    }
  }
}

export default useUpdateSecretModal
