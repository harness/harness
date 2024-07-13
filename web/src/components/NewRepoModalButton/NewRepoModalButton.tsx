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

import React, { useEffect, useMemo, useState } from 'react'
import {
  Icon as BPIcon,
  Classes,
  Dialog,
  Intent,
  Menu,
  MenuDivider,
  MenuItem,
  PopoverPosition
} from '@blueprintjs/core'
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
  Text,
  ButtonVariation,
  ButtonSize,
  TextInput,
  SplitButton
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import { Color, FontVariation } from '@harnessio/design-system'
import { useGet, useMutate } from 'restful-react'
import { Render } from 'react-jsx-match'
import { compact, get } from 'lodash-es'
import { useModalHook } from 'hooks/useModalHook'
import { useStrings } from 'framework/strings'
import {
  DEFAULT_BRANCH_NAME,
  getErrorMessage,
  permissionProps,
  REGEX_VALID_REPO_NAME,
  SUGGESTED_BRANCH_NAMES
} from 'utils/Utils'
import {
  ConvertPipelineLabel,
  GitProviders,
  ImportFormData,
  ImportSpaceFormData,
  RepoCreationType,
  RepoFormData,
  RepoVisibility,
  isGitBranchNameValid,
  getProviderTypeMapping
} from 'utils/GitUtils'
import type {
  SpaceSpaceOutput,
  RepoRepositoryOutput,
  SpaceImportRepositoriesOutput,
  OpenapiCreateRepositoryRequest
} from 'services/code'
import { useAppContext } from 'AppContext'
import type { TypesRepository } from 'cde-gitness/services'
import ImportForm from './ImportForm/ImportForm'
import ImportReposForm from './ImportReposForm/ImportReposForm'
import Private from '../../icons/private.svg?url'
import css from './NewRepoModalButton.module.scss'

const formInitialValues: RepoFormData = {
  name: '',
  description: '',
  license: '',
  defaultBranch: 'main',
  gitignore: '',
  addReadme: false,
  isPublic: RepoVisibility.PRIVATE
}

export interface NewRepoModalButtonProps extends Omit<ButtonProps, 'onClick' | 'onSubmit'> {
  space: string
  modalTitle: string
  submitButtonTitle?: string
  cancelButtonTitle?: string
  onSubmit: (data: TypesRepository & RepoRepositoryOutput & SpaceImportRepositoriesOutput) => void
  repoCreationType?: RepoCreationType
  customRenderer?: (onChange: (event: any) => void) => React.ReactNode
}

