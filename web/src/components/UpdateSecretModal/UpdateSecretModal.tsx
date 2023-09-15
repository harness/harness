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
import { getErrorMessage } from 'utils/Utils'

const useUpdateSecretModal = () => {
  const { getString } = useStrings()
  const space = useGetSpaceParam()
  const { showError, showSuccess } = useToaster()
  const [secret, setSecret] = useState<TypesSecret>()
  const postUpdate = useRef<() => Promise<void>>()

  const { mutate: updateSecret, loading } = useMutate<TypesSecret>({
    verb: 'PATCH',
    path: `/api/v1/secrets/${space}/${secret?.uid}/+`
  })

  const handleSubmit = async (formData: SecretFormData) => {
    try {
      const payload: OpenapiUpdateSecretRequest = {
        data: formData.value,
        description: formData.description,
        uid: formData.name
      }
      await updateSecret(payload)
      hideModal()
      showSuccess(
        <StringSubstitute
          str={getString('secrets.secretUpdated')}
          vars={{
            uid: formData.name
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
              initialValues={{ name: secret?.uid || '', description: secret?.description || '', value: '' }}
              formName="addSecret"
              enableReinitialize={true}
              validationSchema={yup.object().shape({
                name: yup.string().trim().required(),
                value: yup.string().trim().required()
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
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
                <FormInput.Text
                  name="value"
                  label={getString('value')}
                  placeholder={getString('secrets.value')}
                  tooltipProps={{
                    dataTooltipId: 'secretDescriptionTextField'
                  }}
                  inputGroup={{ type: 'password' }}
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
                  <Button text={getString('cancel')} minimal variation={ButtonVariation.SECONDARY} onClick={onClose} />
                  <FlexExpander />
                  {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
                </Layout.Horizontal>
              </FormikForm>
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
