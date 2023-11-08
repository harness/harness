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
import { Dialog, Intent } from '@blueprintjs/core'
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
  ButtonVariation
} from '@harnessio/uicore'
import { Icon } from '@harnessio/icons'
import cx from 'classnames'
import { FontVariation } from '@harnessio/design-system'
import { useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { useModalHook } from 'hooks/useModalHook'
import { String, useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import type { OpenapiCommitFilesRequest, TypesListCommitResponse } from 'services/code'
import { GitCommitAction, GitInfoProps, isGitBranchNameValid } from 'utils/GitUtils'
import css from './CommitModalButton.module.scss'

enum CommitToGitRefOption {
  DIRECTLY = 'directly',
  NEW_BRANCH = 'new-branch'
}

interface FormData {
  title?: string
  message?: string
  branch?: string
  newBranch?: string
}

interface CommitModalProps extends Pick<GitInfoProps, 'repoMetadata'> {
  commitAction: GitCommitAction
  gitRef: string
  resourcePath: string
  commitTitlePlaceHolder: string
  disableBranchCreation?: boolean
  oldResourcePath?: string
  payload?: string
  sha?: string
  onSuccess: (data: TypesListCommitResponse, newBranch?: string) => void
}

export function useCommitModal({
  repoMetadata,
  commitAction,
  gitRef,
  resourcePath,
  commitTitlePlaceHolder,
  oldResourcePath,
  disableBranchCreation = false,
  payload = '',
  sha,
  onSuccess
}: CommitModalProps) {
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const [targetBranchOption, setTargetBranchOption] = useState(CommitToGitRefOption.DIRECTLY)
    const { showError, showSuccess } = useToaster()
    const { mutate, loading } = useMutate<TypesListCommitResponse>({
      verb: 'POST',
      path: `/api/v1/repos/${repoMetadata.path}/+/commits`
    })

    const handleSubmit = (formData: FormData) => {
      try {
        const data: OpenapiCommitFilesRequest = {
          actions: [
            {
              action: commitAction,
              path: oldResourcePath || resourcePath,
              payload: `${oldResourcePath ? `file://${resourcePath}\n` : ''}${payload}`,
              sha
              // encoding: 'base64',
              // payload: window.btoa(payload || '')
            }
          ],
          branch: gitRef,
          new_branch: formData.newBranch,
          title: formData.title || commitTitlePlaceHolder,
          message: formData.message
        }

        mutate(data)
          .then(response => {
            hideModal()
            onSuccess(response, formData.newBranch)

            if (commitAction === GitCommitAction.DELETE) {
              showSuccess(getString('fileDeleted').replace('__path__', resourcePath))
            }
          })
          .catch(_error => {
            showError(getErrorMessage(_error), 0, getString('failedToCreateRepo'))
          })
      } catch (exception) {
        showError(getErrorMessage(exception), 0, getString('failedToCreateRepo'))
      }
    }

    return (
      <Dialog
        isOpen
        enforceFocus={false}
        onClose={hideModal}
        title={''}
        style={{ width: 700, maxHeight: '95vh', overflow: 'auto' }}>
        <Layout.Vertical className={cx(css.main)}>
          <Heading level={3} font={{ variation: FontVariation.H3 }} margin={{ bottom: 'xlarge' }}>
            {getString('commitChanges')}
          </Heading>

          <Container margin={{ right: 'xxlarge' }}>
            <Formik<FormData>
              initialValues={{
                title: '',
                message: '',
                branch: CommitToGitRefOption.DIRECTLY,
                newBranch: ''
              }}
              formName="commitChanges"
              enableReinitialize={true}
              validationSchema={yup.object().shape({
                newBranch: yup
                  .string()
                  .trim()
                  .test('valid-branch-name', getString('validation.gitBranchNameInvalid'), value => {
                    if (targetBranchOption === CommitToGitRefOption.NEW_BRANCH) {
                      const val = value || ''
                      return !!val && isGitBranchNameValid(val)
                    }
                    return true
                  })
              })}
              validateOnChange
              validateOnBlur
              onSubmit={handleSubmit}>
              <FormikForm>
                <FormInput.Text
                  name="title"
                  label={getString('commitMessage')}
                  placeholder={commitTitlePlaceHolder}
                  tooltipProps={{
                    dataTooltipId: 'commitMessage'
                  }}
                  inputGroup={{ autoFocus: true }}
                />
                <FormInput.TextArea
                  className={css.extendedDescription}
                  name="message"
                  placeholder={getString('optionalExtendedDescription')}
                />
                <Container
                  className={cx(
                    css.radioGroup,
                    targetBranchOption === CommitToGitRefOption.DIRECTLY ? css.directly : css.newBranch
                  )}>
                  <FormInput.RadioGroup
                    name="branch"
                    disabled={disableBranchCreation}
                    label=""
                    onChange={e => {
                      setTargetBranchOption(get(e.target, 'defaultValue') as unknown as CommitToGitRefOption)
                    }}
                    items={[
                      {
                        label: <String stringID="commitDirectlyTo" vars={{ gitRef }} useRichText />,
                        value: CommitToGitRefOption.DIRECTLY
                      },
                      {
                        label: <String stringID="commitToNewBranch" useRichText />,
                        value: CommitToGitRefOption.NEW_BRANCH
                      }
                    ]}
                  />
                  {targetBranchOption === CommitToGitRefOption.NEW_BRANCH && (
                    <Container>
                      <Layout.Horizontal spacing="medium" className={css.newBranchContainer}>
                        <Icon name="git-branch" />
                        <FormInput.Text
                          name="newBranch"
                          placeholder={getString('enterNewBranchName')}
                          tooltipProps={{
                            dataTooltipId: 'enterNewBranchName'
                          }}
                          inputGroup={{ autoFocus: true }}
                        />
                      </Layout.Horizontal>
                    </Container>
                  )}
                </Container>

                <Layout.Horizontal spacing="small" padding={{ right: 'xxlarge', top: 'xxlarge', bottom: 'large' }}>
                  <Button
                    type="submit"
                    variation={ButtonVariation.PRIMARY}
                    text={getString('commit')}
                    disabled={loading}
                  />
                  <Button text={getString('cancel')} variation={ButtonVariation.LINK} onClick={hideModal} />
                  <FlexExpander />

                  {loading && <Icon intent={Intent.PRIMARY} name="steps-spinner" size={16} />}
                </Layout.Horizontal>
              </FormikForm>
            </Formik>
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }

  const [openModal, hideModal] = useModalHook(ModalComponent, [onSuccess, gitRef, resourcePath, commitTitlePlaceHolder])

  return [openModal, hideModal]
}

interface CommitModalButtonProps extends Omit<ButtonProps, 'onClick' | 'onSubmit'>, Pick<GitInfoProps, 'repoMetadata'> {
  commitAction: GitCommitAction
  gitRef: string
  resourcePath: string
  commitTitlePlaceHolder: string
  disableBranchCreation?: boolean
  oldResourcePath?: string
  payload?: string
  sha?: string
  onSuccess: (data: TypesListCommitResponse, newBranch?: string) => void
}

export const CommitModalButton: React.FC<CommitModalButtonProps> = ({
  repoMetadata,
  commitAction,
  gitRef,
  resourcePath,
  commitTitlePlaceHolder,
  oldResourcePath,
  disableBranchCreation = false,
  payload = '',
  sha,
  onSuccess,
  ...props
}) => {
  const [openModal] = useCommitModal({
    repoMetadata,
    commitAction,
    gitRef,
    resourcePath,
    commitTitlePlaceHolder,
    oldResourcePath,
    disableBranchCreation,
    payload,
    sha,
    onSuccess
  })

  return <Button onClick={openModal} {...props} />
}
