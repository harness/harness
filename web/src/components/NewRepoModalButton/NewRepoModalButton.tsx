/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React, { useEffect, useMemo, useState } from 'react'
import { Icon as BPIcon, Classes, Dialog, Intent, Menu, MenuDivider, MenuItem } from '@blueprintjs/core'
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
  FormInput,
  Text,
  ButtonVariation,
  ButtonSize,
  TextInput
} from '@harness/uicore'
import { FontVariation } from '@harness/design-system'
import { useGet, useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { useModalHook } from '@harness/use-modal'
import { useStrings } from 'framework/strings'
import {
  DEFAULT_BRANCH_NAME,
  getErrorMessage,
  REGEX_VALID_REPO_NAME,
  SUGGESTED_BRANCH_NAMES,
  Unknown
} from 'utils/Utils'
import { isGitBranchNameValid } from 'utils/GitUtils'
import type { TypesRepository, OpenapiCreateRepositoryRequest } from 'services/scm'
import { useAppContext } from 'AppContext'
import css from './NewRepoModalButton.module.scss'

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
    const { standalone } = useAppContext()
    const { getString } = useStrings()
    const [branchName, setBranchName] = useState(DEFAULT_BRANCH_NAME)
    const { showError } = useToaster()
    const { mutate: createRepo, loading: submitLoading } = useMutate<TypesRepository>({
      verb: 'POST',
      path: `/api/v1/repos?spacePath=${space}`
    })
    const {
      data: gitignores,
      loading: gitIgnoreLoading,
      error: gitIgnoreError
    } = useGet({
      path: '/api/v1/resources/gitignore'
    })
    const {
      data: licences,
      loading: licenseLoading,
      error: licenseError
    } = useGet({
      path: '/api/v1/resources/license'
    })
    const loading = submitLoading || gitIgnoreLoading || licenseLoading

    useEffect(() => {
      if (gitIgnoreError || licenseError) {
        showError(getErrorMessage(gitIgnoreError || licenseError), 0)
      }
    }, [gitIgnoreError, licenseError, showError])

    const handleSubmit = (formData?: Unknown): void => {
      try {
        createRepo({
          defaultBranch: branchName || get(formData, 'defaultBranch', DEFAULT_BRANCH_NAME),
          description: get(formData, 'description', '').trim(),
          gitIgnore: get(formData, 'gitignore', 'none'),
          isPublic: get(formData, 'isPublic') === RepoVisibility.PUBLIC,
          license: get(formData, 'license', 'none'),
          name: get(formData, 'name', '').trim(),
          pathName: get(formData, 'name', '').trim(),
          readme: get(formData, 'addReadme', false),
          spaceId: standalone ? space : 0
        } as OpenapiCreateRepositoryRequest)
          .then(response => {
            hideModal()
            onSubmit(response)
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, 'failedToCreateRepo')
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, 'failedToCreateRepo')
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
                  .matches(REGEX_VALID_REPO_NAME, getString('validation.namePatternIsNotValid'))
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
