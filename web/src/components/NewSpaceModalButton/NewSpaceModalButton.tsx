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

import React, { useState } from 'react'
import { Dialog, Intent, PopoverPosition, Classes } from '@blueprintjs/core'
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
  Text,
  SplitButtonOption
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import { compact, get } from 'lodash-es'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import { getErrorMessage, permissionProps, REGEX_VALID_REPO_NAME } from 'utils/Utils'
import type { TypesSpace, OpenapiCreateSpaceRequest } from 'services/code'
import { useAppContext } from 'AppContext'
import {
  ImportSpaceFormData,
  SpaceCreationType,
  GitProviders,
  getProviderTypeMapping,
  ConvertPipelineLabel
} from 'utils/GitUtils'
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

    const handleSubmit = async (formData: SpaceFormData) => {
      try {
        const payload: OpenapiCreateSpaceRequestExtended = {
          description: get(formData, 'description', '').trim(),
          is_public: get(formData, 'isPublic') === RepoVisibility.PUBLIC,
          uid: get(formData, 'name', '').trim(),
          parent_id: standalone ? Number(space) : 0 // TODO: Backend needs to fix parentID: accept string or number
        }
        await createSpace(payload)
        hideModal()
        handleNavigation?.(formData.name.trim())
        onRefetch()
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('failedToCreateSpace'))
      }
    }

    const handleImportSubmit = async (formData: ImportSpaceFormData) => {
      const type = getProviderTypeMapping(formData.gitProvider)

      const provider = {
        type,
        username: formData.username,
        password: formData.password,
        host: ''
      }

      if (
        ![GitProviders.GITHUB, GitProviders.GITLAB, GitProviders.BITBUCKET, GitProviders.AZURE].includes(
          formData.gitProvider
        )
      ) {
        provider.host = formData.host
      }

      try {
        const importPayload = {
          description: (formData.description || '').trim(),
          uid: formData.name.trim(),
          provider,
          provider_space: compact([
            formData.organization,
            formData.gitProvider === GitProviders.AZURE ? formData.project : ''
          ]).join('/'),
          pipelines:
            standalone && formData.importPipelineLabel ? ConvertPipelineLabel.CONVERT : ConvertPipelineLabel.IGNORE
        }
        const response = await importSpace(importPayload)
        hideModal()
        onSubmit(response)
        onRefetch()
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
      desc: getString('importSpace.createASpace')
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
          {spaceCreateOptions[0].title}
        </Text>
      }
      variation={ButtonVariation.PRIMARY}
      popoverProps={{
        interactionKind: 'click',
        usePortal: true,
        captureDismiss: true,
        popoverClassName: fromSpace ? css.popoverSpace : css.popoverSplit,
        position: PopoverPosition.BOTTOM_RIGHT
      }}
      icon={'plus'}
      {...permissionProps(permResult, standalone)}
      onClick={() => {
        setSpaceOption(spaceCreateOptions[0])
        setTimeout(() => openModal(), 0)
      }}>
      <Container className={Classes.POPOVER_DISMISS_OVERRIDE}>
        <SplitButtonOption
          className={css.menuItem}
          onClick={() => {
            setSpaceOption(spaceCreateOptions[1])
            setTimeout(() => openModal(), 0)
          }}
          text={<Text font={{ variation: FontVariation.BODY2 }}>{getString('importSpace.title')}</Text>}
        />
      </Container>
    </SplitButton>
  )
}

interface SpaceCreationOption {
  type: SpaceCreationType
  title: string
  desc: string
}
