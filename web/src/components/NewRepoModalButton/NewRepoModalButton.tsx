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
import { FontVariation } from '@harnessio/design-system'
import { useGet, useMutate } from 'restful-react'
import { get } from 'lodash-es'
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
  ImportFormData,
  RepoCreationType,
  RepoFormData,
  RepoVisibility,
  isGitBranchNameValid,
  parseUrl
} from 'utils/GitUtils'
import type { TypesRepository, OpenapiCreateRepositoryRequest } from 'services/code'
import { useAppContext } from 'AppContext'
import ImportForm from './ImportForm/ImportForm'
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
  onSubmit: (data: TypesRepository) => void
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
    const { showError } = useToaster()

    const { mutate: createRepo, loading: submitLoading } = useMutate<TypesRepository>({
      verb: 'POST',
      path: `/api/v1/repos`,
      queryParams: standalone
        ? undefined
        : {
            space_path: space
          }
    })
    const { mutate: importRepo, loading: importRepoLoading } = useMutate<TypesRepository>({
      verb: 'POST',
      path: `/api/v1/repos/import`,
      queryParams: standalone
        ? undefined
        : {
            space_path: space
          }
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
    const loading = submitLoading || gitIgnoreLoading || licenseLoading || importRepoLoading

    useEffect(() => {
      if (gitIgnoreError || licenseError) {
        showError(getErrorMessage(gitIgnoreError || licenseError), 0)
      }
    }, [gitIgnoreError, licenseError, showError])
    const handleSubmit = (formData: RepoFormData) => {
      try {
        const payload: OpenapiCreateRepositoryRequest = {
          default_branch: branchName || get(formData, 'defaultBranch', DEFAULT_BRANCH_NAME),
          description: get(formData, 'description', '').trim(),
          git_ignore: get(formData, 'gitignore', 'none'),
          is_public: get(formData, 'isPublic') === RepoVisibility.PUBLIC,
          license: get(formData, 'license', 'none'),
          uid: get(formData, 'name', '').trim(),
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
      const provider = parseUrl(formData.repoUrl)
      const importPayload = {
        description: formData.description || '',
        parent_ref: space,
        uid: formData.name,
        provider: { type: provider?.provider.toLowerCase(), username: formData.username, password: formData.password },
        provider_repo: provider?.fullRepo
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
            {repoOption.type === RepoCreationType.IMPORT ? getString('importRepo.title') : modalTitle}
          </Heading>

          <Container margin={{ right: 'xxlarge' }}>
            {repoOption.type === RepoCreationType.IMPORT ? (
              <ImportForm hideModal={hideModal} handleSubmit={handleImportSubmit} loading={false} />
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
                                <Icon name="git-clone-step" size={20} margin={{ right: 'medium' }} />
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
                  <Layout.Horizontal
                    spacing="small"
                    padding={{ right: 'xxlarge', top: 'xxxlarge', bottom: 'large' }}
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
      desc: getString('createNewRepo')
    },
    {
      type: RepoCreationType.IMPORT,
      title: getString('importGitRepo'),
      desc: getString('importGitRepo')
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
  return (
    <SplitButton
      {...props}
      loading={false}
      text={repoOption.title}
      variation={ButtonVariation.PRIMARY}
      popoverProps={{
        interactionKind: 'click',
        usePortal: true,
        popoverClassName: css.popover,
        position: PopoverPosition.BOTTOM_RIGHT,
        transitionDuration: 1000
      }}
      icon={repoOption.type === RepoCreationType.IMPORT ? undefined : 'plus'}
      {...permissionProps(permResult, standalone)}
      onClick={() => {
        openModal()
      }}>
      {repoCreateOptions.map(option => {
        return (
          <Menu.Item
            key={option.type}
            className={css.menuItem}
            text={
              <>
                <p>{option.desc}</p>
              </>
            }
            onClick={() => {
              setRepoOption(option)
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
