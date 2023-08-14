import {
  useToaster,
  type ButtonProps,
  Button,
  Dialog,
  Layout,
  Heading,
  FontVariation,
  Container,
  Formik,
  FormikForm,
  FormInput,
  Intent,
  FlexExpander,
  Icon
} from '@harness/uicore'
import { useModalHook } from '@harness/use-modal'
import React from 'react'
import { useMutate } from 'restful-react'
import * as yup from 'yup'
import { useStrings } from 'framework/strings'
import type { OpenapiCreateSecretRequest, TypesSecret } from 'services/code'
import { getErrorMessage } from 'utils/Utils'

interface SecretFormData {
  value: string
  description: string
  name: string
}

const formInitialValues: SecretFormData = {
  value: '',
  description: '',
  name: ''
}

export interface NewSecretModalButtonProps extends Omit<ButtonProps, 'onClick' | 'onSubmit'> {
  space: string
  modalTitle: string
  submitButtonTitle?: string
  cancelButtonTitle?: string
  onSubmit: (data: TypesSecret) => void
}

export const NewSecretModalButton: React.FC<NewSecretModalButtonProps> = ({
  space,
  modalTitle,
  submitButtonTitle,
  cancelButtonTitle,
  onSubmit,
  ...props
}) => {
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const { showError } = useToaster()

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
          uid: formData.name
        }
        const response = await createSecret(payload)
        hideModal()
        onSubmit(response)
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('secrets.failedToCreate'))
      }
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={''}
        style={{ width: 700, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical
          padding={{ left: 'xxlarge' }}
          style={{ height: '100%' }}
          data-testid="add-target-to-flag-modal">
          <Heading level={3} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'xlarge' }}>
            {modalTitle}
          </Heading>

          <Container margin={{ right: 'xxlarge' }}>
            <Formik
              initialValues={formInitialValues}
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
                  padding={{ right: 'xxlarge', top: 'xxxlarge', bottom: 'large' }}
                  style={{ alignItems: 'center' }}>
                  <Button
                    type="submit"
                    text={getString('secrets.createSecret')}
                    intent={Intent.PRIMARY}
                    disabled={loading}
                  />
                  <Button text={cancelButtonTitle || getString('cancel')} minimal onClick={hideModal} />
                  <FlexExpander />
                  {loading && <Icon intent={Intent.PRIMARY} name="spinner" size={16} />}
                </Layout.Horizontal>
              </FormikForm>
            </Formik>
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }

  const [openModal, hideModal] = useModalHook(ModalComponent, [onSubmit])

  return <Button onClick={openModal} {...props} />
}
