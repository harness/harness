import React from 'react'
import { Dialog, Intent } from '@blueprintjs/core'
import * as yup from 'yup'
import {
  Button,
  ButtonProps,
  Container,
  Layout,
  FlexExpander,
  Icon,
  Formik,
  FormikForm,
  Heading,
  useToaster,
  FormInput
} from '@harness/uicore'
import { FontVariation } from '@harness/design-system'
import { useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { useModalHook } from '@harness/use-modal'
import { useStrings } from 'framework/strings'
import { getErrorMessage, REGEX_VALID_REPO_NAME } from 'utils/Utils'
import type { TypesSpace, OpenapiCreateSpaceRequest } from 'services/code'
import { useAppContext } from 'AppContext'

enum RepoVisibility {
  PUBLIC = 'public',
  PRIVATE = 'private'
}

interface RepoFormData {
  name: string
  description: string
  license: string
  defaultBranch: string
  gitignore: string
  addReadme: boolean
  isPublic: RepoVisibility
}

const formInitialValues: RepoFormData = {
  name: '',
  description: '',
  license: '',
  defaultBranch: 'main',
  gitignore: '',
  addReadme: false,
  isPublic: RepoVisibility.PRIVATE
}

export interface NewSpaceModalButtonProps extends Omit<ButtonProps, 'onClick'> {
  space: string
  modalTitle: string
  submitButtonTitle?: string
  cancelButtonTitle?: string
  //   onSubmit: (data: TypesRepository) => void
}
export interface OpenapiCreateSpaceRequestExtended extends OpenapiCreateSpaceRequest {
  parent_id?: number
}

export const NewSpaceModalButton: React.FC<NewSpaceModalButtonProps> = ({
  space,
  modalTitle,
  submitButtonTitle,
  cancelButtonTitle,
  //   onSubmit,
  ...props
}) => {
  const ModalComponent: React.FC = () => {
    const { standalone } = useAppContext()
    const { getString } = useStrings()
    const { showError } = useToaster()

    const { mutate: createSpace, loading: submitLoading } = useMutate<TypesSpace>({
      verb: 'POST',
      path: `/api/v1/spaces`
    })

    const loading = submitLoading

    const handleSubmit = (formData: RepoFormData) => {
      try {
        const payload: OpenapiCreateSpaceRequestExtended = {
          description: get(formData, 'description', '').trim(),
          is_public: get(formData, 'isPublic') === RepoVisibility.PUBLIC,
          uid: get(formData, 'name', '').trim(),
          parent_id: standalone ? Number(space) : 0 // TODO: Backend needs to fix parentID: accept string or number
        }
        createSpace(payload)
          .then(() => {
            hideModal()
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, getString('failedToCreateSpace'))
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('failedToCreateSpace'))
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
              formName="editVariations"
              enableReinitialize={true}
              validationSchema={yup.object().shape({
                name: yup
                  .string()
                  .trim()
                  .required()
                  .matches(REGEX_VALID_REPO_NAME, getString('validation.spaceNamePatternIsNotValid'))
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              <FormikForm>
                <FormInput.Text
                  name="name"
                  label={getString('name')}
                  placeholder={getString('enterName')}
                  tooltipProps={{
                    dataTooltipId: 'spaceNameTextField'
                  }}
                  inputGroup={{ autoFocus: true }}
                />
                <FormInput.Text
                  name="description"
                  label={getString('description')}
                  placeholder={getString('enterDescription')}
                  tooltipProps={{
                    dataTooltipId: 'spaceDescriptionTextField'
                  }}
                />

                <Layout.Horizontal
                  spacing="small"
                  padding={{ right: 'xxlarge', top: 'xxxlarge', bottom: 'large' }}
                  style={{ alignItems: 'center' }}>
                  <Button type="submit" text={getString('createSpace')} intent={Intent.PRIMARY} disabled={loading} />
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

  const [openModal, hideModal] = useModalHook(ModalComponent)

  return <Button onClick={openModal} {...props} />
}
