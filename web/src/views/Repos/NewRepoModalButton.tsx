/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
 */

import React, { useMemo, useState } from 'react'
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
import { useMutate } from 'restful-react'
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
import type { Repository } from 'types/Repository'
import { isGitBranchNameValid } from 'utils/GitUtils'
import licences from './licences'
import gitignores from './gitignores'
import css from './NewRepoModalButton.module.scss'

export interface NewRepoModalButtonProps extends Omit<ButtonProps, 'onClick' | 'onSubmit'> {
  space: string
  modalTitle: string
  submitButtonTitle?: string
  cancelButtonTitle?: string
  onSubmit: (data: Repository) => void
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
    const { mutate: createRepo, loading } = useMutate({
      verb: 'POST',
      path: '/api/v1/repos'
    })

    const handleSubmit = (formData?: Unknown): void => {
      try {
        // TODO: Backend is lacking support for these fields except
        // name and description (Oct 5)
        createRepo({
          pathName: get(formData, 'name', '').trim(),
          name: get(formData, 'name', '').trim(),
          spaceId: space,
          description: get(formData, 'description', '').trim(),
          isPublic: false,
          license: get(formData, 'license', 'none'),
          gitIgnore: get(formData, 'gitignore', 'none'),
          defaultBranch: get(formData, 'defaultBranch', 'main'),
          addReadme: get(formData, 'addReadme', false)
        })
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
              initialValues={{
                name: '',
                description: '',
                license: '',
                defaultBranch: 'main',
                gitignore: '',
                addReadme: false
              }}
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
                <Container margin={{ top: 'medium', bottom: 'xlarge' }}>
                  <Text>
                    Your repository will be initialized with a{' '}
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
                    </strong>{' '}
                    branch.
                  </Text>
                </Container>

                <FormInput.Select
                  name="license"
                  label={getString('addLicense')}
                  placeholder={getString('none')}
                  items={licences}
                  usePortal
                />

                <FormInput.Select
                  name="gitignore"
                  label={getString('addGitIgnore')}
                  placeholder={getString('none')}
                  items={gitignores}
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
