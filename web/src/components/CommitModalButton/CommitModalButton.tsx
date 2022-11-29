/*
 * Copyright 2021 Harness Inc. All rights reserved.
 * Use of this source code is governed by the PolyForm Shield 1.0.0 license
 * that can be found in the licenses directory at the root of this repository, also available at
 * https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.
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
  Icon,
  Formik,
  FormikForm,
  Heading,
  useToaster,
  FormInput,
  ButtonVariation
} from '@harness/uicore'
import cx from 'classnames'
import { FontVariation } from '@harness/design-system'
import { useMutate } from 'restful-react'
import { get } from 'lodash-es'
import { useModalHook } from '@harness/use-modal'
import { String, useStrings } from 'framework/strings'
import { getErrorMessage } from 'utils/Utils'
import type { OpenapiCommitFilesRequest, RepoCommitFilesResponse } from 'services/code'
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

interface CommitModalButtonProps extends Omit<ButtonProps, 'onClick' | 'onSubmit'>, Pick<GitInfoProps, 'repoMetadata'> {
  commitAction: GitCommitAction
  gitRef: string
  resourcePath: string
  commitTitlePlaceHolder: string
  oldResourcePath?: string
  payload?: string
  onSuccess: (data: RepoCommitFilesResponse, newBranch?: string) => void
}

export const CommitModalButton: React.FC<CommitModalButtonProps> = ({
  repoMetadata,
  commitAction,
  gitRef,
  resourcePath,
  commitTitlePlaceHolder,
  oldResourcePath,
  payload = '',
  onSuccess,
  ...props
}) => {
  const ModalComponent: React.FC = () => {
    const { getString } = useStrings()
    const [targetBranchOption, setTargetBranchOption] = useState(CommitToGitRefOption.DIRECTLY)
    const { showError, showSuccess } = useToaster()
    const { mutate, loading } = useMutate<RepoCommitFilesResponse>({
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
              payload: `${oldResourcePath ? `file://${resourcePath}\n` : ''}${payload}`
              // encoding: 'base64',
              // payload: window.btoa(payload || '')
            }
          ],
          branch: gitRef,
          newBranch: formData.newBranch,
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
                    label=""
                    onChange={e => {
                      setTargetBranchOption(get(e.target, 'defaultValue'))
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

                  {loading && <Icon intent={Intent.PRIMARY} name="spinner" size={16} />}
                </Layout.Horizontal>
              </FormikForm>
            </Formik>
          </Container>
        </Layout.Vertical>
      </Dialog>
    )
  }

  const [openModal, hideModal] = useModalHook(ModalComponent, [onSuccess, gitRef, resourcePath, commitTitlePlaceHolder])

  return <Button onClick={openModal} {...props} />
}