export const NewRepoModalButton: React.FC<NewRepoModalButtonProps> = ({
  space,
  modalTitle,
  submitButtonTitle,
  cancelButtonTitle,
  onSubmit,
  ...props
}) => {
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const [branchName, setBranchName] = useState(DEFAULT_BRANCH_NAME)
    const [enablePublicRepo, setEnablePublicRepo] = useState(false)
    const { showError } = useToaster()

    const { mutate: createRepo, loading: submitLoading } = useMutate<RepoRepositoryOutput>({
      verb: 'POST',
      path: `/api/v1/repos`,
      queryParams: standalone
        ? undefined
        : {
            space_path: space
          }
    })
    const { mutate: importRepo, loading: importRepoLoading } = useMutate<RepoRepositoryOutput>({
      verb: 'POST',
      path: `/api/v1/repos/import`,
      queryParams: standalone
        ? undefined
        : {
            space_path: space
          }
    })
    const { mutate: importMultipleRepositories, loading: submitImportLoading } = useMutate<SpaceSpaceOutput>({
      verb: 'POST',
      path: `/api/v1/spaces/${space}/+/import`
    })
    const {
      data: gitignores,
      loading: gitIgnoreLoading,
      error: gitIgnoreError
    } = useGet({ path: '/api/v1/resources/gitignore' })
    const {
      data: licences,
      loading: licenseLoading,
      error: licenseError
    } = useGet({ path: '/api/v1/resources/license' })
    const {
      data: systemConfig,
      loading: systemConfigLoading,
      error: systemConfigError
    } = useGet({ path: 'api/v1/system/config' })

    const loading =
      submitLoading ||
      gitIgnoreLoading ||
      licenseLoading ||
      importRepoLoading ||
      submitImportLoading ||
      systemConfigLoading

    useEffect(() => {
      if (gitIgnoreError || licenseError || systemConfigError) {
        showError(getErrorMessage(gitIgnoreError || licenseError || systemConfigError), 0)
      }
    }, [gitIgnoreError, licenseError, systemConfigError, showError])

    useEffect(() => {
      if (systemConfig) {
        setEnablePublicRepo(systemConfig.public_resource_creation_enabled)
      }
    }, [systemConfig])
    const handleSubmit = (formData: RepoFormData) => {
      try {
        const payload: OpenapiCreateRepositoryRequest = {
          default_branch: branchName || get(formData, 'defaultBranch', DEFAULT_BRANCH_NAME),
          description: get(formData, 'description', '').trim(),
          git_ignore: get(formData, 'gitignore', 'none'),
          is_public: get(formData, 'isPublic') === RepoVisibility.PUBLIC,
          license: get(formData, 'license', 'none'),
          identifier: get(formData, 'name', '').trim(),
          readme: get(formData, 'addReadme', false),
          parent_ref: space
        }
        createRepo(payload)
          .then(response => {
            hideModal()
            onSubmit(response)
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, getString('failedToCreateRepo'))
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('failedToCreateRepo'))
      }
    }

    const handleImportSubmit = (formData: ImportFormData) => {
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
        provider.host = formData.hostUrl
      }

      const importPayload = {
        description: formData.description || '',
        parent_ref: space,
        identifier: formData.name,
        provider,
        provider_repo: compact([
          formData.org,
          formData.gitProvider === GitProviders.AZURE ? formData.project : '',
          formData.repo
        ])
          .join('/')
          .replace(/\.git$/, ''),
        pipelines:
          standalone && formData.importPipelineLabel ? ConvertPipelineLabel.CONVERT : ConvertPipelineLabel.IGNORE
      }
      importRepo(importPayload)
        .then(response => {
          hideModal()
          onSubmit(response)
        })
        .catch(_error => {
          showError(getErrorMessage(_error), 0, getString('importRepo.failedToImportRepo'))
        })
    }

    const handleMultiRepoImportSubmit = async (formData: ImportSpaceFormData) => {
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
          parent_ref: space,
          identifier: formData.name.trim(),
          provider,
          provider_space: compact([
            formData.organization,
            formData.gitProvider === GitProviders.AZURE ? formData.project : ''
          ]).join('/'),
          pipelines:
            standalone && formData.importPipelineLabel ? ConvertPipelineLabel.CONVERT : ConvertPipelineLabel.IGNORE
        }
        const response = await importMultipleRepositories(importPayload)
        hideModal()
        onSubmit(response)
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('failedToImportSpace'))
      }
    }
    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={''}
        style={{ width: 610, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical
          padding={{ left: 'xxlarge' }}
          style={{ height: '100%' }}
          data-testid="add-target-to-flag-modal">
          <Heading level={3} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'xlarge' }}>
            {repoOption.type === RepoCreationType.IMPORT
              ? getString('importRepo.title')
              : repoOption.type === RepoCreationType.IMPORT_MULTIPLE
              ? getString('importRepos.title')
              : modalTitle}
          </Heading>

          <Container margin={{ right: 'xxlarge' }}>
            {repoOption.type === RepoCreationType.IMPORT ? (
              <ImportForm hideModal={hideModal} handleSubmit={handleImportSubmit} loading={false} />
            ) : repoOption.type === RepoCreationType.IMPORT_MULTIPLE ? (
              <ImportReposForm
                hideModal={hideModal}
                handleSubmit={handleMultiRepoImportSubmit}
                loading={false}
                spaceRef={space}
              />
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
                    placeholder={getString('enterRepoName')}
                    tooltipProps={{
                      dataTooltipId: 'repositoryNameTextField'
                    }}
                    inputGroup={{ autoFocus: true }}
                  />
                  <FormInput.Text
                    name="description"
                    label={getString('description')}
                    placeholder={getString('enterDescription')}
                    tooltipProps={{
                      dataTooltipId: 'repositoryDescriptionTextField'
                    }}
                  />
                  <Container margin={{ top: 'medium', bottom: 'medium' }}>
                    <Text>
                      {getString('createRepoModal.branchLabel')}
                      <strong>
                        <Button
                          text={branchName}
                          icon="git-new-branch"
                          rightIcon="chevron-down"
                          variation={ButtonVariation.TERTIARY}
                          size={ButtonSize.SMALL}
                          iconProps={{ size: 14 }}
                          tooltip={<BranchName currentBranchName={branchName} onSelect={name => setBranchName(name)} />}
                          tooltipProps={{ interactionKind: 'click' }}
                        />
                      </strong>
                      {getString('createRepoModal.branch')}
                    </Text>
                  </Container>
                  <Render when={enablePublicRepo}>
                    <hr className={css.dividerContainer} />
                    <Container>
                      <FormInput.RadioGroup
                        name="isPublic"
                        label=""
                        items={[
                          {
                            label: (
                              <Container>
                                <Layout.Horizontal>
                                  <Icon name="git-clone-step" size={20} margin={{ right: 'medium' }} />
                                  <Container>
                                    <Layout.Vertical spacing="xsmall">
                                      <Text>{getString('public')}</Text>
                                      <Text font={{ variation: FontVariation.TINY }}>
                                        {getString('createRepoModal.publicLabel')}
                                      </Text>
                                    </Layout.Vertical>
                                  </Container>
                                </Layout.Horizontal>
                              </Container>
                            ),
                            value: RepoVisibility.PUBLIC
                          },
                          {
                            label: (
                              <Container>
                                <Layout.Horizontal>
                                  <Container margin={{ right: 'medium' }}>
                                    <img width={20} height={20} src={Private} />
                                  </Container>
                                  {/* <Icon name="git-clone-step" size={20} margin={{ right: 'medium' }} /> */}
                                  <Container margin={{ left: 'small' }}>
                                    <Layout.Vertical spacing="xsmall">
                                      <Text>{getString('private')}</Text>
                                      <Text font={{ variation: FontVariation.TINY }}>
                                        {getString('createRepoModal.privateLabel')}
                                      </Text>
                                    </Layout.Vertical>
                                  </Container>
                                </Layout.Horizontal>
                              </Container>
                            ),
                            value: RepoVisibility.PRIVATE
                          }
                        ]}
                      />
                    </Container>
                  </Render>
                  <hr className={css.dividerContainer} />

                  <FormInput.Select
                    name="license"
                    label={getString('addLicense')}
                    placeholder={getString('none')}
                    items={licences || []}
                    usePortal
                  />

                  <FormInput.Select
                    name="gitignore"
                    label={getString('addGitIgnore')}
                    placeholder={getString('none')}
                    items={(gitignores || []).map((entry: string) => ({ label: entry, value: entry }))}
                    usePortal
                  />

                  <FormInput.CheckBox
                    name="addReadme"
                    label={getString('addReadMe')}
                    tooltipProps={{
                      dataTooltipId: 'addReadMe'
                    }}
                  />
                  <hr className={css.dividerContainer} />
                  <Layout.Horizontal
                    spacing="small"
                    padding={{ right: 'xxlarge', bottom: 'large' }}
                    style={{ alignItems: 'center' }}>
                    <Button type="submit" text={getString('createRepo')} intent={Intent.PRIMARY} disabled={loading} />
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

  const repoCreateOptions: RepoCreationOption[] = [
    {
      type: RepoCreationType.CREATE,
      title: getString('newRepo'),
      desc: getString('createARepo')
    },
    {
      type: RepoCreationType.IMPORT,
      title: getString('importGitRepo'),
      desc: getString('importGitRepo')
    },
    {
      type: RepoCreationType.IMPORT_MULTIPLE,
      title: getString('importGitRepos'),
      desc: getString('importGitRepos')
    }
  ]
  const [repoOption, setRepoOption] = useState<RepoCreationOption>(repoCreateOptions[0])

  const [openModal, hideModal] = useModalHook(ModalComponent, [onSubmit, repoOption])
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

  return props?.repoCreationType ? (
    <>
      {props?.customRenderer?.(e => {
        e.preventDefault()
        e.stopPropagation()
        setRepoOption(repoCreateOptions.find(option => option.type === props?.repoCreationType) || repoCreateOptions[0])
        setTimeout(() => openModal(), 0)
      })}
    </>
  ) : (
    <SplitButton
      {...props}
      loading={false}
      text={
        <Text color={Color.WHITE} font={{ variation: FontVariation.BODY2_SEMI, weight: 'bold' }}>
          {repoCreateOptions[0].title}
        </Text>
      }
      variation={ButtonVariation.PRIMARY}
      popoverProps={{
        interactionKind: 'click',
        usePortal: true,
        popoverClassName: css.popover,
        position: PopoverPosition.BOTTOM_RIGHT
      }}
      icon={'plus'}
      {...permissionProps(permResult, standalone)}
      onClick={() => {
        setRepoOption(repoCreateOptions[0])
        setTimeout(() => openModal(), 0)
      }}>
      {[repoCreateOptions[1], repoCreateOptions[2]].map(option => {
        return (
          <Menu.Item
            className={css.menuItem}
            key={option.type}
            text={<Text font={{ variation: FontVariation.BODY2 }}>{option.desc}</Text>}
            onClick={() => {
              setRepoOption(option)
              setTimeout(() => openModal(), 0)
            }}
          />
        )
      })}
    </SplitButton>
  )
}

interface RepoCreationOption {
  type: RepoCreationType
  title: string
  desc: string
}

interface BranchNameProps {
  currentBranchName: string
  onSelect: (branchName: string) => void
}

const BranchName: React.FC<BranchNameProps> = ({ currentBranchName, onSelect }) => {
  const { getString } = useStrings()
  const [customName, setCustomName] = useState(
    SUGGESTED_BRANCH_NAMES.includes(currentBranchName) ? '' : currentBranchName
  )
  const isCustomNameValid = useMemo(() => !customName || isGitBranchNameValid(customName), [customName])

  return (
    <Container padding="medium" width={250}>
      <Layout.Vertical spacing="small">
        <Menu>
          {SUGGESTED_BRANCH_NAMES.map(name => (
            <MenuItem
              key={name}
              text={name}
              labelElement={name === currentBranchName ? <BPIcon icon="small-tick" /> : undefined}
              disabled={name === currentBranchName}
              onClick={() => onSelect(name)}
            />
          ))}
          <MenuDivider className={css.divider} />
          <TextInput
            defaultValue={customName}
            autoFocus
            placeholder={getString('repos.enterBranchName')}
            onInput={e => setCustomName((e.currentTarget.value || '').trim())}
            errorText={isCustomNameValid ? undefined : getString('validation.gitBranchNameInvalid')}
            intent={!customName ? undefined : isCustomNameValid ? 'success' : 'danger'}
          />
        </Menu>
        <Container>
          <Layout.Horizontal>
            <Button
              type="submit"
              text={getString('ok')}
              intent={Intent.PRIMARY}
              className={Classes.POPOVER_DISMISS}
              disabled={!customName || !isCustomNameValid}
              onClick={() => onSelect(customName)}
            />
            <Button text={getString('cancel')} minimal className={Classes.POPOVER_DISMISS} />
          </Layout.Horizontal>
        </Container>
      </Layout.Vertical>
    </Container>
  )
}

export default NewRepoModalButton
