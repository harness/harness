import React, { useState } from 'react'
import { Dialog, Intent, PopoverPosition, Menu } from '@blueprintjs/core'
import * as yup from 'yup'
import {
  Button,
  ButtonProps,
  Container,
  Layout,
  FlexExpander,
  Formik,
  FormikForm,
  Heading,
  useToaster,
  FormInput,
  ButtonVariation,
  SplitButton,
  Text
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { getErrorMessage, permissionProps, REGEX_VALID_REPO_NAME } from 'utils/Utils'
import type { TypesSpace, OpenapiCreateSpaceRequest } from 'services/code'
import { useAppContext } from 'AppContext'
import { ImportSpaceFormData, SpaceCreationType } from 'utils/GitUtils'
import ImportSpaceForm from './ImportSpaceForm/ImportSpaceForm'
import css from './NewSpaceModalButton.module.scss'

enum RepoVisibility {
  PUBLIC = 'public',
  PRIVATE = 'private'
}

interface SpaceFormData {
  name: string
  description: string
  license: string
  defaultBranch: string
  gitignore: string
  addReadme: boolean
  isPublic: RepoVisibility
}

const formInitialValues: SpaceFormData = {
  name: '',
  description: '',
  license: '',
  defaultBranch: 'main',
  gitignore: '',
  addReadme: false,
  isPublic: RepoVisibility.PRIVATE
}

export interface NewSpaceModalButtonProps extends Omit<ButtonProps, 'onClick' | 'onSubmit'> {
  space: string
  modalTitle: string
  submitButtonTitle?: string
  cancelButtonTitle?: string
  onRefetch: () => void
  handleNavigation?: (value: string) => void
  onSubmit: (data: TypesSpace) => void
  fromSpace?: boolean
}
export interface OpenapiCreateSpaceRequestExtended extends OpenapiCreateSpaceRequest {
  parent_id?: number
}

export const NewSpaceModalButton: React.FC<NewSpaceModalButtonProps> = ({
  space,
  modalTitle,
  submitButtonTitle,
  cancelButtonTitle,
  onRefetch,
  handleNavigation,
  onSubmit,
  fromSpace = false,
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
    const { mutate: importSpace, loading: submitImportLoading } = useMutate<TypesSpace>({
      verb: 'POST',
      path: `/api/v1/spaces/import`
    })

    const loading = submitLoading || submitImportLoading

    const handleSubmit = (formData: SpaceFormData) => {
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
            handleNavigation?.(formData.name)
            onRefetch()
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, getString('failedToCreateSpace'))
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('failedToCreateSpace'))
      }
    }

    const handleImportSubmit = async (formData: ImportSpaceFormData) => {
      try {
        const importPayload = {
          description: formData.description || '',
          uid: formData.name,
          provider: {
            type: formData.gitProvider.toLowerCase(),
            username: formData.username,
            password: formData.password
          },
          provider_space: formData.organization
        }
        await importSpace(importPayload)
          .then(response => {
            hideModal()
            onSubmit(response)
            onRefetch()
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, getString('failedToImportSpace'))
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('failedToImportSpace'))
      }
    }

    return (
      <Dialog
        isOpen
        onClose={hideModal}
        enforceFocus={false}
        title={''}
        style={{
          width: spaceOption.type === SpaceCreationType.IMPORT ? 610 : 700,
          maxHeight: '95vh',
          overflow: 'auto'
        }}>
        <Layout.Vertical
          padding={{ left: 'xxlarge' }}
          style={{ height: '100%' }}
          data-testid="add-target-to-flag-modal">
          <Heading level={3} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'large' }}>
            {spaceOption.type === SpaceCreationType.IMPORT ? getString('importSpace.title') : modalTitle}
          </Heading>

          <Container margin={{ right: 'xxlarge' }}>
            {spaceOption.type === SpaceCreationType.IMPORT ? (
              <ImportSpaceForm hideModal={hideModal} handleSubmit={handleImportSubmit} loading={false} />
            ) : (
              <Formik
                initialValues={formInitialValues}
                formName="editVariations"
                enableReinitialize={true}
                validationSchema={yup.object().shape({
                  name: yup.string().trim().required().matches(REGEX_VALID_REPO_NAME, getString('validation.nameLogic'))
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

                    {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
                  </Layout.Horizontal>
                </FormikForm>
              </Formik>
            )}
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }

  const { getString } = useStrings()

  const spaceCreateOptions: SpaceCreationOption[] = [
    {
      type: SpaceCreationType.CREATE,
      title: getString('newSpace'),
      desc: getString('importSpace.createNewSpace')
    },
    {
      type: SpaceCreationType.IMPORT,
      title: getString('importSpace.title'),
      desc: getString('importSpace.title')
    }
  ]
  const [spaceOption, setSpaceOption] = useState<SpaceCreationOption>(spaceCreateOptions[0])

  const [openModal, hideModal] = useModalHook(ModalComponent, [onSubmit, spaceOption])
  const { standalone } = useAppContext()
  const { hooks } = useAppContext()
  const permResult = hooks?.usePermissionTranslate?.(
    {
      resource: {
        resourceType: 'CODE_REPOSITORY'
      },
      permissions: ['code_repo_push']
    },
    [space]
  )
  return (
    <SplitButton
      {...props}
      loading={false}
      text={
        <Text color={Color.WHITE} font={{ variation: FontVariation.BODY2_SEMI, weight: 'bold' }}>
          {spaceOption.title}
        </Text>
      }
      variation={ButtonVariation.PRIMARY}
      popoverProps={{
        interactionKind: 'click',
        usePortal: true,
        popoverClassName: fromSpace ? css.popoverSpace : css.popoverSplit,
        position: PopoverPosition.BOTTOM_RIGHT,
        transitionDuration: 1000
      }}
      icon={'plus'}
      {...permissionProps(permResult, standalone)}
      onClick={() => {
        openModal()
      }}>
      {spaceCreateOptions.map(option => {
        return (
          <Container key={`import_space_container_${option.type}`}>
            <Menu.Item
              key={`import_space_${option.type}`}
              className={css.menuItem}
              text={<Text font={{ variation: FontVariation.BODY2 }}>{option.desc}</Text>}
              onClick={event => {
                event.stopPropagation()
                event.preventDefault()
                setSpaceOption(option)
              }}
            />
          </Container>
        )
      })}
    </SplitButton>
  )
}

interface SpaceCreationOption {
  type: SpaceCreationType
  title: string
  desc: string
}
